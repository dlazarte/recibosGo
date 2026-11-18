package handlers

import (
	"net/http"
	"time"

	"recibosgo/internal/models"
)

type ReporteData struct {
	Mes         string
	Total       float64
	Cobrado     float64
	Pendiente   float64
	Recibos     []models.Recibo
	PorServicio map[string]float64
	Paginado    Pagination
}

func (a *App) Reportes(w http.ResponseWriter, r *http.Request) {
	mes := r.URL.Query().Get("mes")
	if mes == "" {
		mes = time.Now().Format("2006-01")
	}

	start, err := time.Parse("2006-01", mes)
	if err != nil {
		http.Error(w, "mes inválido", http.StatusBadRequest)
		return
	}
	end := start.AddDate(0, 1, 0)

	var total int64
	a.DB.Model(&models.Recibo{}).Where("fecha >= ? AND fecha < ?", start, end).Count(&total)

	var recibos []models.Recibo
	a.DB.Preload("Departamento").Preload("Detalles.Servicio").
		Where("fecha >= ? AND fecha < ?", start, end).
		Order("fecha ASC, id ASC").
		Offset((paginaActual(r) - 1) * perPage).
		Limit(perPage).
		Find(&recibos)

	data := ReporteData{Mes: mes, Recibos: recibos, PorServicio: map[string]float64{}}

	a.DB.Model(&models.Recibo{}).
		Where("fecha >= ? AND fecha < ?", start, end).
		Select("COALESCE(SUM(total),0)").
		Scan(&data.Total)

	a.DB.Model(&models.DetalleRecibo{}).
		Joins("JOIN recibo ON recibo.id = detallerecibo.id_recibo").
		Where("recibo.fecha >= ? AND recibo.fecha < ? AND detallerecibo.forma_pago <> ''", start, end).
		Select("COALESCE(SUM(detallerecibo.precio*detallerecibo.cantidad),0)").
		Scan(&data.Cobrado)
	data.Pendiente = data.Total - data.Cobrado

	type servicioRow struct {
		Nombre  string
		Importe float64
	}
	var rows []servicioRow
	a.DB.Model(&models.DetalleRecibo{}).
		Joins("JOIN recibo ON recibo.id = detallerecibo.id_recibo").
		Joins("JOIN servicio ON servicio.id = detallerecibo.id_servicio").
		Where("recibo.fecha >= ? AND recibo.fecha < ?", start, end).
		Select("servicio.nombre AS nombre, SUM(detallerecibo.precio*detallerecibo.cantidad) AS importe").
		Group("servicio.nombre").
		Scan(&rows)
	for _, r := range rows {
		data.PorServicio[r.Nombre] = r.Importe
	}

	data.Paginado = paginar(r, total, "#reportes-content")

	if r.Header.Get("HX-Request") == "true" {
		a.renderFrag(w, "reportes_content", PageData{Data: data})
		return
	}
	a.render(w, r, "reportes", PageData{Data: data})
}
