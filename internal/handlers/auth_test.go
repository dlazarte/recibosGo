package handlers_test

import (
	"net/http"
	"net/url"
	"testing"
)

func TestLoginValid(t *testing.T) {
	e := newTestEnv(t)
	u := e.createUser(t, "owner@test.local", "dueno")

	rec := e.request(t, http.MethodPost, "/login", nil, url.Values{
		"email":    {u.Email},
		"password": {testPassword},
	})
	if rec.Code != http.StatusFound {
		t.Fatalf("login válido: status = %d, quería 302", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/dashboard" {
		t.Fatalf("redirect = %q, quería /dashboard", loc)
	}
	if len(rec.Result().Cookies()) == 0 {
		t.Fatal("login válido no devolvió cookie de sesión")
	}
}

func TestLoginInvalid(t *testing.T) {
	e := newTestEnv(t)
	u := e.createUser(t, "owner@test.local", "dueno")

	rec := e.request(t, http.MethodPost, "/login", nil, url.Values{
		"email":    {u.Email},
		"password": {"incorrecta"},
	})
	if rec.Code != http.StatusFound {
		t.Fatalf("login inválido: status = %d, quería 302", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/login?error=1" {
		t.Fatalf("redirect = %q, quería /login?error=1", loc)
	}
}

func TestProtectedRedirectsToLogin(t *testing.T) {
	e := newTestEnv(t)

	rec := e.request(t, http.MethodGet, "/dashboard", nil, nil)
	if rec.Code != http.StatusFound {
		t.Fatalf("sin sesión: status = %d, quería 302", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/login" {
		t.Fatalf("redirect = %q, quería /login", loc)
	}
}

func TestLogoutClearsSession(t *testing.T) {
	e := newTestEnv(t)
	u := e.createUser(t, "owner@test.local", "dueno")

	rec := e.request(t, http.MethodPost, "/logout", e.sessionCookie(t, u), nil)
	if rec.Code != http.StatusFound {
		t.Fatalf("logout: status = %d, quería 302", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/login" {
		t.Fatalf("redirect = %q, quería /login", loc)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) == 0 || cookies[0].MaxAge >= 0 {
		t.Fatal("logout no invalidó la cookie")
	}
}
