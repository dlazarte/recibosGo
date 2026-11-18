package handlers

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"gorm.io/gorm"

	"recibosgo/internal/middleware"
	"recibosgo/internal/models"
)

// Servicios existentes que representan cada componente de la deuda mensual.
const (
	servicioExpensas        = "Expensas Comunes"
	servicioSatCisi         = "Servicios"
	servicioLuz             = "Servicios"
	servicioExtraordinarias = "Expensas Extraordinarias"
	servicioDeudaAnterior   = "Deudas Anteriores"
	tasaInteres             = 0.20
)

type DeudaItem struct {
	Recibo          models.Recibo
	Expensas        float64
	Servicios       float64
	Extraordinarias float64
	Anteriores      float64
}

type DeudasData struct {
	Items            []DeudaItem
	Departamentos    []models.Departamento
	Personas         []models.Persona
	FiltroDepto      string
	FiltroPersona    string
	FiltroMes        string
	FiltroEstado     string
	Periodo          string
	Lecturas         int64
	Msg              string
	Volver           string
	FiltroAnteriores bool
	Paginado         Pagination
}

// DeudasList muestra los registros de deuda por unidad o persona.
func (a *App) DeudasList(w http.ResponseWriter, r *http.Request) {
	user, _ := middleware.UserFromContext(r.Context())

	depID := r.URL.Query().Get("departamento")
	personaID := r.URL.Query().Get("persona")
	mes := r.URL.Query().Get("mes")
	estado := r.URL.Query().Get("estado")
	if estado == "" {
		estado = "pendiente"
	}

	q := a.DB.Model(&models.Recibo{}).Where("periodo <> ''")
	if user.Role != "administrador" && user.Role != "receptor" {
		persona, found := a.personaDelUsuario(user)
		var ids []uint
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
			// El inquilino ve solo lo imputado a él; el nil es legacy.
			q = q.Where("id_persona = ? OR (id_persona IS NULL AND id_departamento IN ?)", persona.ID, ids)
		} else {
			q = q.Where("id_departamento IN ?", ids)
		}
	}
	if depID != "" && depID != "0" {
		q = q.Where("id_departamento = ?", depID)
	}
	if personaID != "" && personaID != "0" {
		q = q.Where("id_departamento IN (SELECT id FROM departamento WHERE id_dueño = ? OR id_inquilino = ?)", personaID, personaID)
	}
	if mes != "" {
		if start, err := time.Parse("2006-01", mes); err == nil {
			end := start.AddDate(0, 1, 0)
			q = q.Where("fecha >= ? AND fecha < ?", start, end)
		}
	}
	if estado == "pagado" {
		q = q.Where("id IN (SELECT DISTINCT id_recibo FROM detallerecibo WHERE forma_pago <> '')")
	} else if estado == "pendiente" {
		q = q.Where("id NOT IN (SELECT DISTINCT id_recibo FROM detallerecibo WHERE forma_pago <> '')")
	}
	anteriores := r.URL.Query().Get("anteriores") == "1"
	if anteriores {
		q = q.Where("id_persona IS NOT NULL AND id_persona <> (SELECT id_dueño FROM departamento WHERE id = recibo.id_departamento)")
	}

	var total int64
	q.Count(&total)

	var recibos []models.Recibo
	q.Preload("Departamento.Dueño").Preload("Titular").Preload("Detalles.Servicio").
		Order("fecha DESC, id DESC").
		Offset((paginaActual(r) - 1) * perPage).
		Limit(perPage).
		Find(&recibos)

	items := make([]DeudaItem, 0, len(recibos))
	for _, rec := range recibos {
		it := DeudaItem{Recibo: rec}
		for _, d := range rec.Detalles {
			monto := d.Precio * float64(d.Cantidad)
			switch d.Servicio.Nombre {
			case servicioExpensas:
				it.Expensas += monto
			case servicioSatCisi:
				it.Servicios += monto
			case servicioExtraordinarias:
				it.Extraordinarias += monto
			case servicioDeudaAnterior:
				it.Anteriores += monto
			}
		}
		items = append(items, it)
	}

	var depts []models.Departamento
	a.DB.Order(ordenUnidades).Find(&depts)
	var personas []models.Persona
	a.DB.Order("nombre ASC").Find(&personas)

	periodo := mes
	if periodo == "" {
		periodo = time.Now().Format("2006-01")
	}
	var lecturas int64
	a.DB.Model(&models.ConsumoLuz{}).Where("periodo = ?", periodo).Count(&lecturas)

	data := DeudasData{
		Items:            items,
		Departamentos:    depts,
		Personas:         personas,
		FiltroDepto:      depID,
		FiltroPersona:    personaID,
		FiltroMes:        mes,
		FiltroEstado:     estado,
		Periodo:          periodo,
		Lecturas:         lecturas,
		Msg:              r.URL.Query().Get("msg"),
		Volver:           r.URL.RequestURI(),
		FiltroAnteriores: anteriores,
		Paginado:         paginar(r, total, "#deudas-table"),
	}
	if r.Header.Get("HX-Request") == "true" {
		a.renderFrag(w, "deudas_table", PageData{Data: data})
		return
	}
	a.render(w, r, "deudas", PageData{Data: data})
}

