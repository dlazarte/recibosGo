package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"recibosgo/internal/middleware"
	"recibosgo/internal/models"
)

type RecibosData struct {
	Recibos       []models.Recibo
	Departamentos []models.Departamento
	Servicios     []models.Servicio
	FiltroDepto   string
	FiltroMes     string
	FiltroEstado  string
	Msg           string
	Paginado      Pagination
}

type ReciboCreateData struct {
	Personas      []models.Persona
	Departamentos []models.Departamento
	Servicios     []models.Servicio
	Fecha         string
	PersonaID     uint
	PersonaTipo   string // propietario | inquilino
}

type ReciboDetailData struct {
	Recibo models.Recibo
	Saldo  float64
}

func (a *App) RecibosList(w http.ResponseWriter, r *http.Request) {
	user, _ := middleware.UserFromContext(r.Context())

	depID := r.URL.Query().Get("departamento")
	mes := r.URL.Query().Get("mes")
	estado := r.URL.Query().Get("estado")

	var q *gorm.DB
	{
		base := a.DB.Model(&models.Recibo{})
		if user.Role != "administrador" && user.Role != "receptor" {
			// Find departments via persona (v2.1)
			var persona models.Persona
			var ids []uint
			found := a.DB.Where("email = ?", user.Email).First(&persona).Error == nil
			if found {
				var depts []models.Departamento
				a.DB.Where("id_dueño = ? OR id_inquilino = ?", persona.ID, persona.ID).Find(&depts)
				for _, d := range depts {
					ids = append(ids, d.ID)
				}
			}
			if len(ids) == 0 {
				ids = []uint{0}
			}
			if user.Role == "inquilino" && found {
				// El inquilino ve solo lo imputado a él; nil = legacy.
				base = base.Where("id_persona = ? OR (id_persona IS NULL AND id_departamento IN ?)", persona.ID, ids)
			} else {
				base = base.Where("id_departamento IN ?", ids)
			}
		}
		if depID != "" && depID != "0" {
			base = base.Where("id_departamento = ?", depID)
		}
		if mes != "" {
			if start, err := time.Parse("2006-01", mes); err == nil {
				end := start.AddDate(0, 1, 0)
				base = base.Where("fecha >= ? AND fecha < ?", start, end)
			}
		}
		if estado == "pagado" {
			base = base.Where("id IN (SELECT DISTINCT id_recibo FROM detallerecibo WHERE forma_pago <> '')")
		} else if estado == "pendiente" {
			base = base.Where("id NOT IN (SELECT DISTINCT id_recibo FROM detallerecibo WHERE forma_pago <> '')")
		}
		q = base
	}

	var total int64
	q.Count(&total)

	page := 1
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		page = p
	}

	var recibos []models.Recibo
	q.Preload("Departamento").Preload("Titular").Preload("Detalles.Servicio").
		Order("fecha DESC, id DESC").
		Offset((page - 1) * perPage).
		Limit(perPage).
		Find(&recibos)

	var depts []models.Departamento
	a.DB.Order(ordenUnidades).Find(&depts)

	data := RecibosData{
		Recibos:       recibos,
		Departamentos: depts,
		FiltroDepto:   depID,
		FiltroMes:     mes,
		FiltroEstado:  estado,
		Msg:           r.URL.Query().Get("msg"),
		Paginado:      paginar(r, total, "#recibos-table"),
	}
	if r.Header.Get("HX-Request") == "true" {
		a.renderFrag(w, "recibos_table", PageData{Data: data})
		return
	}
	a.render(w, r, "recibos_list", PageData{Data: data})
}

