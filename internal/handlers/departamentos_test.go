package handlers_test

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"recibosgo/internal/models"
)

func deptForm(s *scenario, inquilino string) url.Values {
	return url.Values{
		"nombre":       {"T1"},
		"numero":       {"T1"},
		"propietario":  {"OWNER B"},
		"telefono":     {""},
		"email":        {""},
		"coeficiente":  {"10"},
		"id_dueño":     {fmt.Sprintf("%d", s.PersonaB.ID)},
		"id_inquilino": {inquilino},
		"activo":       {"on"},
	}
}

func TestDepartamentoMuestraInquilino(t *testing.T) {
	e := newTestEnv(t)
	s := seedScenario(t, e)

	rec := e.request(t, http.MethodGet, "/departamentos?q=T1", e.sessionCookie(t, s.Admin), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("listado: status = %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Inquilino") {
		t.Fatal("falta la columna Inquilino")
	}
	if !strings.Contains(body, s.PersonaT.Nombre) {
		t.Fatalf("no aparece el inquilino %s", s.PersonaT.Nombre)
	}
}

func TestDepartamentoFormularioInquilinoSeleccionado(t *testing.T) {
	e := newTestEnv(t)
	s := seedScenario(t, e)

	rec := e.request(t, http.MethodGet, fmt.Sprintf("/departamentos?edit=%d", s.DeptoT1.ID), e.sessionCookie(t, s.Admin), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("edicion: status = %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `name="id_inquilino"`) {
		t.Fatal("falta el select id_inquilino")
	}
	if !strings.Contains(body, fmt.Sprintf(`value="%d" selected`, s.PersonaT.ID)) {
		t.Fatal("el inquilino actual no queda seleccionado")
	}
}

func TestDepartamentoUpdateConservaInquilino(t *testing.T) {
	e := newTestEnv(t)
	s := seedScenario(t, e)
	rec := e.request(t, http.MethodPost, fmt.Sprintf("/departamentos/%d", s.DeptoT1.ID),
		e.sessionCookie(t, s.Admin), deptForm(s, fmt.Sprintf("%d", s.PersonaT.ID)))
	if rec.Code != http.StatusFound {
		t.Fatalf("update: status = %d", rec.Code)
	}
	var d models.Departamento
	if err := e.DB.First(&d, s.DeptoT1.ID).Error; err != nil {
		t.Fatalf("leyendo depto: %v", err)
	}
	if d.IDInquilino == nil || *d.IDInquilino != s.PersonaT.ID {
		t.Fatalf("se perdió el inquilino: %v", d.IDInquilino)
	}
}

func TestDepartamentoUpdateLimpiaInquilino(t *testing.T) {
	e := newTestEnv(t)
	s := seedScenario(t, e)
	rec := e.request(t, http.MethodPost, fmt.Sprintf("/departamentos/%d", s.DeptoT1.ID),
		e.sessionCookie(t, s.Admin), deptForm(s, ""))
	if rec.Code != http.StatusFound {
		t.Fatalf("update: status = %d", rec.Code)
	}
	var d models.Departamento
	if err := e.DB.First(&d, s.DeptoT1.ID).Error; err != nil {
		t.Fatalf("leyendo depto: %v", err)
	}
	if d.IDInquilino != nil {
		t.Fatalf("el inquilino no se limpió: %v", d.IDInquilino)
	}
}

func TestDepartamentoCreateConInquilino(t *testing.T) {
	e := newTestEnv(t)
	s := seedScenario(t, e)
	form := deptForm(s, fmt.Sprintf("%d", s.PersonaT.ID))
	form.Set("nombre", "9Z")

	rec := e.request(t, http.MethodPost, "/departamentos", e.sessionCookie(t, s.Admin), form)
	if rec.Code != http.StatusFound {
		t.Fatalf("create: status = %d", rec.Code)
	}
	var d models.Departamento
	if err := e.DB.Where("nombre = ?", "9Z").First(&d).Error; err != nil {
		t.Fatalf("no se creó: %v", err)
	}
	if d.IDInquilino == nil || *d.IDInquilino != s.PersonaT.ID {
		t.Fatalf("inquilino no asignado: %v", d.IDInquilino)
	}
}
