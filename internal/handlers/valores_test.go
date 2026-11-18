package handlers_test

import (
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"recibosgo/internal/models"
)

func TestGenerarExpensas(t *testing.T) {
	e := newTestEnv(t)
	s := seedScenario(t, e)

	form := url.Values{
		"periodo":        {"2026-11"},
		"total_expensas": {"1000"},
		"total_sat_cisi": {"200"},
	}
	rec := e.request(t, http.MethodPost, "/deudas", e.sessionCookie(t, s.Admin), form)
	if rec.Code != http.StatusFound {
		t.Fatalf("generar expensas: status = %d, quería 302", rec.Code)
	}

	var recibos []models.Recibo
	e.DB.Where("periodo = ? AND tipo = ?", "2026-11", "expensas").Find(&recibos)
	if len(recibos) != 4 {
		t.Fatalf("recibos generados = %d, quería 4", len(recibos))
	}

	want := map[uint]float64{
		s.DeptoA1.ID: 600, // 1000*50% + 200*50%
		s.DeptoA2.ID: 300, // 1000*25% + 200*25%
		s.DeptoB1.ID: 300,
		s.DeptoT1.ID: 120, // 1000*10% + 200*10%
	}
	deudor := map[uint]uint{
		s.DeptoA1.ID: s.PersonaA.ID,
		s.DeptoA2.ID: s.PersonaA.ID,
		s.DeptoB1.ID: s.PersonaB.ID,
		s.DeptoT1.ID: s.PersonaT.ID, // con inquilino, se imputa al inquilino
	}
	for _, r := range recibos {
		esperado, ok := want[r.IDDepartamento]
		if !ok {
			t.Fatalf("recibo para depto inesperado %d", r.IDDepartamento)
		}
		if math.Abs(r.Total-esperado) > 0.01 {
			t.Fatalf("total depto %d = %.2f, quería %.2f", r.IDDepartamento, r.Total, esperado)
		}
		if r.IDPersona == nil || *r.IDPersona != deudor[r.IDDepartamento] {
			t.Fatalf("deudor depto %d incorrecto", r.IDDepartamento)
		}
	}

	// Idempotencia: un segundo POST no debe duplicar.
	rec = e.request(t, http.MethodPost, "/deudas", e.sessionCookie(t, s.Admin), form)
	if rec.Code != http.StatusFound {
		t.Fatalf("segundo POST: status = %d, quería 302", rec.Code)
	}
	var count int64
	e.DB.Model(&models.Recibo{}).Where("periodo = ? AND tipo = ?", "2026-11", "expensas").Count(&count)
	if count != 4 {
		t.Fatalf("tras segundo POST hay %d recibos, quería 4", count)
	}
}