func (a *App) ReciboNew(w http.ResponseWriter, r *http.Request) {
	var personas []models.Persona
	a.DB.Where("activo = ?", true).Order("nombre ASC").Find(&personas)
	var servicios []models.Servicio
	a.DB.Where("activo = ?", true).Order("nombre ASC").Find(&servicios)

	fecha := time.Now().Format("2006-01-02")
	if v := r.URL.Query().Get("fecha"); v != "" {
		fecha = v
	}

	// HTMX: load departments for selected persona (HTMX envía el name del select)
	var depts []models.Departamento
	personaID, _ := strconv.Atoi(r.URL.Query().Get("id_persona"))
	if personaID == 0 {
		personaID, _ = strconv.Atoi(r.URL.Query().Get("persona_id"))
	}
	var personaTipo string
	if personaID > 0 {
		var persona models.Persona
		if a.DB.First(&persona, personaID).Error == nil {
			personaTipo = persona.Tipo
		}
		a.DB.Where("(id_dueño = ? OR id_inquilino = ?) AND activo = true", personaID, personaID).
			Order("nombre ASC").Find(&depts)
	}

	data := ReciboCreateData{
		Personas:      personas,
		Departamentos: depts,
		Servicios:     servicios,
		Fecha:         fecha,
		PersonaID:     uint(personaID),
		PersonaTipo:   personaTipo,
	}

	if r.Header.Get("HX-Request") == "true" {
		a.renderFrag(w, "recibo_unidades", data)
		return
	}
	a.render(w, r, "recibo_create", PageData{Data: data})
}

func (a *App) ReciboCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	personaID, _ := strconv.Atoi(r.FormValue("id_persona"))
	fechaStr := r.FormValue("fecha")
	if fechaStr == "" {
		fechaStr = time.Now().Format("2006-01-02")
	}
	fecha, err := time.Parse("2006-01-02", fechaStr)
	if err != nil {
		http.Redirect(w, r, "/recibos/new?msg="+urlEscape("Fecha inválida"), http.StatusFound)
		return
	}
	observaciones := r.FormValue("observaciones")
	var obsPtr *string
	if observaciones != "" {
		obsPtr = &observaciones
	}

	// v2.1: multiple departments (one receipt per department)
	deptoIDs := r.Form["id_departamento"]
	if len(deptoIDs) == 0 {
		http.Redirect(w, r, "/recibos/new?msg="+urlEscape("Seleccioná al menos una unidad"), http.StatusFound)
		return
	}

	// Validate: all departments must belong to the selected persona
	var personaTipo string
	if personaID > 0 {
		var persona models.Persona
		if a.DB.First(&persona, personaID).Error == nil {
			personaTipo = persona.Tipo
		}
		var validCount int64
		a.DB.Model(&models.Departamento{}).
			Where("(id_dueño = ? OR id_inquilino = ?) AND id IN ?", personaID, personaID, deptoIDs).
			Count(&validCount)
		if int(validCount) != len(deptoIDs) {
			http.Redirect(w, r, "/recibos/new?msg="+urlEscape("Departamentos inválidos para el titular seleccionado"), http.StatusFound)
			return
		}
	}

	ids := r.Form["id_servicio"]
	cants := r.Form["cantidad"]
	precios := r.Form["precio"]
	descs := r.Form["descripcion"]
	pagos := r.Form["forma_pago"]

	// Build items (shared across all departments)
	type item struct {
		servicioID uint
		cantidad   int
		precio     float64
		desc       *string
		formaPago  string
	}
	var items []item
	total := 0.0
	for i, rawID := range ids {
		sID, _ := strconv.Atoi(rawID)
		if sID == 0 {
			continue
		}
		cant := 1
		if i < len(cants) {
			if c, err := strconv.Atoi(cants[i]); err == nil && c > 0 {
				cant = c
			}
		}
		precio := 0.0
		if i < len(precios) {
			precio, _ = strconv.ParseFloat(precios[i], 64)
		}
		var desc *string
		if i < len(descs) && descs[i] != "" {
			d := descs[i]
			desc = &d
		}
		forma := ""
		if i < len(pagos) {
			forma = pagos[i]
		}
		items = append(items, item{
			servicioID: uint(sID),
			cantidad:   cant,
			precio:     precio,
			desc:       desc,
			formaPago:  forma,
		})
		total += precio * float64(cant)
	}

	if len(items) == 0 {
		http.Redirect(w, r, "/recibos/new?msg="+urlEscape("Agregá al menos un ítem"), http.StatusFound)
		return
	}

	// Regla de negocio: un inquilino no puede tener servicios extraordinarios
	if personaTipo == "inquilino" {
		for _, it := range items {
			var s models.Servicio
			if a.DB.First(&s, it.servicioID).Error == nil && strings.Contains(s.Nombre, "Extraordinarias") {
				http.Redirect(w, r, "/recibos/new?msg="+urlEscape("Los inquilinos no pueden cargar servicios extraordinarios"), http.StatusFound)
				return
			}
		}
	}

	// Create one receipt per department
	var created []uint
	for _, depStr := range deptoIDs {
		depID, _ := strconv.Atoi(depStr)
		if depID == 0 {
			continue
		}

		var detalles []models.DetalleRecibo
		for _, it := range items {
			detalles = append(detalles, models.DetalleRecibo{
				IDServicio:  it.servicioID,
				Precio:      it.precio,
				Cantidad:    it.cantidad,
				Descripcion: it.desc,
				FormaPago:   it.formaPago,
			})
		}

		recibo := models.Recibo{
			NumeroRecibo:   a.nextNumeroRecibo(fecha),
			Fecha:          models.Fecha{Time: fecha},
			IDDepartamento: uint(depID),
			IDPersona:      uintPtr(uint(personaID)),
			Observaciones:  obsPtr,
			Total:          total,
			Detalles:       detalles,
		}

		if err := a.DB.Create(&recibo).Error; err != nil {
			http.Redirect(w, r, "/recibos/new?msg="+urlEscape("Error al crear recibo: "+err.Error()), http.StatusFound)
			return
		}
		created = append(created, recibo.ID)
	}

	if len(created) == 1 {
		http.Redirect(w, r, fmt.Sprintf("/recibos/%d", created[0]), http.StatusFound)
		return
	}
	http.Redirect(w, r, "/recibos?msg="+urlEscape(fmt.Sprintf("Se generaron %d recibos", len(created))), http.StatusFound)
}

