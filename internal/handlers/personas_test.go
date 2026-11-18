package handlers_test

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"recibosgo/internal/models"
)

func TestPersonaCuenta(t *testing.T) {
	e := newTestEnv(t)
	s := seedScenario(t, e)

	rec := e.request(t, http.MethodGet, fmt.Sprintf("/personas/%d/cuenta", s.PersonaA.ID), e.sessionCookie(t, s.Admin), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("cuenta: status = %d, quería 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		s.PersonaA.Nombre,
		"A1", "A2",
		s.ReciboA1.NumeroRecibo,
		s.ReciboA2.NumeroRecibo,
		"1500.00",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("la cuenta no muestra %q", want)
		}
	}
	if strings.Contains(body, s.ReciboB1.NumeroRecibo) {
		t.Fatal("la cuenta muestra recibos de otra persona")
	}
}

func TestPersonaCuentaSoloAdmin(t *testing.T) {
	e := newTestEnv(t)
	s := seedScenario(t, e)

	rec := e.request(t, http.MethodGet, fmt.Sprintf("/personas/%d/cuenta", s.PersonaA.ID), e.sessionCookie(t, s.Receptor), nil)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("receptor: status = %d, quería 403", rec.Code)
	}
}

func TestPersonaCuentaPagoParcial(t *testing.T) {
	e := newTestEnv(t)
	s := seedScenario(t, e)
	admin := e.sessionCookie(t, s.Admin)

	e.request(t, http.MethodPost, fmt.Sprintf("/recibos/%d/pagar", s.ReciboA1.ID), admin, url.Values{
		"forma_pago": {"E"},
		"importe":    {"400"},
	})

	var carry models.Recibo
	if err := e.DB.Where("id_recibo_origen = ?", s.ReciboA1.ID).First(&carry).Error; err != nil {
		t.Fatalf("no se generó la deuda anterior: %v", err)
	}

	rec := e.request(t, http.MethodGet, fmt.Sprintf("/personas/%d/cuenta?estado=todos", s.PersonaA.ID), admin, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("cuenta: status = %d, quería 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Pago parcial") {
		t.Fatal("la cuenta no muestra el pago parcial")
	}
	if !strings.Contains(body, carry.NumeroRecibo) {
		t.Fatal("la cuenta no muestra la deuda anterior")
	}
	if !strings.Contains(body, "Deudas Anteriores") {
		t.Fatal("la cuenta no muestra el detalle de la deuda anterior")
	}
	if !strings.Contains(body, "1100.00") {
		t.Fatal("el total a cobrar no incluye el saldo de la deuda anterior")
	}
}

func TestPersonaCuentaInquilino(t *testing.T) {
	e := newTestEnv(t)
	s := seedScenario(t, e)

	rec := e.request(t, http.MethodGet, fmt.Sprintf("/personas/%d/cuenta", s.PersonaT.ID), e.sessionCookie(t, s.Admin), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("cuenta: status = %d, quería 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "T1") {
		t.Fatal("la cuenta no muestra la unidad del inquilino")
	}
	if !strings.Contains(body, s.ReciboT1.NumeroRecibo) {
		t.Fatal("la cuenta no muestra el recibo del inquilino")
	}
	if strings.Contains(body, s.ReciboA1.NumeroRecibo) {
		t.Fatal("la cuenta del inquilino muestra recibos de otra persona")
	}
	if !strings.Contains(body, "300.00") {
		t.Fatal("el total del inquilino no es 300.00")
	}
}
