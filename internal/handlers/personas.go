package handlers

import (
	"net/http"
	"strconv"

	"recibosgo/internal/models"
)

type PersonaData struct {
	Personas []models.Persona
	Edit     *models.Persona
	Search   string
	Msg      string
}

func (a *App) PersonasList(w http.ResponseWriter, r *http.Request) {
	var personas []models.Persona
	query := a.DB.Order("nombre ASC")

	q := r.URL.Query().Get("q")
	if q != "" {
		like := "%" + q + "%"
		query = query.Where("nombre LIKE ? OR telefono LIKE ? OR email LIKE ?", like, like, like)
	}

	query.Find(&personas)

	data := PersonaData{Personas: personas, Search: q, Msg: r.URL.Query().Get("msg")}
	if editID, err := strconv.Atoi(r.URL.Query().Get("edit")); err == nil {
		var p models.Persona
		if a.DB.First(&p, editID).Error == nil {
			data.Edit = &p
		}
	}

	if r.Header.Get("HX-Request") == "true" {
		a.renderFrag(w, "personas_table", data)
		return
	}
	a.render(w, r, "titulares", PageData{Data: data})
}

func (a *App) PersonaCreate(w http.ResponseWriter, r *http.Request) {
	p := models.Persona{
		Nombre:   r.FormValue("nombre"),
		Telefono: r.FormValue("telefono"),
		Email:    r.FormValue("email"),
		Tipo:     r.FormValue("tipo"),
		Activo:   r.FormValue("activo") == "on" || r.FormValue("activo") == "1",
	}
	if p.Nombre == "" || p.Tipo == "" {
		http.Redirect(w, r, "/personas?msg="+urlEscape("Nombre y tipo son obligatorios"), http.StatusFound)
		return
	}
	if err := a.DB.Create(&p).Error; err != nil {
		http.Redirect(w, r, "/personas?msg="+urlEscape("Error al crear: "+err.Error()), http.StatusFound)
		return
	}
	http.Redirect(w, r, "/personas?msg="+urlEscape("Persona creada"), http.StatusFound)
}

func (a *App) PersonaUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}
	var p models.Persona
	if a.DB.First(&p, id).Error != nil {
		http.Error(w, "no encontrado", http.StatusNotFound)
		return
	}
	p.Nombre = r.FormValue("nombre")
	p.Telefono = r.FormValue("telefono")
	p.Email = r.FormValue("email")
	p.Tipo = r.FormValue("tipo")
	p.Activo = r.FormValue("activo") == "on" || r.FormValue("activo") == "1"
	if err := a.DB.Save(&p).Error; err != nil {
		http.Redirect(w, r, "/personas?msg="+urlEscape("Error al actualizar: "+err.Error()), http.StatusFound)
		return
	}
	http.Redirect(w, r, "/personas?msg="+urlEscape("Persona actualizada"), http.StatusFound)
}