func TestGenerarServicios(t *testing.T) {
	e := newTestEnv(t)
	s := seedScenario(t, e)

	for _, c := range []struct {
		id  uint
		kwh float64
	}{
		{s.DeptoA1.ID, 100},
		{s.DeptoA2.ID, 50},
		{s.DeptoB1.ID, 50},
		{s.DeptoT1.ID, 0},
	} {
		if err := e.DB.Create(&models.ConsumoLuz{
			Periodo: "2026-11", IDDepartamento: c.id, ConsumoKwh: c.kwh,
		}).Error; err != nil {
			t.Fatalf("creando consumo: %v", err)
		}
	}

	rec := e.request(t, http.MethodPost, "/deudas/servicios", e.sessionCookie(t, s.Admin), url.Values{
		"periodo":       {"2026-11"},
		"total_factura": {"20000"},
		"precio_kw":     {"50"},
	})
	if rec.Code != http.StatusFound {
		t.Fatalf("generar servicios: status = %d, quería 302", rec.Code)
	}

	var consumos []models.ConsumoLuz
	e.DB.Where("periodo = ?", "2026-11").Find(&consumos)
	if len(consumos) != 4 {
		t.Fatalf("consumos = %d, quería 4", len(consumos))
	}
	for _, c := range consumos {
		// (20000 - 200 kWh * 50) / 4 = 2500.00
		if math.Abs(c.ServicioComun-2500) > 0.01 {
			t.Fatalf("servicio común = %.2f, quería 2500.00", c.ServicioComun)
		}
	}

	var recibos []models.Recibo
	e.DB.Where("periodo = ? AND tipo = ?", "2026-11", "servicios").Find(&recibos)
	if len(recibos) != 4 {
		t.Fatalf("recibos de servicios = %d, quería 4", len(recibos))
	}
	want := map[uint]float64{
		s.DeptoA1.ID: 7500, // 100*50 + 2500
		s.DeptoA2.ID: 5000, // 50*50 + 2500
		s.DeptoB1.ID: 5000,
		s.DeptoT1.ID: 2500, // 0*50 + 2500
	}
	deudor := map[uint]uint{
		s.DeptoA1.ID: s.PersonaA.ID,
		s.DeptoA2.ID: s.PersonaA.ID,
		s.DeptoB1.ID: s.PersonaB.ID,
		s.DeptoT1.ID: s.PersonaT.ID,
	}
	for _, r := range recibos {
		esperado, ok := want[r.IDDepartamento]
		if !ok {
			t.Fatalf("recibo para depto inesperado %d", r.IDDepartamento)
		}
		if math.Abs(r.Total-esperado) > 0.01 {
			t.Fatalf("total depto %d = %.2f, quería %.2f", r.IDDepartamento, r.Total, esperado)
		}
		if r.IDPersona == nil || *r.IDPersona != deudor[r.IDDepartamento] {
			t.Fatalf("deudor depto %d incorrecto", r.IDDepartamento)
		}
	}
}

func TestFiltradoRecibosPorRol(t *testing.T) {
	e := newTestEnv(t)
	s := seedScenario(t, e)

	cases := []struct {
		name   string
		cookie *http.Cookie
		want   []string
		not    []string
	}{
		{
			name:   "admin",
			cookie: e.sessionCookie(t, s.Admin),
			want:   []string{s.ReciboA1.NumeroRecibo, s.ReciboB1.NumeroRecibo, s.ReciboT1.NumeroRecibo},
		},
		{
			name:   "propietario A",
			cookie: e.sessionCookie(t, s.OwnerA),
			want:   []string{s.ReciboA1.NumeroRecibo, s.ReciboA2.NumeroRecibo},
			not:    []string{s.ReciboB1.NumeroRecibo, s.ReciboT1.NumeroRecibo},
		},
		{
			name:   "inquilino",
			cookie: e.sessionCookie(t, s.Tenant),
			want:   []string{s.ReciboT1.NumeroRecibo},
			not:    []string{s.ReciboA1.NumeroRecibo, s.ReciboB1.NumeroRecibo},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := e.request(t, http.MethodGet, "/recibos", tc.cookie, nil)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, quería 200", rec.Code)
			}
			body := rec.Body.String()
			for _, w := range tc.want {
				if !strings.Contains(body, w) {
					t.Fatalf("no aparece el recibo propio %s", w)
				}
			}
			for _, n := range tc.not {
				if strings.Contains(body, n) {
					t.Fatalf("aparece el recibo ajeno %s", n)
				}
			}
		})
	}
}

