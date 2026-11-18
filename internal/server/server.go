package server

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"time"

	"recibosgo/internal/handlers"
	"recibosgo/internal/middleware"
)

// NewRouter arma el router Chi de producción. root es la raíz del proyecto
// (para resolver templates y static).
func NewRouter(app *handlers.App, auth *middleware.AppCtx, root string) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Logging)

	r.Get("/login", app.LoginPage)
	r.Post("/login", app.Login)
	r.Post("/logout", app.Logout)

	registroLimit := middleware.NewRateLimiter(5, time.Hour)
	r.Get("/registro", app.RegistroPage)
	r.Post("/registro", registroLimit.Limit(app.RegistroCreate))

	staticFS := http.FileServer(http.Dir(filepath.Join(root, "static")))
	r.Get("/static/*", func(w http.ResponseWriter, r *http.Request) {
		http.StripPrefix("/static/", staticFS).ServeHTTP(w, r)
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
				r.Get("/{id}/historial", app.DepartamentoHistorial)
				r.Group(func(r chi.Router) {
					r.Use(middleware.RequireRole("administrador"))
					r.Post("/", app.DepartamentoCreate)
					r.Post("/{id}", app.DepartamentoUpdate)
				})
			})
			r.Route("/servicios", func(r chi.Router) {
				r.Get("/", app.ServiciosList)
				r.Post("/", app.ServicioCreate)
				r.Post("/{id}", app.ServicioUpdate)
			})
			r.Route("/personas", func(r chi.Router) {
				r.Get("/", app.PersonasList)
				r.Get("/{id}/cuenta", app.PersonaCuenta)
				r.Post("/", app.PersonaCreate)
				r.Post("/{id}", app.PersonaUpdate)
			})
			r.Get("/reportes", app.Reportes)
			r.Post("/registro/{id}/aprobar", app.RegistroAprobar)
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
					r.Post("/servicios", app.ServicioGenerar)
					r.Post("/extraordinarias", app.ExtraordinariaGenerar)
				})
			})
		})
	})

	return r
}

// LoadTemplates parsea páginas y fragmentos una sola vez. root es la raíz del
// proyecto (para resolver templates/).
func LoadTemplates(root string) (map[string]*template.Template, error) {
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
		"login", "registro", "dashboard", "departamentos", "departamento_historial", "servicios", "titulares", "persona_cuenta",
		"recibos_list", "recibo_create", "recibo_detail", "reportes", "deudas",
	}
	fragments := []string{
		"recibos_table", "deptos_table", "recibo_detalle", "reportes_content",
		"personas_table", "recibo_unidades", "deudas_table",
	}
	out := map[string]*template.Template{}
	templatesDir := filepath.Join(root, "templates")
	for _, page := range pages {
		t := template.New("layout.html").Funcs(funcs)
		t, err := t.ParseFiles(
			filepath.Join(templatesDir, "layout.html"),
			filepath.Join(templatesDir, page+".html"),
		)
		if err != nil {
			return nil, fmt.Errorf("parseando %s: %w", page, err)
		}
		t, err = t.ParseGlob(filepath.Join(templatesDir, "fragments", "*.html"))
		if err != nil {
			return nil, fmt.Errorf("parseando fragmentos para %s: %w", page, err)
		}
		out[page] = t
	}
	for _, frag := range fragments {
		t := template.New(frag).Funcs(funcs)
		t, err := t.ParseFiles(filepath.Join(templatesDir, "fragments", frag+".html"))
		if err != nil {
			return nil, fmt.Errorf("parseando fragmento %s: %w", frag, err)
		}
		t, err = t.ParseGlob(filepath.Join(templatesDir, "fragments", "*.html"))
		if err != nil {
			return nil, fmt.Errorf("parseando fragmentos para %s: %w", frag, err)
		}
		out[frag] = t
	}
	return out, nil
}
