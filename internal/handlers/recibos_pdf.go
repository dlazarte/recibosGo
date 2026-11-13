package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"recibosgo/internal/models"
	"recibosgo/internal/pdf"
)

func (a *App) ReciboPDF(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}
	var recibo models.Recibo
	if a.DB.Preload("Departamento").Preload("Departamento.Dueño").Preload("Titular").Preload("Detalles.Servicio").First(&recibo, id).Error != nil {
		http.Error(w, "recibo no encontrado", http.StatusNotFound)
		return
	}
	data, err := pdf.Recibo(recibo)
	if err != nil {
		http.Error(w, "error generando PDF: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.pdf", recibo.NumeroRecibo))
	w.Write(data)
}
