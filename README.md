# RecibosGo - Sistema de Gestión de Expensas

Sistema web para administrar expensas y servicios de edificios de departamentos. Diseñado para 150 clientes y 5 administradores/operadores.

## 🚀 Características

- Gestión de departamentos, propietarios e inquilinos
- Administración de servicios (expensas comunes, extraordinarias)
- Generación de recibos con vista previa y PDF
- Deuda mensual por coeficiente (expensas + SAT/CISI) y consulta por unidad o persona
- Roles diferenciados: Administrador, Propietario, Inquilino, Operador
- Dashboard con métricas clave
- Modo oscuro/claro
- Diseño responsive (móvil/tablet/desktop)

## 📚 Documentación

La documentación técnica completa (arquitectura, modelo de datos, mapa de pantallas y migración) vive en [docs/](./docs/README.md). Cada pantalla tiene una especificación estándar en [docs/pantallas/](./docs/pantallas/) con ruta, handler, template, roles, campos y navegación.

## 🛠️ Stack Tecnológico

- **Backend**: Go 1.26+
- **Router**: Chi (`github.com/go-chi/chi/v5`)
- **ORM**: GORM
- **Base de Datos**: PostgreSQL 14+ (desarrollo y producción)
- **Frontend**: `html/template` + HTMX + TailwindCSS
- **PDF**: `github.com/go-pdf/fpdf`
- **Autenticación**: bcrypt + cookies de sesión

## 📦 Instalación

### Requisitos Previos

- Go 1.26 o superior
- PostgreSQL 14 o superior
- Git

### 1. Clonar el repositorio

```bash
git clone https://github.com/dlazarte/recibosGo.git
cd recibosGo
```

### 2. Crear la base de datos PostgreSQL

Con un PostgreSQL local:

```bash
sudo -u postgres psql -c "CREATE USER recibosgo WITH PASSWORD 'recibosgo';"
sudo -u postgres psql -c "CREATE DATABASE recibosgo OWNER recibosgo;"
```

O con Docker:

```bash
docker run -d --name recibosgo-db \
  -e POSTGRES_USER=recibosgo \
  -e POSTGRES_PASSWORD=recibosgo \
  -e POSTGRES_DB=recibosgo \
  -p 5432:5432 postgres:16
```

### 3. Configurar variables de entorno

```bash
export DATABASE_URL="postgres://recibosgo:recibosgo@localhost:5432/recibosgo?sslmode=disable"
export PORT=8090
export SESSION_SECRET="cambiar-en-produccion"
export ADMIN_EMAIL="admin@recibosgo.local"
export ADMIN_PASSWORD="admin123"
```

| Variable | Obligatoria | Default | Descripción |
|---|---|---|---|
| `DATABASE_URL` | sí | — | Cadena de conexión PostgreSQL (`postgres://usuario:clave@host:5432/base?sslmode=disable`) |
| `PORT` | no | `8090` | Puerto HTTP |
| `SESSION_SECRET` | no | `dev-secret-cambiar-en-produccion` | Firma HMAC de la cookie de sesión |
| `ADMIN_EMAIL` | no | `admin@recibosgo.local` | Email del usuario administrador inicial |
| `ADMIN_PASSWORD` | no | `admin123` | Contraseña del administrador inicial |
| `CSV_PATH` | no | `dptos_dueños_inquilinos.csv` | CSV usado por el backfill v2.1 |

> `DATABASE_URL` debe ser una URL de PostgreSQL. La aplicación crea y actualiza el esquema automáticamente al iniciar (AutoMigrate + migración aditiva v2.1); no hay paso manual de migración.

### 4. Ejecutar

```bash
go run ./cmd/server
```

Abrir [http://localhost:8090](http://localhost:8090) e iniciar sesión con `ADMIN_EMAIL` / `ADMIN_PASSWORD`.

### 5. Tests

```bash
go test ./...
```

## 🏗️ Estructura del Proyecto

```
recibosgo/
├── cmd/
│   ├── server/              # Punto de entrada HTTP
│   └── verify/              # Verificación de datos
├── internal/
│   ├── config/              # Configuración por variables de entorno
│   ├── database/            # Conexión PostgreSQL y migraciones
│   ├── handlers/            # Handlers HTTP (Chi)
│   ├── middleware/          # Autenticación y roles
│   ├── models/              # Modelos GORM
│   ├── parser/              # Parser de CSV/planillas
│   └── pdf/                 # Generación de PDFs
├── templates/               # Templates html/template
│   └── fragments/           # Fragmentos HTMX
├── static/                  # Imágenes y vendor (HTMX)
├── docs/                    # Arquitectura, modelo, pantallas, migraciones
├── data/                    # CSVs y scripts de importación
├── go.mod
├── go.sum
├── README.md
└── AGENTS.md
```

## 👥 Roles y Permisos

| Rol | Descripción | Acceso |
|---|---|---|
| Administrador | Gestión completa | Todo |
| Propietario | Dueño de departamentos | Sus deptos, recibos y deudas |
| Inquilino | Ocupante de unidad | Su unidad, recibos y deudas |
| Operador | Genera recibos y consultas | Generación, consultas, reportes |

## 🎨 Diseño

- Colores: azul corporativo (#1a56db) con acentos.
- Modo oscuro: soporte nativo.
- Responsive: TailwindCSS con breakpoints (sm/md/lg/xl).
- Menú móvil: hamburguesa ☰ sin animación.

## 📋 Tareas Sugeridas (Orden de Implementación)

- ✅ Setup inicial: estructura, configuración, conexión a DB
- ✅ Modelos: User, Departamento, Servicio, Recibo, Fecha
- ✅ Autenticación: login, logout, middleware, roles
- ✅ Dashboard: métricas por rol
- ✅ Departamentos: CRUD + listado con filtros
- ✅ Servicios: CRUD + listado
- ✅ Recibos: flujo de generación
- ✅ PDF: generación y descarga
- ✅ Reportes: básicos por período
- ✅ UI/UX: menú responsive, modo oscuro, estilos finales
- ✅ Deudas: generación mensual por coeficiente y consulta por unidad/persona
