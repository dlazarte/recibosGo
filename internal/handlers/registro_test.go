package handlers_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"recibosgo/internal/models"
)

func registroForm(unidad, nombre, email, tel string) url.Values {
	return url.Values{
		"unidad":    {unidad},
		"nombre":    {nombre},
		"email":     {email},
		"telefono":  {tel},
		"password":  {"clave12345"},
		"password2": {"clave12345"},
	}
}

func TestRegistroAutoPorNombreYUnidad(t *testing.T) {
	e := newTestEnv(t)
	s := seedScenario(t, e)
	email := "nuevo1@test.local"

	rec := e.request(t, http.MethodPost, "/registro", nil, registroForm("A1", "OWNER A", email, ""))
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/login?registro=1" {
		t.Fatalf("registro: %d → %q", rec.Code, rec.Header().Get("Location"))
	}

	var u models.User
	if err := e.DB.Where("email = ?", email).First(&u).Error; err != nil {
		t.Fatalf("no se creó el usuario: %v", err)
	}
	if u.Role != "dueno" {
		t.Fatalf("rol = %q, quería dueno", u.Role)
	}
	if u.IDPersona == nil || *u.IDPersona != s.PersonaA.ID {
		t.Fatal("el usuario no quedó vinculado a su persona")
	}

	var reg models.Registro
	if err := e.DB.Where("email = ?", email).First(&reg).Error; err != nil || reg.Estado != "auto" {
		t.Fatalf("registro no quedó como auto: %+v %v", reg, err)
	}

	login := e.request(t, http.MethodPost, "/login", nil, url.Values{"email": {email}, "password": {"clave12345"}})
	if login.Code != http.StatusFound {
		t.Fatalf("login con la cuenta nueva: status = %d", login.Code)
	}
	cookies := login.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("login no devolvió cookie")
	}
	dash := e.request(t, http.MethodGet, "/dashboard", cookies[0], nil)
	if !strings.Contains(dash.Body.String(), "A1") {
		t.Fatal("el usuario registrado no ve sus unidades en el dashboard")
	}
}

func TestRegistroAutoPorTelefono(t *testing.T) {
	e := newTestEnv(t)
	s := seedScenario(t, e)
	if err := e.DB.Model(&models.Persona{}).Where("id = ?", s.PersonaA.ID).
		Update("telefono", "3814000001").Error; err != nil {
		t.Fatalf("seteando teléfono: %v", err)
	}
	email := "nuevo2@test.local"

	rec := e.request(t, http.MethodPost, "/registro", nil,
		registroForm("", "DESCONOCIDO", email, "3814000001"))
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/login?registro=1" {
		t.Fatalf("registro por teléfono: %d → %q", rec.Code, rec.Header().Get("Location"))
	}
	var u models.User
	if err := e.DB.Where("email = ?", email).First(&u).Error; err != nil || u.Role != "dueno" {
		t.Fatalf("usuario por teléfono: %+v %v", u, err)
	}
}

func TestRegistroSinMatchQuedaPendiente(t *testing.T) {
	e := newTestEnv(t)
	s := seedScenario(t, e)
	email := "nuevo3@test.local"

	rec := e.request(t, http.MethodPost, "/registro", nil,
		registroForm("A1", "PERSONA QUE NO EXISTE", email, ""))
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/registro?pendiente=1" {
		t.Fatalf("registro sin match: %d → %q", rec.Code, rec.Header().Get("Location"))
	}
	var count int64
	e.DB.Model(&models.User{}).Where("email = ?", email).Count(&count)
	if count != 0 {
		t.Fatal("se creó un usuario sin match")
	}
	var reg models.Registro
	if err := e.DB.Where("email = ?", email).First(&reg).Error; err != nil || reg.Estado != "pendiente" {
		t.Fatalf("no quedó pendiente: %+v %v", reg, err)
	}

	// El admin ve la notificación en el dashboard.
	dash := e.request(t, http.MethodGet, "/dashboard", e.sessionCookie(t, s.Admin), nil)
	if !strings.Contains(dash.Body.String(), "pendientes") || !strings.Contains(dash.Body.String(), email) {
		t.Fatal("el admin no ve el registro pendiente")
	}
}

func TestRegistroEmailDuplicado(t *testing.T) {
	e := newTestEnv(t)
	s := seedScenario(t, e)

	var antes int64
	e.DB.Model(&models.User{}).Count(&antes)
	rec := e.request(t, http.MethodPost, "/registro", nil,
		registroForm("A1", "OWNER A", s.Admin.Email, ""))
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/registro?error=1" {
		t.Fatalf("email duplicado: %d → %q", rec.Code, rec.Header().Get("Location"))
	}
	var despues int64
	e.DB.Model(&models.User{}).Count(&despues)
	if despues != antes {
		t.Fatalf("se creó un usuario duplicado: %d → %d", antes, despues)
	}
}

func TestRegistroRateLimit(t *testing.T) {
	e := newTestEnv(t)
	form := url.Values{"password": {"corta"}, "password2": {"distinta"}}
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/registro", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.RemoteAddr = "198.51.100.7:1234"
		rec := httptest.NewRecorder()
		e.Router.ServeHTTP(rec, req)
		if rec.Code == http.StatusTooManyRequests {
			t.Fatalf("intento %d bloqueado antes de tiempo", i+1)
		}
	}
	req := httptest.NewRequest(http.MethodPost, "/registro", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.RemoteAddr = "198.51.100.7:1234"
	rec := httptest.NewRecorder()
	e.Router.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("6to intento: status = %d, quería 429", rec.Code)
	}
}

func TestRegistroAprobar(t *testing.T) {
	e := newTestEnv(t)
	s := seedScenario(t, e)
	email := "nuevo4@test.local"
	e.request(t, http.MethodPost, "/registro", nil,
		registroForm("A1", "SIN COINCIDENCIA", email, ""))

	var reg models.Registro
	if err := e.DB.Where("email = ?", email).First(&reg).Error; err != nil {
		t.Fatalf("no hay pendiente: %v", err)
	}
	rec := e.request(t, http.MethodPost, fmt.Sprintf("/registro/%d/aprobar", reg.ID),
		e.sessionCookie(t, s.Admin), url.Values{"rol": {"inquilino"}})
	if rec.Code != http.StatusFound {
		t.Fatalf("aprobar: status = %d, quería 302", rec.Code)
	}
	var u models.User
	if err := e.DB.Where("email = ?", email).First(&u).Error; err != nil || u.Role != "inquilino" {
		t.Fatalf("usuario aprobado: %+v %v", u, err)
	}
	e.DB.First(&reg, reg.ID)
	if reg.Estado != "aprobado" {
		t.Fatalf("estado = %q, quería aprobado", reg.Estado)
	}
}

func TestRegistroAprobarSoloAdmin(t *testing.T) {
	e := newTestEnv(t)
	s := seedScenario(t, e)
	rec := e.request(t, http.MethodPost, "/registro/1/aprobar", e.sessionCookie(t, s.Receptor), url.Values{})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("receptor: status = %d, quería 403", rec.Code)
	}
}