// DeudaGenerar crea un recibo por unidad activa prorrateando expensas y SAT/CISI por coeficiente.
func (a *App) DeudaGenerar(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	periodo := r.FormValue("periodo")
	start, err := time.Parse("2006-01", periodo)
	if err != nil {
		http.Redirect(w, r, "/deudas?msg="+urlEscape("Período inválido (usar AAAA-MM)"), http.StatusFound)
		return
	}
	totalExp, _ := strconv.ParseFloat(r.FormValue("total_expensas"), 64)
	totalSat, _ := strconv.ParseFloat(r.FormValue("total_sat_cisi"), 64)
	if totalExp < 0 || totalSat < 0 || (totalExp == 0 && totalSat == 0) {
		http.Redirect(w, r, "/deudas?msg="+urlEscape("Ingresá importes mayores a cero"), http.StatusFound)
		return
	}

	var existing int64
	a.DB.Model(&models.Recibo{}).Where("periodo = ? AND tipo = ?", periodo, "expensas").Count(&existing)
	if existing > 0 {
		http.Redirect(w, r, "/deudas?mes="+periodo+"&msg="+urlEscape("La deuda de expensas de "+periodo+" ya fue generada"), http.StatusFound)
		return
	}

	var depts []models.Departamento
	a.DB.Where("activo = ? AND coeficiente > 0", true).Order("nombre ASC").Find(&depts)
	if len(depts) == 0 {
		http.Redirect(w, r, "/deudas?msg="+urlEscape("No hay unidades activas con coeficiente mayor a cero"), http.StatusFound)
		return
	}

	expSvc := a.findOrCreateServicio(servicioExpensas)
	satSvc := a.findOrCreateServicio(servicioSatCisi)

	created := 0
	err = a.DB.Transaction(func(tx *gorm.DB) error {
		for _, d := range depts {
			exp, sat := calcDeuda(totalExp, totalSat, d.Coeficiente)
			if exp+sat == 0 {
				continue
			}
			descExp := "Expensas Comunes " + d.Nombre
			descSat := "Servicios " + d.Nombre
			var detalles []models.DetalleRecibo
			if exp > 0 {
				detalles = append(detalles, models.DetalleRecibo{
					IDServicio: expSvc.ID, Precio: exp, Cantidad: 1, Descripcion: &descExp,
				})
			}
			if sat > 0 {
				detalles = append(detalles, models.DetalleRecibo{
					IDServicio: satSvc.ID, Precio: sat, Cantidad: 1, Descripcion: &descSat,
				})
			}
			total := exp + sat
			recibo := models.Recibo{
				NumeroRecibo:    nextNumeroRecibo(tx, start),
				Fecha:           models.Fecha{Time: start},
				Periodo:         periodo,
				Tipo:            "expensas",
				IDDepartamento:  d.ID,
				IDPersona:       personaDeudora(d, false),
				Total:           total,
				FechaCorte:      corteDe(start),
				TotalConInteres: round2(total * (1 + tasaInteres)),
				Detalles:        detalles,
			}
			if err := tx.Create(&recibo).Error; err != nil {
				return err
			}
			created++
		}
		return nil
	})
	if err != nil {
		http.Redirect(w, r, "/deudas?msg="+urlEscape("Error al generar deudas: "+err.Error()), http.StatusFound)
		return
	}

	http.Redirect(w, r, "/deudas?mes="+periodo+"&msg="+urlEscape(fmt.Sprintf("Se generaron %d recibos de expensas para %s", created, periodo)), http.StatusFound)
}

