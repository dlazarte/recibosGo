package handlers

import (
	"net/http"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"recibosgo/internal/middleware"
	"recibosgo/internal/models"
)

func (a *App) LoginPage(w http.ResponseWriter, r *http.Request) {
	tmpl, ok := a.Templates["login"]
	if !ok {
		http.Error(w, "template no encontrado", http.StatusInternalServerError)
		return
	}
	data := PageData{}
	if err := r.ParseForm(); err == nil {
		if r.FormValue("error") == "1" {
			data.Err = "Email o contraseña incorrectos"
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.ExecuteTemplate(w, "layout.html", data)
}

func (a *App) Login(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")

	var user models.User
	err := a.DB.Where("email = ?", email).First(&user).Error
	if err == gorm.ErrRecordNotFound {
		http.Redirect(w, r, "/login?error=1", http.StatusFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil {
		http.Redirect(w, r, "/login?error=1", http.StatusFound)
		return
	}

	mw := middleware.New(a.DB, a.SessionSecret)
	mw.SetSession(w, user.ID)
	http.Redirect(w, r, dashboardPath(user.Role), http.StatusFound)
}

func (a *App) Logout(w http.ResponseWriter, r *http.Request) {
	mw := middleware.New(a.DB, a.SessionSecret)
	mw.ClearSession(w)
	http.Redirect(w, r, "/login", http.StatusFound)
}

func dashboardPath(role string) string {
	switch role {
	case "administrador":
		return "/dashboard"
	case "dueño", "dueno":
		return "/dashboard"
	case "inquilino":
		return "/dashboard"
	case "receptor":
		return "/recibos/new"
	default:
		return "/dashboard"
	}
}