func TestPagarDesdeLista(t *testing.T) {
	e := newTestEnv(t)
	s := seedScenario(t, e)
	admin := e.sessionCookie(t, s.Admin)
	link := fmt.Sprintf("/recibos/%d", s.ReciboA1.ID)

	// Por defecto /deudas muestra pendientes.
	rec := e.request(t, http.MethodGet, "/deudas", admin, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /deudas: status = %d, quería 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), link) {
		t.Fatal("la deuda pendiente no aparece por defecto")
	}

	// Pagar A1 desde la lista, volviendo a /deudas.
	rec = e.request(t, http.MethodPost, fmt.Sprintf("/recibos/%d/pagar", s.ReciboA1.ID), admin, url.Values{
		"forma_pago": {"T"},
		"volver":     {"/deudas?estado=pendiente"},
	})
	if rec.Code != http.StatusFound {
		t.Fatalf("pagar: status = %d, quería 302", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/deudas?estado=pendiente" {
		t.Fatalf("redirect = %q, quería /deudas?estado=pendiente", loc)
	}

	// Quedó pagado.
	var detalles []models.DetalleRecibo
	e.DB.Where("id_recibo = ?", s.ReciboA1.ID).Find(&detalles)
	if len(detalles) == 0 {
		t.Fatal("el recibo no tiene detalles")
	}
	for _, d := range detalles {
		if d.FormaPago != "T" {
			t.Fatalf("forma_pago = %q, quería T", d.FormaPago)
		}
	}

	// Ya no aparece en pendientes; sí en pagados.
	rec = e.request(t, http.MethodGet, "/deudas?estado=pendiente", admin, nil)
	if strings.Contains(rec.Body.String(), link) {
		t.Fatal("el recibo pagado sigue en pendientes")
	}
	rec = e.request(t, http.MethodGet, "/deudas?estado=pagado", admin, nil)
	if !strings.Contains(rec.Body.String(), link) {
		t.Fatal("el recibo pagado no aparece en pagados")
	}

	var pagado models.Recibo
	e.DB.First(&pagado, s.ReciboA1.ID)
	if pagado.FechaPago == nil {
		t.Fatal("no se grabó la fecha de pago")
	}
	if math.Abs(pagado.MontoPagado-1000) > 0.01 {
		t.Fatalf("monto pagado = %.2f, quería 1000", pagado.MontoPagado)
	}
}

func TestRecargoPorVencimiento(t *testing.T) {
	e := newTestEnv(t)
	s := seedScenario(t, e)

	if err := e.DB.Exec(
		"UPDATE recibo SET fecha_corte = '2020-01-01', total_con_interes = 1200 WHERE id = ?",
		s.ReciboA1.ID,
	).Error; err != nil {
		t.Fatalf("actualizando recibo: %v", err)
	}
	var r models.Recibo
	if err := e.DB.First(&r, s.ReciboA1.ID).Error; err != nil {
		t.Fatalf("leyendo recibo: %v", err)
	}
	if !r.Vencido() {
		t.Fatal("el recibo debería estar vencido")
	}
	if r.MontoPagar() != 1200 {
		t.Fatalf("monto a pagar = %.2f, quería 1200 (con recargo)", r.MontoPagar())
	}
}