// ServicioGenerar calcula la luz EDET: consumo por unidad + servicio común.
// servicio_común = (factura EDET − Σ(consumo × $/kW)) / cantidad de unidades.
func (a *App) ServicioGenerar(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	periodo := r.FormValue("periodo")
	start, err := time.Parse("2006-01", periodo)
	if err != nil {
		http.Redirect(w, r, "/deudas?msg="+urlEscape("Período inválido (usar AAAA-MM)"), http.StatusFound)
		return
	}
	totalFactura, _ := strconv.ParseFloat(r.FormValue("total_factura"), 64)
	precioKw, _ := strconv.ParseFloat(r.FormValue("precio_kw"), 64)
	if totalFactura <= 0 || precioKw <= 0 {
		http.Redirect(w, r, "/deudas?msg="+urlEscape("Ingresá factura y precio del kW mayores a cero"), http.StatusFound)
		return
	}

	var existing int64
	a.DB.Model(&models.Recibo{}).Where("periodo = ? AND tipo = ?", periodo, "servicios").Count(&existing)
	if existing > 0 {
		http.Redirect(w, r, "/deudas?mes="+periodo+"&msg="+urlEscape("Los servicios de "+periodo+" ya fueron generados"), http.StatusFound)
		return
	}

	var consumos []models.ConsumoLuz
	a.DB.Preload("Departamento").Where("periodo = ?", periodo).Order("id_departamento").Find(&consumos)
	if len(consumos) == 0 {
		http.Redirect(w, r, "/deudas?mes="+periodo+"&msg="+urlEscape("No hay lecturas de luz cargadas para "+periodo), http.StatusFound)
		return
	}

	suma := 0.0
	for _, c := range consumos {
		suma += c.ConsumoKwh * precioKw
	}
	comun := round2((totalFactura - suma) / float64(len(consumos)))

	luzSvc := a.findOrCreateServicio(servicioLuz)
	created := 0
	err = a.DB.Transaction(func(tx *gorm.DB) error {
		for _, c := range consumos {
			monto := round2(c.ConsumoKwh * precioKw)
			total := round2(monto + comun)
			c.PrecioKw = precioKw
			c.MontoConsumo = monto
			c.ServicioComun = comun
			c.Total = total
			if err := tx.Omit("Departamento").Save(&c).Error; err != nil {
				return err
			}
			desc := "Servicios " + c.Departamento.Nombre
			recibo := models.Recibo{
				NumeroRecibo:    nextNumeroRecibo(tx, start),
				Fecha:           models.Fecha{Time: start},
				Periodo:         periodo,
				Tipo:            "servicios",
				IDDepartamento:  c.IDDepartamento,
				IDPersona:       personaDeudora(c.Departamento, false),
				Total:           total,
				FechaCorte:      corteDe(start),
				TotalConInteres: round2(total * (1 + tasaInteres)),
				Detalles: []models.DetalleRecibo{{
					IDServicio: luzSvc.ID, Precio: total, Cantidad: 1, Descripcion: &desc,
				}},
			}
			if err := tx.Omit("Departamento", "Titular").Create(&recibo).Error; err != nil {
				return err
			}
			created++
		}
		return nil
	})
	if err != nil {
		http.Redirect(w, r, "/deudas?msg="+urlEscape("Error al generar servicios: "+err.Error()), http.StatusFound)
		return
	}

	http.Redirect(w, r, "/deudas?mes="+periodo+"&msg="+urlEscape(fmt.Sprintf("Se generaron %d recibos de servicios para %s (común $ %.2f)", created, periodo, comun)), http.StatusFound)
}

