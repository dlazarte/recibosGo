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

type UnidadCuenta struct {
	Departamento models.Departamento
	Rol          string
	Pendiente    float64
	Historica    bool
}

type PersonaCuentaData struct {
	Persona         models.Persona
	Unidades        []UnidadCuenta
	Recibos         []models.Recibo
	FiltroEstado    string
	TotalPendiente  float64
	TotalPagado     float64
	TotalSaldoFavor float64
}

// PersonaCuenta muestra la cuenta corriente de una persona: unidades, recibos
// imputados a ella, detalle de cada recibo e importes.
func (a *App) PersonaCuenta(w http.ResponseWriter, r *http.Request) {
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

	estado := r.URL.Query().Get("estado")
	if estado == "" {
		estado = "pendiente"
	}

	q := a.DB.Model(&models.Recibo{}).Where("id_persona = ?", p.ID)
	if estado == "pendiente" {
		q = q.Where("id NOT IN (SELECT DISTINCT id_recibo FROM detallerecibo WHERE forma_pago <> '')")
	}
	var recibos []models.Recibo
	q.Preload("Departamento").Preload("Detalles.Servicio").
		Order("periodo DESC, fecha DESC, id DESC").Find(&recibos)

	// Totales sobre todos los recibos del deudor (independientes del filtro).
	var todos []models.Recibo
	a.DB.Preload("Detalles").Where("id_persona = ?", p.ID).Find(&todos)
	data := PersonaCuentaData{Persona: p, Recibos: recibos, FiltroEstado: estado}
	pendPorUnidad := map[uint]float64{}
	for _, rec := range todos {
		if rec.Pagado() {
			data.TotalPagado += rec.MontoPagado
			if rec.SaldoFavor > 0 {
				data.TotalSaldoFavor += rec.SaldoFavor
			}
			continue
		}
		monto := rec.MontoPagar()
		data.TotalPendiente += monto
		pendPorUnidad[rec.IDDepartamento] += monto
	}

	// Unidades actuales (propietario o inquilino).
	var actuales []models.Departamento
	a.DB.Where("id_dueño = ? OR id_inquilino = ?", p.ID, p.ID).Order("nombre ASC").Find(&actuales)
	vistos := map[uint]bool{}
	for _, d := range actuales {
		rol := ""
		if d.IDDueño == p.ID {
			rol = "propietario"
		}
		if d.IDInquilino != nil && *d.IDInquilino == p.ID {
			if rol == "" {
				rol = "inquilino"
			} else {
				rol += " e inquilino"
			}
		}
		data.Unidades = append(data.Unidades, UnidadCuenta{
			Departamento: d, Rol: rol, Pendiente: pendPorUnidad[d.ID],
		})
		vistos[d.ID] = true
	}

	// Unidades históricas con deuda que ya no están asociadas.
	var idsHist []uint
	for uid := range pendPorUnidad {
		if uid != 0 && !vistos[uid] {
			idsHist = append(idsHist, uid)
		}
	}
	if len(idsHist) > 0 {
		var hist []models.Departamento
		a.DB.Where("id IN ?", idsHist).Order("nombre ASC").Find(&hist)
		for _, d := range hist {
			data.Unidades = append(data.Unidades, UnidadCuenta{
				Departamento: d, Rol: "histórica", Pendiente: pendPorUnidad[d.ID], Historica: true,
			})
		}
	}

	a.render(w, r, "persona_cuenta", PageData{Data: data})
}
