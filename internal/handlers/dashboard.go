package handlers

import (
	"net/http"

	"recibosgo/internal/middleware"
	"recibosgo/internal/models"
)

type DashboardData struct {
	DeptCount    int64
	ServCount    int64
	RecibosCount int64
	Recibos      []models.Recibo
	Departamentos []models.Departamento
}

func (a *App) Dashboard(w http.ResponseWriter, r *http.Request) {
	user, _ := middleware.UserFromContext(r.Context())

	data := DashboardData{}
	a.DB.Model(&models.Recibo{}).Count(&data.RecibosCount)

	var recibos []models.Recibo
	q := a.DB.Preload("Departamento").Preload("Detalles").Order("fecha DESC").Limit(10)

	if user.Role == "administrador" || user.Role == "receptor" {
		a.DB.Model(&models.Departamento{}).Where("activo = ?", true).Count(&data.DeptCount)
		a.DB.Model(&models.Servicio{}).Where("activo = ?", true).Count(&data.ServCount)
		q.Find(&recibos)
	} else {
		// Find departments via persona (v2.1)
		var persona models.Persona
		if a.DB.Where("email = ?", user.Email).First(&persona).Error == nil {
			var depts []models.Departamento
			a.DB.Where("id_dueño = ? OR id_inquilino = ?", persona.ID, persona.ID).Find(&depts)
			data.Departamentos = depts
		}
		ids := make([]uint, 0, len(data.Departamentos))
		for _, d := range data.Departamentos {
			ids = append(ids, d.ID)
		}
		if len(ids) > 0 {
			q.Where("id_departamento IN ?", ids).Find(&recibos)
		}
	}

	data.Recibos = recibos
	a.render(w, r, "dashboard", PageData{Data: data})
}
