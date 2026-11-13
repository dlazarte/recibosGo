# Arquitectura — RecibosGo

## Stack

| Capa | Tecnología |
|---|---|
| Lenguaje | Go 1.26+ (`go.mod` declara `go 1.26.5`) |
| Router | `github.com/go-chi/chi/v5` |
| ORM | `gorm.io/gorm` |
| DB | PostgreSQL 14+ (`github.com/jackc/pgx/v5/stdlib` + `gorm.io/driver/postgres`) |
| Auth | `bcrypt` + cookies de sesión con firma HMAC |
| Frontend | `html/template` estándar + HTMX (`static/vendor/htmx.min.js`) |
| PDF | `github.com/go-pdf/fpdf` |

## Estructura de carpetas

```
recibosgo/
├── cmd/server/main.go          # Punto de entrada: config, DB, templates, router
├── internal/
│   ├── config/config.go        # Variables de entorno (incluye CSV_PATH)
│   ├── database/
│   │   ├── database.go         # Conexión + AutoMigrate + migrateV21
│   │   └── backfill.go         # BackfillV21 desde CSV (idempotente)
│   ├── models/models.go        # Modelos GORM (User, Departamento, Persona, Servicio, Recibo, DetalleRecibo, Fecha)
│   ├── handlers/               # Handlers HTTP por dominio
│   │   ├── app.go              # App (DB, Templates), render/renderFrag, PageData
│   │   ├── auth.go             # Login/Logout
│   │   ├── dashboard.go
│   │   ├── departamentos.go
│   │   ├── personas.go         # CRUD Titulares (nuevo v2.1)
│   │   ├── servicios.go
│   │   ├── recibos.go          # Listado, creación, detalle, pago, item-row
│   │   ├── recibos_pdf.go
│   │   ├── reportes.go
│   │   ├── deudas.go           # Generación mensual y consulta de deudas
│   │   ├── funcs.go            # helpers de template (fmtMoney, fmtFecha, trimNumero, mul)
│   │   └── paginado.go         # Paginación (perPage = 25)
│   ├── middleware/middleware.go # Autenticación, sesión, RequireRole
│   ├── parser/
│   │   ├── parser.go           # Parser de nombres + ParseCSV (nuevo v2.1)
│   │   └── parser_test.go      # Tests del parser
│   └── pdf/recibo.go           # Generación de PDF de recibo
├── templates/                  # Páginas + fragments/ (HTMX)
├── static/                     # style.css, htmx.min.js, imágenes
├── recibos2026.sql             # Dump legacy (fuente de migración)
├── dptos_dueños_inquilinos.csv # CSV limpio para backfill
```

## Flujo de una request

1. `main.go` arma el router Chi con `middleware.Logging`.
2. Rutas públicas: `GET /login`, `POST /login`, `POST /logout`, `GET /static/*`.
3. Grupo autenticado con `auth.Authenticate` (cookie `recibosgo_session`).
4. Subgrupos por rol con `middleware.RequireRole(...)`.
5. El handler consulta con GORM y renderiza:
   - `app.render(w, r, "página", PageData)` → ejecuta `layout.html` + `página.html`.
   - `app.renderFrag(w, "fragmento", data)` → respuesta parcial para HTMX (cuando `HX-Request: true`).

### Carga de templates

`main.loadTemplates()` parsea UNA vez al iniciar:
- Cada página = `layout.html` + `{página}.html` + `ParseGlob(fragments/*.html)`.
- Cada fragmento suelto = `{fragmento}.html` + `ParseGlob(fragments/*.html)`.
- FuncMap: `fmtMoney`, `fmtFecha`, `fmtFechaD`, `trimNumero`, `mul`.

### Configuración (internal/config)

| Variable | Default | Uso |
|---|---|---|
| `PORT` | `8090` | Puerto HTTP |
| `DATABASE_URL` | `postgres://recibosgo:recibosgo@localhost:5432/recibosgo?sslmode=disable` | DSN PostgreSQL |
| `SESSION_SECRET` | `dev-secret-...` | Firma HMAC de la cookie |
| `ADMIN_EMAIL` | `admin@recibosgo.local` | Seed del admin |
| `ADMIN_PASSWORD` | `admin123` | Seed del admin |
| `CSV_PATH` | `dptos_dueños_inquilinos.csv` | CSV para backfill v2.1 |

> `DATABASE_URL` debe ser un DSN de PostgreSQL; la app no usa SQLite.

### Sesión y roles

- Cookie `recibosgo_session` = `{userID}:{HMAC-SHA256}`.
- `UserFromContext` expone `*models.User` con `Role`:
  - `administrador` — acceso total
  - `dueño` / `dueno` — propietario
  - `inquilino` — inquilino
  - `receptor` — operador (genera recibos)
- `RequireRole(roles...)` devuelve 403 si el rol no está en la lista.

### Login v2.1

El login sigue usando `users.email` para autenticación. Para encontrar los departamentos del usuario:
1. Buscar `persona.email = user.email`
2. Buscar `departamento` donde `id_dueño = persona.ID` O `id_inquilino = persona.ID`

## Reglas de diseño (resumen AGENTS.md)

- ✅ Templates en `templates/`, nunca HTML dentro de handlers.
- ✅ GORM para todo acceso a datos (nada de SQL crudo).
- ✅ PostgreSQL: usar GORM Migrator / AutoMigrate, sin dialectos específicos.
- ✅ Contraseñas con bcrypt.
- ✅ Importar drivers de DB solo en `internal/database`.
- ❌ NO renombrar tablas/columnas del esquema de `mp135.db` (respetar nombres 1:1).
- ✅ Modelos GORM con tags, handlers Chi, templates parseados una vez, middleware en `internal/middleware`, config por env.
- ✅ HTMX en vez de JS pesado; sin animaciones de menú.
- ✅ Modo oscuro/claro con Tailwind (CSS en `static/style.css`).
