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
	servicioExpensas = "Expensas Comunes. Pago Final"
	servicioSatCisi  = "Servicios. Pago Final"
	servicioLuz      = "Luz (EDET)"
)

type DeudaItem struct {
	Recibo   models.Recibo
	Expensas float64
	SatCisi  float64
	Luz      float64
}

type DeudasData struct {
	Items         []DeudaItem
	Departamentos []models.Departamento
	Personas      []models.Persona
	FiltroDepto   string
	FiltroPersona string
	FiltroMes     string
	Periodo       string
	Lecturas      int64
	Msg           string
	Paginado      Pagination
}

// DeudasList muestra los registros de deuda por unidad o persona.
func (a *App) DeudasList(w http.ResponseWriter, r *http.Request) {
	user, _ := middleware.UserFromContext(r.Context())

	depID := r.URL.Query().Get("departamento")
	personaID := r.URL.Query().Get("persona")
	mes := r.URL.Query().Get("mes")

	q := a.DB.Model(&models.Recibo{}).Where("periodo <> ''")
	if user.Role != "administrador" && user.Role != "receptor" {
		var persona models.Persona
		var ids []uint
		if a.DB.Where("email = ?", user.Email).First(&persona).Error == nil {
			var depts []models.Departamento
			a.DB.Where("id_dueño = ? OR id_inquilino = ?", persona.ID, persona.ID).Find(&depts)
			for _, d := range depts {
				ids = append(ids, d.ID)
			}
		}
		if len(ids) == 0 {
			ids = []uint{0}
		}
		q = q.Where("id_departamento IN ?", ids)
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

	var total int64
	q.Count(&total)

	var recibos []models.Recibo
	q.Preload("Departamento.Dueño").Preload("Detalles.Servicio").
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
				it.SatCisi += monto
			case servicioLuz:
				it.Luz += monto
			}
		}
		items = append(items, it)
	}

	var depts []models.Departamento
	a.DB.Order("numero+0, nombre ASC").Find(&depts)
	var personas []models.Persona
	a.DB.Order("nombre ASC").Find(&personas)

	periodo := mes
	if periodo == "" {
		periodo = time.Now().Format("2006-01")
	}
	var lecturas int64
	a.DB.Model(&models.ConsumoLuz{}).Where("periodo = ?", periodo).Count(&lecturas)

	data := DeudasData{
		Items:         items,
		Departamentos: depts,
		Personas:      personas,
		FiltroDepto:   depID,
		FiltroPersona: personaID,
		FiltroMes:     mes,
		Periodo:       periodo,
		Lecturas:      lecturas,
		Msg:           r.URL.Query().Get("msg"),
		Paginado:      paginar(r, total, "#deudas-table"),
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
			nombre := d.Nombre
			var detalles []models.DetalleRecibo
			if exp > 0 {
				detalles = append(detalles, models.DetalleRecibo{
					IDServicio: expSvc.ID, Precio: exp, Cantidad: 1, Descripcion: &nombre,
				})
			}
			if sat > 0 {
				detalles = append(detalles, models.DetalleRecibo{
					IDServicio: satSvc.ID, Precio: sat, Cantidad: 1, Descripcion: &nombre,
				})
			}
			recibo := models.Recibo{
				NumeroRecibo:   nextNumeroRecibo(tx, start),
				Fecha:          models.Fecha{Time: start},
				Periodo:        periodo,
				Tipo:           "expensas",
				IDDepartamento: d.ID,
				IDPersona:      uintPtr(d.IDDueño),
				Total:          exp + sat,
				Detalles:       detalles,
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
			desc := "Luz EDET " + periodo
			recibo := models.Recibo{
				NumeroRecibo:   nextNumeroRecibo(tx, start),
				Fecha:          models.Fecha{Time: start},
				Periodo:        periodo,
				Tipo:           "servicios",
				IDDepartamento: c.IDDepartamento,
				IDPersona:      uintPtr(c.Departamento.IDDueño),
				Total:          total,
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

// calcDeuda aplica el coeficiente (en %) sobre los totales y redondea a 2 decimales.
func calcDeuda(totalExpensas, totalSatCisi, coeficiente float64) (float64, float64) {
	return round2(totalExpensas * coeficiente / 100), round2(totalSatCisi * coeficiente / 100)
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func (a *App) findOrCreateServicio(nombre string) models.Servicio {
	var s models.Servicio
	if err := a.DB.Where("nombre = ?", nombre).First(&s).Error; err != nil {
		s = models.Servicio{Nombre: nombre, Activo: true}
		a.DB.Create(&s)
	}
	return s
}
