package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"recibosgo/internal/config"
	"recibosgo/internal/database"
	"recibosgo/internal/handlers"
	"recibosgo/internal/middleware"
)

func main() {
	cfg := config.Load()

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("error conectando a la base: %v", err)
	}

	tmpls, err := loadTemplates()
	if err != nil {
		log.Fatalf("error cargando templates: %v", err)
	}

	app := handlers.NewApp(db, tmpls, cfg.SessionSecret)
	auth := middleware.New(db, cfg.SessionSecret)

	r := chi.NewRouter()
	r.Use(middleware.Logging)

	r.Get("/login", app.LoginPage)
	r.Post("/login", app.Login)
	r.Post("/logout", app.Logout)

	r.Get("/static/*", func(w http.ResponseWriter, r *http.Request) {
		fs := http.StripPrefix("/static/", http.FileServer(http.Dir("static")))
		fs.ServeHTTP(w, r)
	})

	r.Group(func(r chi.Router) {
		r.Use(auth.Authenticate)

		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/dashboard", http.StatusFound)
		})
		r.Get("/dashboard", app.Dashboard)

		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireRole("administrador", "dueño", "dueno"))
			r.Route("/departamentos", func(r chi.Router) {
				r.Get("/", app.DepartamentosList)
				r.Post("/", app.DepartamentoCreate)
				r.Post("/{id}", app.DepartamentoUpdate)
			})
			r.Route("/servicios", func(r chi.Router) {
				r.Get("/", app.ServiciosList)
				r.Post("/", app.ServicioCreate)
				r.Post("/{id}", app.ServicioUpdate)
			})
			r.Route("/personas", func(r chi.Router) {
				r.Get("/", app.PersonasList)
				r.Post("/", app.PersonaCreate)
				r.Post("/{id}", app.PersonaUpdate)
			})
			r.Get("/reportes", app.Reportes)
		})

		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireRole("administrador", "dueño", "dueno", "inquilino", "receptor"))
			r.Route("/recibos", func(r chi.Router) {
				r.Get("/", app.RecibosList)
				r.Get("/new", app.ReciboNew)
				r.Post("/", app.ReciboCreate)
				r.Get("/{id}", app.ReciboDetail)
				r.Get("/{id}/pdf", app.ReciboPDF)
				r.Post("/{id}/pagar", app.ReciboPagar)
			})
			r.Route("/deudas", func(r chi.Router) {
				r.Get("/", app.DeudasList)
				r.Group(func(r chi.Router) {
					r.Use(middleware.RequireRole("administrador", "receptor"))
					r.Post("/", app.DeudaGenerar)
				})
			})
		})
	})

	addr := ":" + cfg.Port
	log.Printf("RecibosGo escuchando en http://localhost%s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}

func loadTemplates() (map[string]*template.Template, error) {
	funcs := template.FuncMap{
		"fmtMoney":   handlers.FmtMoney,
		"fmtFecha":   handlers.FmtFecha,
		"fmtFechaD":  handlers.FmtFechaDate,
		"trimNumero": handlers.TrimNumero,
		"contains":   strings.Contains,
		"mul": func(a, b interface{}) float64 {
			af, err := strconv.ParseFloat(fmt.Sprintf("%v", a), 64)
			if err != nil {
				return 0
			}
			bf, err := strconv.ParseFloat(fmt.Sprintf("%v", b), 64)
			if err != nil {
				return 0
			}
			return af * bf
		},
	}
	pages := []string{
		"login", "dashboard", "departamentos", "servicios", "titulares",
		"recibos_list", "recibo_create", "recibo_detail", "reportes", "deudas",
	}
	fragments := []string{
		"recibos_table", "deptos_table", "recibo_detalle", "reportes_content",
		"personas_table", "recibo_unidades", "deudas_table",
	}
	out := map[string]*template.Template{}
	for _, page := range pages {
		t := template.New("layout.html").Funcs(funcs)
		t, err := t.ParseFiles(
			filepath.Join("templates", "layout.html"),
			filepath.Join("templates", page+".html"),
		)
		if err != nil {
			return nil, fmt.Errorf("parseando %s: %w", page, err)
		}
		t, err = t.ParseGlob(filepath.Join("templates", "fragments", "*.html"))
		if err != nil {
			return nil, fmt.Errorf("parseando fragmentos para %s: %w", page, err)
		}
		out[page] = t
	}
	for _, frag := range fragments {
		t := template.New(frag).Funcs(funcs)
		t, err := t.ParseFiles(filepath.Join("templates", "fragments", frag+".html"))
		if err != nil {
			return nil, fmt.Errorf("parseando fragmento %s: %w", frag, err)
		}
		t, err = t.ParseGlob(filepath.Join("templates", "fragments", "*.html"))
		if err != nil {
			return nil, fmt.Errorf("parseando fragmentos para %s: %w", frag, err)
		}
		out[frag] = t
	}
	return out, nil
}