// ExtraordinariaGenerar divide un total de expensas extraordinarias por coeficiente
// y lo imputa siempre al propietario de la unidad.
func (a *App) ExtraordinariaGenerar(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	periodo := r.FormValue("periodo")
	start, err := time.Parse("2006-01", periodo)
	if err != nil {
		http.Redirect(w, r, "/deudas?msg="+urlEscape("Período inválido (usar AAAA-MM)"), http.StatusFound)
		return
	}
	total, _ := strconv.ParseFloat(r.FormValue("total"), 64)
	if total <= 0 {
		http.Redirect(w, r, "/deudas?msg="+urlEscape("Ingresá un importe mayor a cero"), http.StatusFound)
		return
	}

	var existing int64
	a.DB.Model(&models.Recibo{}).Where("periodo = ? AND tipo = ?", periodo, "extraordinarias").Count(&existing)
	if existing > 0 {
		http.Redirect(w, r, "/deudas?mes="+periodo+"&msg="+urlEscape("Las expensas extraordinarias de "+periodo+" ya fueron generadas"), http.StatusFound)
		return
	}

	var depts []models.Departamento
	a.DB.Where("activo = ? AND coeficiente > 0", true).Order("nombre ASC").Find(&depts)
	if len(depts) == 0 {
		http.Redirect(w, r, "/deudas?msg="+urlEscape("No hay unidades activas con coeficiente mayor a cero"), http.StatusFound)
		return
	}

	svc := a.findOrCreateServicio(servicioExtraordinarias)
	created := 0
	err = a.DB.Transaction(func(tx *gorm.DB) error {
		for _, d := range depts {
			monto := round2(total * d.Coeficiente / 100)
			if monto == 0 {
				continue
			}
			desc := "Expensas Extraordinarias " + d.Nombre
			recibo := models.Recibo{
				NumeroRecibo:    nextNumeroRecibo(tx, start),
				Fecha:           models.Fecha{Time: start},
				Periodo:         periodo,
				Tipo:            "extraordinarias",
				IDDepartamento:  d.ID,
				IDPersona:       personaDeudora(d, true),
				Total:           monto,
				FechaCorte:      corteDe(start),
				TotalConInteres: round2(monto * (1 + tasaInteres)),
				Detalles: []models.DetalleRecibo{{
					IDServicio: svc.ID, Precio: monto, Cantidad: 1, Descripcion: &desc,
				}},
			}
			if err := tx.Create(&recibo).Error; err != nil {
				return err
			}
			created++
		}
		return nil
	})
	if err != nil {
		http.Redirect(w, r, "/deudas?msg="+urlEscape("Error al generar extraordinarias: "+err.Error()), http.StatusFound)
		return
	}

	http.Redirect(w, r, "/deudas?mes="+periodo+"&msg="+urlEscape(fmt.Sprintf("Se generaron %d recibos extraordinarios para %s", created, periodo)), http.StatusFound)
}

// personaDeudora: inquilino si existe (salvo soloPropietario), si no el dueño.
func personaDeudora(d models.Departamento, soloPropietario bool) *uint {
	if !soloPropietario && d.IDInquilino != nil && *d.IDInquilino != 0 {
		return d.IDInquilino
	}
	return uintPtr(d.IDDueño)
}

// calcDeuda aplica el coeficiente (en %) sobre los totales y redondea a 2 decimales.
func calcDeuda(totalExpensas, totalSatCisi, coeficiente float64) (float64, float64) {
	return round2(totalExpensas * coeficiente / 100), round2(totalSatCisi * coeficiente / 100)
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// corteDe devuelve el día 15 del período (fecha de corte para el recargo).
func corteDe(start time.Time) *models.Fecha {
	d := time.Date(start.Year(), start.Month(), 15, 0, 0, 0, 0, start.Location())
	return &models.Fecha{Time: d}
}

func (a *App) findOrCreateServicio(nombre string) models.Servicio {
	var s models.Servicio
	if err := a.DB.Where("nombre = ?", nombre).First(&s).Error; err != nil {
		s = models.Servicio{Nombre: nombre, Activo: true}
		a.DB.Create(&s)
	}
	return s
}
