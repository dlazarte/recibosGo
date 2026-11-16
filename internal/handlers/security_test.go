package handlers_test

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestRoleMatrix(t *testing.T) {
	e := newTestEnv(t)
	s := seedScenario(t, e)

	cookies := map[string]*http.Cookie{
		"admin":    e.sessionCookie(t, s.Admin),
		"ownerA":   e.sessionCookie(t, s.OwnerA),
		"tenant":   e.sessionCookie(t, s.Tenant),
		"receptor": e.sessionCookie(t, s.Receptor),
	}

	cases := []struct {
		name   string
		method string
		path   string
		role   string
		want   int
	}{
		{"admin lista deptos", http.MethodGet, "/departamentos", "admin", http.StatusOK},
		{"inquilino lista deptos", http.MethodGet, "/departamentos", "tenant", http.StatusForbidden},
		{"receptor lista personas", http.MethodGet, "/personas", "receptor", http.StatusForbidden},
		{"receptor reportes", http.MethodGet, "/reportes", "receptor", http.StatusForbidden},
		{"inquilino consulta deudas", http.MethodGet, "/deudas", "tenant", http.StatusOK},
		{"inquilino genera deudas", http.MethodPost, "/deudas", "tenant", http.StatusForbidden},
		{"inquilino lista recibos", http.MethodGet, "/recibos", "tenant", http.StatusOK},
		{"admin crea depto", http.MethodPost, "/departamentos", "admin", http.StatusFound},
		{"propietario crea depto", http.MethodPost, "/departamentos", "ownerA", http.StatusForbidden},
		{"propietario edita depto", http.MethodPost, fmt.Sprintf("/departamentos/%d", s.DeptoB1.ID), "ownerA", http.StatusForbidden},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var form url.Values
			if tc.method == http.MethodPost {
				form = url.Values{}
			}
			rec := e.request(t, tc.method, tc.path, cookies[tc.role], form)
			if rec.Code != tc.want {
				t.Fatalf("%s %s como %s: status = %d, quería %d", tc.method, tc.path, tc.role, rec.Code, tc.want)
			}
		})
	}
}

func TestIDORReciboAjeno(t *testing.T) {
	e := newTestEnv(t)
	s := seedScenario(t, e)
	ownerA := e.sessionCookie(t, s.OwnerA)

	// Propio: debe poder verlo.
	rec := e.request(t, http.MethodGet, fmt.Sprintf("/recibos/%d", s.ReciboA1.ID), ownerA, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("recibo propio: status = %d, quería 200", rec.Code)
	}

	// Ajeno: detalle, PDF y pago deben rechazarse.
	rec = e.request(t, http.MethodGet, fmt.Sprintf("/recibos/%d", s.ReciboB1.ID), ownerA, nil)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("detalle ajeno: status = %d, quería 403", rec.Code)
	}
	rec = e.request(t, http.MethodGet, fmt.Sprintf("/recibos/%d/pdf", s.ReciboB1.ID), ownerA, nil)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("pdf ajeno: status = %d, quería 403", rec.Code)
	}
	rec = e.request(t, http.MethodPost, fmt.Sprintf("/recibos/%d/pagar", s.ReciboB1.ID), ownerA, url.Values{
		"forma_pago": {"E"},
	})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("pago ajeno: status = %d, quería 403", rec.Code)
	}
}

func TestSessionTampering(t *testing.T) {
	e := newTestEnv(t)

	bad := &http.Cookie{Name: "recibosgo_session", Value: "1:firma-invalida"}
	rec := e.request(t, http.MethodGet, "/dashboard", bad, nil)
	if rec.Code != http.StatusFound {
		t.Fatalf("sesión manipulada: status = %d, quería 302", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/login" {
		t.Fatalf("redirect = %q, quería /login", loc)
	}
}

func TestSQLInjectionSearch(t *testing.T) {
	e := newTestEnv(t)
	s := seedScenario(t, e)

	rec := e.request(t, http.MethodGet, "/departamentos?q=%27%20OR%201%3D1%20--", e.sessionCookie(t, s.Admin), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("búsqueda con SQLi: status = %d, quería 200", rec.Code)
	}
}

func TestXSSEscaped(t *testing.T) {
	e := newTestEnv(t)
	s := seedScenario(t, e)

	payload := "<script>alert(1)</script>"
	e.createDepto(t, payload, s.PersonaA.ID, nil, 0)

	rec := e.request(t, http.MethodGet, "/departamentos", e.sessionCookie(t, s.Admin), nil)
	body := rec.Body.String()
	if strings.Contains(body, payload) {
		t.Fatal("el payload XSS aparece sin escapar")
	}
	if !strings.Contains(body, "&lt;script&gt;alert(1)&lt;/script&gt;") {
		t.Fatal("el payload XSS escapado no aparece en el HTML")
	}
}
