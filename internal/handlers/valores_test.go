package handlers_test

import (
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strings"
	"testing"

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
	if len(recibos) != 3 {
		t.Fatalf("recibos generados = %d, quería 3 (T1 tiene coeficiente 0)", len(recibos))
	}

	want := map[uint]float64{
		s.DeptoA1.ID: 600, // 1000*50% + 200*50%
		s.DeptoA2.ID: 300, // 1000*25% + 200*25%
		s.DeptoB1.ID: 300,
	}
	for _, r := range recibos {
		esperado, ok := want[r.IDDepartamento]
		if !ok {
			t.Fatalf("recibo para depto inesperado %d", r.IDDepartamento)
		}
		if math.Abs(r.Total-esperado) > 0.01 {
			t.Fatalf("total depto %d = %.2f, quería %.2f", r.IDDepartamento, r.Total, esperado)
		}
	}

	// Idempotencia: un segundo POST no debe duplicar.
	rec = e.request(t, http.MethodPost, "/deudas", e.sessionCookie(t, s.Admin), form)
	if rec.Code != http.StatusFound {
		t.Fatalf("segundo POST: status = %d, quería 302", rec.Code)
	}
	var count int64
	e.DB.Model(&models.Recibo{}).Where("periodo = ? AND tipo = ?", "2026-11", "expensas").Count(&count)
	if count != 3 {
		t.Fatalf("tras segundo POST hay %d recibos, quería 3", count)
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
	if len(consumos) != 3 {
		t.Fatalf("consumos = %d, quería 3", len(consumos))
	}
	for _, c := range consumos {
		// (20000 - 200 kWh * 50) / 3 = 3333.33
		if math.Abs(c.ServicioComun-3333.33) > 0.01 {
			t.Fatalf("servicio común = %.2f, quería 3333.33", c.ServicioComun)
		}
	}

	var recibos []models.Recibo
	e.DB.Where("periodo = ? AND tipo = ?", "2026-11", "servicios").Find(&recibos)
	if len(recibos) != 3 {
		t.Fatalf("recibos de servicios = %d, quería 3", len(recibos))
	}
	want := map[uint]float64{
		s.DeptoA1.ID: 8333.33, // 100*50 + 3333.33
		s.DeptoA2.ID: 5833.33, // 50*50 + 3333.33
		s.DeptoB1.ID: 5833.33,
	}
	for _, r := range recibos {
		esperado, ok := want[r.IDDepartamento]
		if !ok {
			t.Fatalf("recibo para depto inesperado %d", r.IDDepartamento)
		}
		if math.Abs(r.Total-esperado) > 0.01 {
			t.Fatalf("total depto %d = %.2f, quería %.2f", r.IDDepartamento, r.Total, esperado)
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
}
