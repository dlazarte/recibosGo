package handlers_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"recibosgo/internal/handlers"
	"recibosgo/internal/middleware"
	"recibosgo/internal/models"
	"recibosgo/internal/server"
)

const (
	defaultTestDatabaseURL = "postgres://recibosgo:recibosgo@localhost:5432/recibosgo_test?sslmode=disable"
	testSecret             = "test-secret"
	testPassword           = "pass1234"
)

func testDatabaseURL() string {
	if v := os.Getenv("TEST_DATABASE_URL"); v != "" {
		return v
	}
	return defaultTestDatabaseURL
}

type testEnv struct {
	DB     *gorm.DB
	App    *handlers.App
	Auth   *middleware.AppCtx
	Router *chi.Mux
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	db, err := gorm.Open(postgres.Open(testDatabaseURL()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("conectando a la base de test: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("obteniendo sql.DB: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	if err := db.AutoMigrate(
		&models.User{}, &models.Persona{}, &models.Departamento{},
		&models.Servicio{}, &models.Recibo{}, &models.DetalleRecibo{},
		&models.ConsumoLuz{},
	); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	for _, table := range []string{"detallerecibo", "recibo", "consumo_luz", "departamento", "persona", "servicio", "users"} {
		if err := db.Exec("TRUNCATE TABLE " + table + " RESTART IDENTITY CASCADE").Error; err != nil {
			t.Fatalf("truncate %s: %v", table, err)
		}
	}

	tmpls, err := server.LoadTemplates("../..")
	if err != nil {
		t.Fatalf("cargando templates: %v", err)
	}
	app := handlers.NewApp(db, tmpls, testSecret)
	auth := middleware.New(db, testSecret)

	return &testEnv{
		DB:     db,
		App:    app,
		Auth:   auth,
		Router: server.NewRouter(app, auth, "../.."),
	}
}

func (e *testEnv) createUser(t *testing.T, email, role string) models.User {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	u := models.User{Email: email, Password: string(hash), Role: role}
	if err := e.DB.Create(&u).Error; err != nil {
		t.Fatalf("creando user %s: %v", email, err)
	}
	return u
}

func (e *testEnv) createPersona(t *testing.T, nombre, email, tipo string) models.Persona {
	t.Helper()
	p := models.Persona{Nombre: nombre, Email: email, Tipo: tipo, Activo: true}
	if err := e.DB.Create(&p).Error; err != nil {
		t.Fatalf("creando persona %s: %v", nombre, err)
	}
	return p
}

func (e *testEnv) createDepto(t *testing.T, nombre string, idDueño uint, idInquilino *uint, coeficiente float64) models.Departamento {
	t.Helper()
	d := models.Departamento{
		Nombre:      nombre,
		Numero:      nombre,
		IDDueño:     idDueño,
		IDInquilino: idInquilino,
		Coeficiente: coeficiente,
		Activo:      true,
	}
	if err := e.DB.Create(&d).Error; err != nil {
		t.Fatalf("creando depto %s: %v", nombre, err)
	}
	return d
}

func (e *testEnv) createServicio(t *testing.T, nombre string) models.Servicio {
	t.Helper()
	s := models.Servicio{Nombre: nombre, Activo: true}
	if err := e.DB.Create(&s).Error; err != nil {
		t.Fatalf("creando servicio %s: %v", nombre, err)
	}
	return s
}

func (e *testEnv) createRecibo(t *testing.T, idDepto uint, idPersona *uint, tipo, periodo string, total float64, detalles []models.DetalleRecibo) models.Recibo {
	t.Helper()
	r := models.Recibo{
		NumeroRecibo:   fmt.Sprintf("REC-TEST-%d", time.Now().UnixNano()),
		Fecha:          models.Fecha{Time: time.Now()},
		Periodo:        periodo,
		Tipo:           tipo,
		IDDepartamento: idDepto,
		IDPersona:      idPersona,
		Total:          total,
		Detalles:       detalles,
	}
	if err := e.DB.Create(&r).Error; err != nil {
		t.Fatalf("creando recibo: %v", err)
	}
	return r
}

func (e *testEnv) sessionCookie(t *testing.T, u models.User) *http.Cookie {
	t.Helper()
	rec := httptest.NewRecorder()
	e.Auth.SetSession(rec, u.ID)
	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("no se generó cookie de sesión")
	}
	return cookies[0]
}

func (e *testEnv) request(t *testing.T, method, path string, cookie *http.Cookie, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}
	req := httptest.NewRequest(method, path, body)
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	e.Router.ServeHTTP(rec, req)
	return rec
}

type scenario struct {
	Admin, OwnerA, OwnerB, Tenant, Receptor        models.User
	PersonaA, PersonaB, PersonaT                   models.Persona
	DeptoA1, DeptoA2, DeptoB1, DeptoT1             models.Departamento
	ServicioExpensas, ServicioSatCisi, ServicioLuz models.Servicio
	ReciboA1, ReciboA2, ReciboB1, ReciboT1         models.Recibo
}

func seedScenario(t *testing.T, e *testEnv) *scenario {
	t.Helper()
	s := &scenario{}
	s.Admin = e.createUser(t, "admin@test.local", "administrador")
	s.OwnerA = e.createUser(t, "ownera@test.local", "dueno")
	s.OwnerB = e.createUser(t, "ownerb@test.local", "dueno")
	s.Tenant = e.createUser(t, "tenant@test.local", "inquilino")
	s.Receptor = e.createUser(t, "receptor@test.local", "receptor")

	s.PersonaA = e.createPersona(t, "OWNER A", "ownera@test.local", "propietario")
	s.PersonaB = e.createPersona(t, "OWNER B", "ownerb@test.local", "propietario")
	s.PersonaT = e.createPersona(t, "TENANT", "tenant@test.local", "inquilino")

	s.DeptoA1 = e.createDepto(t, "A1", s.PersonaA.ID, nil, 50)
	s.DeptoA2 = e.createDepto(t, "A2", s.PersonaA.ID, nil, 25)
	s.DeptoB1 = e.createDepto(t, "B1", s.PersonaB.ID, nil, 25)
	inqID := s.PersonaT.ID
	s.DeptoT1 = e.createDepto(t, "T1", s.PersonaB.ID, &inqID, 10)

	s.ServicioExpensas = e.createServicio(t, "Expensas Comunes. Pago Final")
	s.ServicioSatCisi = e.createServicio(t, "Servicios. Pago Final")
	s.ServicioLuz = e.createServicio(t, "Luz (EDET)")

	s.ReciboA1 = e.createRecibo(t, s.DeptoA1.ID, &s.PersonaA.ID, "expensas", "2026-01", 1000, []models.DetalleRecibo{
		{IDServicio: s.ServicioExpensas.ID, Precio: 1000, Cantidad: 1},
	})
	s.ReciboA2 = e.createRecibo(t, s.DeptoA2.ID, &s.PersonaA.ID, "expensas", "2026-01", 500, []models.DetalleRecibo{
		{IDServicio: s.ServicioExpensas.ID, Precio: 500, Cantidad: 1},
	})
	s.ReciboB1 = e.createRecibo(t, s.DeptoB1.ID, &s.PersonaB.ID, "expensas", "2026-01", 2000, []models.DetalleRecibo{
		{IDServicio: s.ServicioExpensas.ID, Precio: 2000, Cantidad: 1},
	})
	s.ReciboT1 = e.createRecibo(t, s.DeptoT1.ID, &s.PersonaT.ID, "servicios", "2026-01", 300, []models.DetalleRecibo{
		{IDServicio: s.ServicioLuz.ID, Precio: 300, Cantidad: 1},
	})
	return s
}
