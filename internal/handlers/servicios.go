package handlers

import (
	"net/http"
	"strconv"

	"recibosgo/internal/models"
)

type ServData struct {
	Servicios []models.Servicio
	Edit      *models.Servicio
	Msg       string
}

func (a *App) ServiciosList(w http.ResponseWriter, r *http.Request) {
	var servicios []models.Servicio
	a.DB.Order("nombre ASC").Find(&servicios)

	data := ServData{Servicios: servicios, Msg: r.URL.Query().Get("msg")}
	if editID, err := strconv.Atoi(r.URL.Query().Get("edit")); err == nil {
		var s models.Servicio
		if a.DB.First(&s, editID).Error == nil {
			data.Edit = &s
		}
	}
	a.render(w, r, "servicios", PageData{Data: data})
}

func (a *App) ServicioCreate(w http.ResponseWriter, r *http.Request) {
	precio, _ := strconv.ParseFloat(r.FormValue("precio_unitario"), 64)
	s := models.Servicio{
		Nombre:         r.FormValue("nombre"),
		PrecioUnitario: precio,
		Activo:         r.FormValue("activo") == "on" || r.FormValue("activo") == "1",
	}
	if s.Nombre == "" {
		http.Redirect(w, r, "/servicios?msg="+urlEscape("El nombre es obligatorio"), http.StatusFound)
		return
	}
	if err := a.DB.Create(&s).Error; err != nil {
		http.Redirect(w, r, "/servicios?msg="+urlEscape("Error al crear: "+err.Error()), http.StatusFound)
		return
	}
	http.Redirect(w, r, "/servicios?msg="+urlEscape("Servicio creado"), http.StatusFound)
}

func (a *App) ServicioUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}
	var s models.Servicio
	if a.DB.First(&s, id).Error != nil {
		http.Error(w, "no encontrado", http.StatusNotFound)
		return
	}
	s.Nombre = r.FormValue("nombre")
	s.PrecioUnitario, _ = strconv.ParseFloat(r.FormValue("precio_unitario"), 64)
	s.Activo = r.FormValue("activo") == "on" || r.FormValue("activo") == "1"
	if err := a.DB.Save(&s).Error; err != nil {
		http.Redirect(w, r, "/servicios?msg="+urlEscape("Error al actualizar: "+err.Error()), http.StatusFound)
		return
	}
	http.Redirect(w, r, "/servicios?msg="+urlEscape("Servicio actualizado"), http.StatusFound)
}
