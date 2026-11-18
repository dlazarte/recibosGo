package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"recibosgo/internal/models"
	"recibosgo/internal/parser"
)

const minPasswordLen = 8

// RegistroPage muestra el formulario público de auto-registro.
func (a *App) RegistroPage(w http.ResponseWriter, r *http.Request) {
	var depts []models.Departamento
	a.DB.Where("activo = ?", true).Order(ordenUnidades).Find(&depts)

	data := PageData{Data: depts}
	if r.URL.Query().Get("pendiente") == "1" {
		data.Msg = "Tu solicitud fue enviada al administrador para su aprobación."
	}
	if r.URL.Query().Get("error") == "1" {
		data.Err = "No se pudo completar el registro. Revisá los datos o contactá al administrador."
	}
	a.render(w, r, "registro", data)
}

// RegistroCreate valida el formulario, matchea la persona y crea la cuenta
// automáticamente si hay coincidencia; si no, deja la solicitud pendiente.
func (a *App) RegistroCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/registro?error=1", http.StatusFound)
		return
	}
	email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	nombre := strings.TrimSpace(r.FormValue("nombre"))
	unidad := strings.TrimSpace(r.FormValue("unidad"))
	telefono := strings.TrimSpace(r.FormValue("telefono"))
	password := r.FormValue("password")

	if email == "" || nombre == "" || len(password) < minPasswordLen || password != r.FormValue("password2") {
		http.Redirect(w, r, "/registro?error=1", http.StatusFound)
		return
	}

	// No revelar si el email ya existe: mensaje genérico.
	var n int64
	a.DB.Model(&models.User{}).Where("LOWER(email) = ?", email).Count(&n)
	if n > 0 {
		http.Redirect(w, r, "/registro?error=1", http.StatusFound)
		return
	}
	a.DB.Model(&models.Registro{}).Where("LOWER(email) = ? AND estado = 'pendiente'", email).Count(&n)
	if n > 0 {
		http.Redirect(w, r, "/registro?error=1", http.StatusFound)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Redirect(w, r, "/registro?error=1", http.StatusFound)
		return
	}

	persona, rol, ok := a.matchRegistro(unidad, nombre, telefono)
	reg := models.Registro{
		Email: email, Nombre: nombre, Unidad: unidad, Telefono: telefono,
		Password: string(hash), IP: r.RemoteAddr, CreadoEn: time.Now(),
	}

	if ok {
		user := models.User{Email: email, Password: string(hash), Role: rol, IDPersona: &persona.ID}
		if err := a.DB.Create(&user).Error; err != nil {
			http.Redirect(w, r, "/registro?error=1", http.StatusFound)
			return
		}
		reg.Estado = "auto"
		reg.Rol = rol
		reg.IDPersona = &persona.ID
		a.DB.Create(&reg)
		a.completarPersona(persona.ID, email, telefono)
		http.Redirect(w, r, "/login?registro=1", http.StatusFound)
		return
	}

	reg.Estado = "pendiente"
	reg.Rol = "dueno" // sugerido; el admin puede cambiarlo al aprobar
	a.DB.Create(&reg)
	http.Redirect(w, r, "/registro?pendiente=1", http.StatusFound)
}

// RegistroAprobar crea la cuenta de una solicitud pendiente (solo admin).
func (a *App) RegistroAprobar(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}
	var reg models.Registro
	if a.DB.First(&reg, id).Error != nil {
		http.Error(w, "no encontrado", http.StatusNotFound)
		return
	}
	if reg.Estado != "pendiente" {
		http.Redirect(w, r, "/dashboard?msg="+urlEscape("La solicitud ya fue procesada"), http.StatusFound)
		return
	}
	rol := r.FormValue("rol")
	if rol != "dueno" && rol != "inquilino" {
		rol = reg.Rol
	}
	var idPersona *uint
	if reg.IDPersona != nil {
		idPersona = reg.IDPersona
	} else if p, _, ok := a.matchRegistro(reg.Unidad, reg.Nombre, reg.Telefono); ok {
		idPersona = &p.ID
	}
	user := models.User{Email: reg.Email, Password: reg.Password, Role: rol, IDPersona: idPersona}
	if err := a.DB.Create(&user).Error; err != nil {
		http.Redirect(w, r, "/dashboard?msg="+urlEscape("No se pudo aprobar: "+err.Error()), http.StatusFound)
		return
	}
	reg.Estado = "aprobado"
	reg.Rol = rol
	reg.IDPersona = idPersona
	a.DB.Save(&reg)
	if idPersona != nil {
		a.completarPersona(*idPersona, reg.Email, reg.Telefono)
	}
	http.Redirect(w, r, "/dashboard?msg="+urlEscape("Registro aprobado"), http.StatusFound)
}

// completarPersona actualiza email/teléfono de la persona si faltaban.
func (a *App) completarPersona(id uint, email, telefono string) {
	var p models.Persona
	if a.DB.First(&p, id).Error != nil {
		return
	}
	cambios := map[string]interface{}{}
	if p.Email == "" && email != "" {
		cambios["email"] = email
	}
	if p.Telefono == "" && telefono != "" {
		cambios["telefono"] = telefono
	}
	if len(cambios) > 0 {
		a.DB.Model(&p).Updates(cambios)
	}
}

// matchRegistro busca la persona por unidad+nombre o por teléfono registrado
// y devuelve el rol que le corresponde.
func (a *App) matchRegistro(unidad, nombre, telefono string) (models.Persona, string, bool) {
	nom := parser.NormalizeName(nombre)
	tel := normalizarTelefono(telefono)

	if unidad != "" {
		var d models.Departamento
		if a.DB.Preload("Dueño").Preload("Inquilino").
			Where("UPPER(TRIM(nombre)) = ?", strings.ToUpper(unidad)).First(&d).Error == nil {
			type candidato struct {
				persona models.Persona
				rol     string
			}
			var candidatos []candidato
			if d.Dueño.ID != 0 {
				candidatos = append(candidatos, candidato{d.Dueño, "dueno"})
			}
			if d.Inquilino != nil && d.Inquilino.ID != 0 {
				candidatos = append(candidatos, candidato{*d.Inquilino, "inquilino"})
			}
			for _, c := range candidatos {
				if nombresCoinciden(nom, parser.NormalizeName(c.persona.Nombre)) ||
					(tel != "" && normalizarTelefono(c.persona.Telefono) == tel) {
					return c.persona, c.rol, true
				}
			}
		}
	}

	if tel != "" {
		var personas []models.Persona
		a.DB.Where("telefono <> ''").Find(&personas)
		for _, p := range personas {
			if normalizarTelefono(p.Telefono) == tel {
				rol := "dueno"
				if p.Tipo == "inquilino" {
					rol = "inquilino"
				}
				return p, rol, true
			}
		}
	}
	return models.Persona{}, "", false
}

func nombresCoinciden(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	if a == b {
		return true
	}
	// Fallback por substring para nombres compuestos; requiere longitud mínima.
	if len(a) >= 4 && len(b) >= 4 {
		return strings.Contains(a, b) || strings.Contains(b, a)
	}
	return false
}

func normalizarTelefono(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	digits := b.String()
	if len(digits) > 10 {
		return digits[len(digits)-10:]
	}
	return digits
}
