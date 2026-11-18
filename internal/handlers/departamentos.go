package handlers

import (
	"net/http"
	"sort"
	"strconv"

	"recibosgo/internal/models"
)

type HistorialGrupo struct {
	Persona   *models.Persona
	Recibos   []models.Recibo
	Total     float64
	Pagado    float64
	Pendiente float64
}

type HistorialData struct {
	Departamento models.Departamento
	Grupos       []HistorialGrupo
}

type DeptData struct {
	Departamentos   []models.Departamento
	Personas        []models.Persona
	Inquilinos      []models.Persona
	Search          string
	Edit            *models.Departamento
	EditInquilinoID uint
	Msg             string
	Err             string
	Paginado        Pagination
}

func (a *App) DepartamentosList(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("q")
	data := DeptData{Search: search}

	var total int64
	q := a.DB.Model(&models.Departamento{})
	if search != "" {
		like := "%" + search + "%"
		q = q.Where("nombre LIKE ? OR propietario LIKE ?", like, like)
	}
	q.Count(&total)

	var depts []models.Departamento
	q.Preload("Dueño").Preload("Inquilino").
		Order(ordenUnidades).
		Offset((paginaActual(r) - 1) * perPage).
		Limit(perPage).
		Find(&depts)
	data.Departamentos = depts

	// Load personas for the dueño / inquilino selects
	var personas []models.Persona
	a.DB.Where("tipo = ? AND activo = true", "propietario").Order("nombre ASC").Find(&personas)
	data.Personas = personas
	var inquilinos []models.Persona
	a.DB.Where("tipo = ? AND activo = true", "inquilino").Order("nombre ASC").Find(&inquilinos)
	data.Inquilinos = inquilinos

	if editID, err := strconv.Atoi(r.URL.Query().Get("edit")); err == nil {
		var d models.Departamento
		if a.DB.Preload("Dueño").Preload("Inquilino").First(&d, editID).Error == nil {
			data.Edit = &d
			if d.IDInquilino != nil {
				data.EditInquilinoID = *d.IDInquilino
			}
		}
	}
	data.Msg = r.URL.Query().Get("msg")
	data.Paginado = paginar(r, total, "#deptos-table")

	if r.Header.Get("HX-Request") == "true" {
		a.renderFrag(w, "deptos_table", PageData{Data: data})
		return
	}
	a.render(w, r, "departamentos", PageData{Data: data})
}

// parsePersonaID convierte el valor de un select en *uint (nil = sin selección).
func parsePersonaID(v string) *uint {
	if v == "" {
		return nil
	}
	id, err := strconv.ParseUint(v, 10, 32)
	if err != nil || id == 0 {
		return nil
	}
	u := uint(id)
	return &u
}

func (a *App) DepartamentoCreate(w http.ResponseWriter, r *http.Request) {
	nombre := r.FormValue("nombre")
	if nombre == "" {
		http.Redirect(w, r, "/departamentos?msg="+urlEscape("El nombre es obligatorio"), http.StatusFound)
		return
	}
	var piso *string
	if v := r.FormValue("piso"); v != "" {
		piso = &v
	}
	var idDueño *uint
	if v := r.FormValue("id_dueño"); v != "" {
		if id, err := strconv.ParseUint(v, 10, 32); err == nil {
			u := uint(id)
			idDueño = &u
		}
	}
	d := models.Departamento{
		Nombre:      nombre,
		Numero:      r.FormValue("numero"),
		Piso:        piso,
		Propietario: r.FormValue("propietario"),
		Telefono:    r.FormValue("telefono"),
		Email:       r.FormValue("email"),
		Activo:      r.FormValue("activo") == "on" || r.FormValue("activo") == "1",
	}
	d.Coeficiente, _ = strconv.ParseFloat(r.FormValue("coeficiente"), 64)
	if idDueño != nil {
		d.IDDueño = *idDueño
	}
	d.IDInquilino = parsePersonaID(r.FormValue("id_inquilino"))
	if d.Propietario == "" {
		d.Propietario = nombre
	}
	if err := a.DB.Create(&d).Error; err != nil {
		http.Redirect(w, r, "/departamentos?msg="+urlEscape("Error al crear: "+err.Error()), http.StatusFound)
		return
	}
	http.Redirect(w, r, "/departamentos?msg="+urlEscape("Departamento creado"), http.StatusFound)
}

func (a *App) DepartamentoUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}
	var d models.Departamento
	if a.DB.First(&d, id).Error != nil {
		http.Error(w, "no encontrado", http.StatusNotFound)
		return
	}
	d.Nombre = r.FormValue("nombre")
	d.Numero = r.FormValue("numero")
	var piso *string
	if v := r.FormValue("piso"); v != "" {
		piso = &v
	}
	d.Piso = piso
	d.Propietario = r.FormValue("propietario")
	d.Telefono = r.FormValue("telefono")
	d.Email = r.FormValue("email")
	d.Activo = r.FormValue("activo") == "on" || r.FormValue("activo") == "1"
	d.Coeficiente, _ = strconv.ParseFloat(r.FormValue("coeficiente"), 64)
	if v := r.FormValue("id_dueño"); v != "" {
		if id, err := strconv.ParseUint(v, 10, 32); err == nil {
			d.IDDueño = uint(id)
		}
	}
	d.IDInquilino = parsePersonaID(r.FormValue("id_inquilino"))
	if err := a.DB.Save(&d).Error; err != nil {
		http.Redirect(w, r, "/departamentos?msg="+urlEscape("Error al actualizar: "+err.Error()), http.StatusFound)
		return
	}
	http.Redirect(w, r, "/departamentos?msg="+urlEscape("Departamento actualizado"), http.StatusFound)
}

// DepartamentoHistorial muestra los recibos de una unidad agrupados por deudor.
func (a *App) DepartamentoHistorial(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}
	var d models.Departamento
	if a.DB.Preload("Dueño").Preload("Inquilino").First(&d, id).Error != nil {
		http.Error(w, "no encontrado", http.StatusNotFound)
		return
	}
	var recibos []models.Recibo
	a.DB.Preload("Titular").Preload("Detalles.Servicio").
		Where("id_departamento = ?", id).
		Order("periodo DESC, id DESC").
		Find(&recibos)

	grupos := map[uint]*HistorialGrupo{}
	var orden []uint
	for _, rec := range recibos {
		key := uint(0)
		if rec.IDPersona != nil {
			key = *rec.IDPersona
		}
		g, ok := grupos[key]
		if !ok {
			g = &HistorialGrupo{Persona: rec.Titular}
			grupos[key] = g
			orden = append(orden, key)
		}
		g.Recibos = append(g.Recibos, rec)
		g.Total += rec.Total
		if rec.Pagado() {
			g.Pagado += rec.MontoPagar()
		} else {
			g.Pendiente += rec.MontoPagar()
		}
	}
	sort.Slice(orden, func(i, j int) bool {
		ga, gb := grupos[orden[i]], grupos[orden[j]]
		if ga.Persona == nil {
			return false
		}
		if gb.Persona == nil {
			return true
		}
		return ga.Persona.Nombre < gb.Persona.Nombre
	})

	data := HistorialData{Departamento: d}
	for _, key := range orden {
		data.Grupos = append(data.Grupos, *grupos[key])
	}
	a.render(w, r, "departamento_historial", PageData{Data: data})
}