func uintPtr(v uint) *uint {
	if v == 0 {
		return nil
	}
	return &v
}

func (a *App) ReciboDetail(w http.ResponseWriter, r *http.Request) {
	user, _ := middleware.UserFromContext(r.Context())
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}
	var recibo models.Recibo
	if a.DB.Preload("Departamento").Preload("Detalles.Servicio").First(&recibo, id).Error != nil {
		http.Error(w, "recibo no encontrado", http.StatusNotFound)
		return
	}
	if !a.canAccessRecibo(user, &recibo) {
		http.Error(w, "Acceso denegado", http.StatusForbidden)
		return
	}
	var carry models.Recibo
	var saldo float64
	if a.DB.Where("id_recibo_origen = ?", recibo.ID).First(&carry).Error == nil {
		saldo = carry.Total
	}
	data := ReciboDetailData{Recibo: recibo, Saldo: saldo}
	msg := r.URL.Query().Get("msg")
	if r.Header.Get("HX-Request") == "true" {
		a.renderFrag(w, "recibo_detalle", PageData{Data: data, Msg: msg})
		return
	}
	a.render(w, r, "recibo_detail", PageData{Data: data, Msg: msg})
}

func (a *App) ReciboPagar(w http.ResponseWriter, r *http.Request) {
	user, _ := middleware.UserFromContext(r.Context())
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}
	forma := r.FormValue("forma_pago")
	if forma != "E" && forma != "T" {
		forma = "E"
	}
	var recibo models.Recibo
	if a.DB.Preload("Departamento").Preload("Detalles").First(&recibo, id).Error != nil {
		http.Error(w, "recibo no encontrado", http.StatusNotFound)
		return
	}
	if !a.canAccessRecibo(user, &recibo) {
		http.Error(w, "Acceso denegado", http.StatusForbidden)
		return
	}

	due := recibo.MontoPagar()
	importe := due
	if v := r.FormValue("importe"); v != "" {
		parsed, err := strconv.ParseFloat(v, 64)
		if err != nil || parsed <= 0 {
			http.Redirect(w, r, fmt.Sprintf("/recibos/%d?msg=%s", id, urlEscape("Importe inválido")), http.StatusFound)
			return
		}
		importe = round2(parsed)
	}
	parcial := importe < due-0.005
	favor := 0.0
	if importe > due+0.005 {
		favor = round2(importe - due)
	}

	svc := a.findOrCreateServicio(servicioDeudaAnterior)
	var saldo float64
	err = a.DB.Transaction(func(tx *gorm.DB) error {
		for i := range recibo.Detalles {
			recibo.Detalles[i].FormaPago = forma
		}
		if err := tx.Save(&recibo.Detalles).Error; err != nil {
			return err
		}
		recibo.FechaPago = &models.Fecha{Time: time.Now()}
		recibo.MontoPagado = importe
		recibo.PagoParcial = parcial
		recibo.SaldoFavor = favor
		if err := tx.Omit("Departamento", "Titular", "Detalles").Save(&recibo).Error; err != nil {
			return err
		}
		if !parcial {
			return nil
		}
		saldo = round2(due - importe)
		deudor := recibo.IDPersona
		if deudor == nil {
			deudor = personaDeudora(recibo.Departamento, false)
		}
		periodo := recibo.Periodo
		if periodo == "" {
			periodo = time.Now().Format("2006-01")
		}
		desc := fmt.Sprintf("Deudas Anteriores %s (recibo %s)", periodo, recibo.NumeroRecibo)
		nuevo := models.Recibo{
			NumeroRecibo:    nextNumeroRecibo(tx, time.Now()),
			Fecha:           models.Fecha{Time: time.Now()},
			Periodo:         time.Now().Format("2006-01"),
			Tipo:            "deuda_anterior",
			IDDepartamento:  recibo.IDDepartamento,
			IDPersona:       deudor,
			Total:           saldo,
			TotalConInteres: saldo,
			IDReciboOrigen:  &recibo.ID,
			Detalles: []models.DetalleRecibo{{
				IDServicio: svc.ID, Precio: saldo, Cantidad: 1, Descripcion: &desc,
			}},
		}
		return tx.Create(&nuevo).Error
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	msg := ""
	if parcial {
		msg = fmt.Sprintf("Pago parcial registrado. Saldo: $ %.2f", saldo)
	} else if favor > 0 {
		msg = fmt.Sprintf("Pago registrado. Saldo a favor: $ %.2f", favor)
	}

	if r.Header.Get("HX-Request") == "true" {
		var updated models.Recibo
		a.DB.Preload("Departamento").Preload("Detalles.Servicio").First(&updated, id)
		var carry models.Recibo
		var saldoTotal float64
		if a.DB.Where("id_recibo_origen = ?", updated.ID).First(&carry).Error == nil {
			saldoTotal = carry.Total
		}
		a.renderFrag(w, "recibo_detalle", PageData{
			Data: ReciboDetailData{Recibo: updated, Saldo: saldoTotal},
			Msg:  msg,
		})
		return
	}
	if back := r.FormValue("volver"); strings.HasPrefix(back, "/") && !strings.HasPrefix(back, "//") {
		if msg != "" {
			sep := "?"
			if strings.Contains(back, "?") {
				sep = "&"
			}
			back += sep + "msg=" + urlEscape(msg)
		}
		http.Redirect(w, r, back, http.StatusFound)
		return
	}
	if msg != "" {
		http.Redirect(w, r, fmt.Sprintf("/recibos/%d?msg=%s", id, urlEscape(msg)), http.StatusFound)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/recibos/%d", id), http.StatusFound)
}

func (a *App) nextNumeroRecibo(fecha time.Time) string {
	return nextNumeroRecibo(a.DB, fecha)
}

// canAccessRecibo: admin/receptor ven todo; dueño/inquilino solo sus unidades.
func (a *App) canAccessRecibo(user *models.User, recibo *models.Recibo) bool {
	if user == nil {
		return false
	}
	if user.Role == "administrador" || user.Role == "receptor" {
		return true
	}
	var persona models.Persona
	if a.DB.Where("email = ?", user.Email).First(&persona).Error != nil {
		return false
	}
	d := recibo.Departamento
	if d.ID == 0 {
		if a.DB.First(&d, recibo.IDDepartamento).Error != nil {
			return false
		}
	}
	if user.Role == "inquilino" {
		if recibo.IDPersona != nil {
			return *recibo.IDPersona == persona.ID
		}
		return d.IDInquilino != nil && *d.IDInquilino == persona.ID
	}
	// dueño
	if d.IDDueño == persona.ID {
		return true
	}
	return recibo.IDPersona != nil && *recibo.IDPersona == persona.ID
}

func nextNumeroRecibo(db *gorm.DB, fecha time.Time) string {
	prefix := "REC-" + fecha.Format("2006")
	var count int64
	db.Model(&models.Recibo{}).Where("numero_recibo LIKE ?", prefix+"-%").Count(&count)
	return fmt.Sprintf("%s-%04d", prefix, count+1)
}
