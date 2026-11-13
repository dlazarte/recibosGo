package handlers

import (
	"html/template"
	"net/http"
	"net/url"

	"gorm.io/gorm"

	"recibosgo/internal/middleware"
	"recibosgo/internal/models"
)

type App struct {
	DB            *gorm.DB
	Templates     map[string]*template.Template
	SessionSecret string
}

type PageData struct {
	User        *models.User
	IsAdmin     bool
	IsOwner     bool
	IsTenant    bool
	IsReceptor  bool
	Data        interface{}
	Msg         string
	Err         string
}

func NewApp(db *gorm.DB, tmpls map[string]*template.Template, sessionSecret string) *App {
	return &App{DB: db, Templates: tmpls, SessionSecret: sessionSecret}
}

func (a *App) render(w http.ResponseWriter, r *http.Request, page string, data PageData) {
	if user, ok := middleware.UserFromContext(r.Context()); ok {
		data.User = user
		data.IsAdmin = user.Role == "administrador"
		data.IsOwner = user.Role == "dueño" || user.Role == "dueno"
		data.IsTenant = user.Role == "inquilino"
		data.IsReceptor = user.Role == "receptor"
	}
	tmpl, ok := a.Templates[page]
	if !ok {
		http.Error(w, "template no encontrado: "+page, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (a *App) renderFrag(w http.ResponseWriter, name string, data interface{}) {
	tmpl, ok := a.Templates[name]
	if !ok {
		http.Error(w, "template no encontrado: "+name, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func redirectLogin(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/login", http.StatusFound)
}

func urlEscape(s string) string {
	return url.QueryEscape(s)
}