func TestHistorialUnidad(t *testing.T) {
	e := newTestEnv(t)
	s := seedScenario(t, e)

	// Segundo deudor en la misma unidad (simula cambio de inquilino).
	otro := e.createRecibo(t, s.DeptoT1.ID, &s.PersonaB.ID, "expensas", "2026-02", 400, []models.DetalleRecibo{
		{IDServicio: s.ServicioExpensas.ID, Precio: 400, Cantidad: 1},
	})

	rec := e.request(t, http.MethodGet, fmt.Sprintf("/departamentos/%d/historial", s.DeptoT1.ID), e.sessionCookie(t, s.Admin), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("historial: status = %d, quería 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, s.PersonaT.Nombre) {
		t.Fatalf("el historial no muestra al inquilino %s", s.PersonaT.Nombre)
	}
	if !strings.Contains(body, s.PersonaB.Nombre) {
		t.Fatalf("el historial no muestra al propietario %s", s.PersonaB.Nombre)
	}
	if !strings.Contains(body, fmt.Sprintf("/recibos/%d", otro.ID)) {
		t.Fatal("el historial no lista el recibo del segundo deudor")
	}
}

func TestDashboardDeudasAnteriores(t *testing.T) {
	e := newTestEnv(t)
	s := seedScenario(t, e)

	rec := e.request(t, http.MethodGet, "/dashboard", e.sessionCookie(t, s.Admin), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("dashboard: status = %d, quería 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Deudas de inquilinos anteriores") {
		t.Fatal("el dashboard no muestra la sección de deudas anteriores")
	}
}

func TestPagoParcialGeneraDeudaAnterior(t *testing.T) {
	e := newTestEnv(t)
	s := seedScenario(t, e)
	admin := e.sessionCookie(t, s.Admin)

	rec := e.request(t, http.MethodPost, fmt.Sprintf("/recibos/%d/pagar", s.ReciboA1.ID), admin, url.Values{
		"forma_pago": {"E"},
		"importe":    {"400"},
		"volver":     {"/deudas?estado=pendiente"},
	})
	if rec.Code != http.StatusFound {
		t.Fatalf("pago parcial: status = %d, quería 302", rec.Code)
	}

	var original models.Recibo
	if err := e.DB.First(&original, s.ReciboA1.ID).Error; err != nil {
		t.Fatalf("leyendo original: %v", err)
	}
	if !original.PagoParcial {
		t.Fatal("el original no quedó marcado como pago parcial")
	}
	if original.MontoPagado != 400 {
		t.Fatalf("monto pagado = %.2f, quería 400", original.MontoPagado)
	}
	if original.SaldoFavor != 0 {
		t.Fatalf("saldo a favor = %.2f, quería 0", original.SaldoFavor)
	}

	var carry models.Recibo
	if err := e.DB.Where("id_recibo_origen = ?", s.ReciboA1.ID).First(&carry).Error; err != nil {
		t.Fatalf("no se generó la deuda anterior: %v", err)
	}
	if carry.Tipo != "deuda_anterior" {
		t.Fatalf("tipo = %q, quería deuda_anterior", carry.Tipo)
	}
	if math.Abs(carry.Total-600) > 0.01 {
		t.Fatalf("saldo = %.2f, quería 600", carry.Total)
	}
	if carry.IDPersona == nil || *carry.IDPersona != s.PersonaA.ID {
		t.Fatal("la deuda anterior no quedó imputada al deudor original")
	}
	if carry.Periodo != time.Now().Format("2006-01") {
		t.Fatalf("período = %q, quería el mes de pago", carry.Periodo)
	}

	// El original queda cerrado; la deuda anterior aparece como pendiente.
	body := e.request(t, http.MethodGet, "/deudas?estado=pendiente", admin, nil).Body.String()
	if strings.Contains(body, fmt.Sprintf("/recibos/%d", s.ReciboA1.ID)) {
		t.Fatal("el recibo original sigue apareciendo como pendiente")
	}
	if !strings.Contains(body, fmt.Sprintf("/recibos/%d", carry.ID)) {
		t.Fatal("la deuda anterior no aparece como pendiente")
	}
}

func TestSobrepagoSaldoAFavor(t *testing.T) {
	e := newTestEnv(t)
	s := seedScenario(t, e)
	admin := e.sessionCookie(t, s.Admin)

	rec := e.request(t, http.MethodPost, fmt.Sprintf("/recibos/%d/pagar", s.ReciboA1.ID), admin, url.Values{
		"forma_pago": {"T"},
		"importe":    {"1200"},
	})
	if rec.Code != http.StatusFound {
		t.Fatalf("sobrepago: status = %d, quería 302", rec.Code)
	}

	var r models.Recibo
	if err := e.DB.First(&r, s.ReciboA1.ID).Error; err != nil {
		t.Fatalf("leyendo recibo: %v", err)
	}
	if r.MontoPagado != 1200 {
		t.Fatalf("monto pagado = %.2f, quería 1200", r.MontoPagado)
	}
	if r.SaldoFavor != 200 {
		t.Fatalf("saldo a favor = %.2f, quería 200", r.SaldoFavor)
	}
	if r.PagoParcial {
		t.Fatal("un sobrepago no debe marcarse como pago parcial")
	}
	var count int64
	e.DB.Model(&models.Recibo{}).Where("id_recibo_origen = ?", s.ReciboA1.ID).Count(&count)
	if count != 0 {
		t.Fatal("un sobrepago no debe generar deuda anterior")
	}
}

func TestGenerarExtraordinarias(t *testing.T) {
	e := newTestEnv(t)
	s := seedScenario(t, e)

	rec := e.request(t, http.MethodPost, "/deudas/extraordinarias", e.sessionCookie(t, s.Admin), url.Values{
		"periodo": {"2026-11"},
		"total":   {"1000"},
	})
	if rec.Code != http.StatusFound {
		t.Fatalf("generar extraordinarias: status = %d, quería 302", rec.Code)
	}

	var recibos []models.Recibo
	e.DB.Where("periodo = ? AND tipo = ?", "2026-11", "extraordinarias").Find(&recibos)
	if len(recibos) != 4 {
		t.Fatalf("recibos extraordinarios = %d, quería 4", len(recibos))
	}
	want := map[uint]float64{
		s.DeptoA1.ID: 500,
		s.DeptoA2.ID: 250,
		s.DeptoB1.ID: 250,
		s.DeptoT1.ID: 100,
	}
	deudor := map[uint]uint{
		s.DeptoA1.ID: s.PersonaA.ID,
		s.DeptoA2.ID: s.PersonaA.ID,
		s.DeptoB1.ID: s.PersonaB.ID,
		s.DeptoT1.ID: s.PersonaB.ID, // propietario, nunca inquilino
	}
	for _, r := range recibos {
		if math.Abs(r.Total-want[r.IDDepartamento]) > 0.01 {
			t.Fatalf("total depto %d = %.2f, quería %.2f", r.IDDepartamento, r.Total, want[r.IDDepartamento])
		}
		if r.IDPersona == nil || *r.IDPersona != deudor[r.IDDepartamento] {
			t.Fatalf("deudor depto %d incorrecto: extraordinarias van al propietario", r.IDDepartamento)
		}
	}
}

func TestInquilinoNoVeExtraordinarias(t *testing.T) {
	e := newTestEnv(t)
	s := seedScenario(t, e)
	admin := e.sessionCookie(t, s.Admin)
	tenant := e.sessionCookie(t, s.Tenant)

	e.request(t, http.MethodPost, "/deudas", admin, url.Values{
		"periodo": {"2026-11"}, "total_expensas": {"1000"}, "total_sat_cisi": {"200"},
	})
	e.request(t, http.MethodPost, "/deudas/extraordinarias", admin, url.Values{
		"periodo": {"2026-11"}, "total": {"1000"},
	})

	var expT, extT models.Recibo
	e.DB.Where("periodo = ? AND tipo = ? AND id_departamento = ?", "2026-11", "expensas", s.DeptoT1.ID).First(&expT)
	e.DB.Where("periodo = ? AND tipo = ? AND id_departamento = ?", "2026-11", "extraordinarias", s.DeptoT1.ID).First(&extT)
	if expT.ID == 0 || extT.ID == 0 {
		t.Fatal("no se generaron los recibos de T1")
	}

	rec := e.request(t, http.MethodGet, fmt.Sprintf("/recibos/%d", expT.ID), tenant, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("el inquilino no puede ver su expensa: status = %d", rec.Code)
	}
	rec = e.request(t, http.MethodGet, fmt.Sprintf("/recibos/%d", extT.ID), tenant, nil)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("el inquilino no debería ver extraordinarias: status = %d", rec.Code)
	}

	rec = e.request(t, http.MethodGet, "/deudas", tenant, nil)
	body := rec.Body.String()
	if !strings.Contains(body, fmt.Sprintf("/recibos/%d", expT.ID)) {
		t.Fatal("el inquilino no ve su expensa en /deudas")
	}
	if strings.Contains(body, fmt.Sprintf("/recibos/%d", extT.ID)) {
		t.Fatal("el inquilino ve una extraordinaria en /deudas")
	}
}
