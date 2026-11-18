package handlers

import (
	"net/http"

	"recibosgo/internal/middleware"
	"recibosgo/internal/models"
)

type DashboardData struct {
	DeptCount        int64
	ServCount        int64
	RecibosCount     int64
	Recibos          []models.Recibo
	Departamentos    []models.Departamento
	DeudaPendiente   float64
	UnidadesConDeuda int64
	DeudasAnteriores []models.Recibo
	Registros        []models.Registro
	RegistrosPend    int64
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

		var pendientes []models.Recibo
		a.DB.Preload("Departamento").Preload("Titular").
			Where("id NOT IN (SELECT DISTINCT id_recibo FROM detallerecibo WHERE forma_pago <> '')").
			Order("fecha ASC").Find(&pendientes)
		unidades := map[uint]bool{}
		for _, r := range pendientes {
			data.DeudaPendiente += r.MontoPagar()
			unidades[r.IDDepartamento] = true
			if r.IDPersona != nil && *r.IDPersona != r.Departamento.IDDueño && len(data.DeudasAnteriores) < 10 {
				data.DeudasAnteriores = append(data.DeudasAnteriores, r)
			}
		}
		data.UnidadesConDeuda = int64(len(unidades))

		a.DB.Order("creado_en DESC").Limit(10).Find(&data.Registros)
		a.DB.Model(&models.Registro{}).Where("estado = 'pendiente'").Count(&data.RegistrosPend)
	} else {
		persona, ok := a.personaDelUsuario(user)
		if ok {
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
