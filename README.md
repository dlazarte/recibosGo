# RecibosGo - Sistema de Gestión de Expensas

Sistema web para administrar expensas y servicios de edificios. Diseñado para 150 clientes y 5 administradores.

## 🚀 Características

- Gestión de departamentos, propietarios e inquilinos
- Administración de servicios (expensas comunes, extraordinarias)
- Generación de recibos con vista previa y PDF
- Roles diferenciados: Administrador, Propietario, Inquilino, Operador
- Dashboard con métricas clave
- Modo oscuro/claro
- Diseño responsive (móvil/tablet/desktop)

## 📚 Documentación

La documentación técnica completa (arquitectura, modelo de datos, mapa de pantallas y migración) vive en [docs/](./docs/README.md). Cada pantalla tiene una especificación estándar en [docs/pantallas/](./docs/pantallas/) con ruta, handler, template, roles, campos y navegación.

## 🛠️ Stack Tecnológico

- **Backend**: Go 1.22+
- **Router**: Chi (`github.com/go-chi/chi/v5`)
- **ORM**: GORM
- **Base de Datos**: PostgreSQL ( desarrollo y producción)
- **Frontend**: `html/template` + HTMX + TailwindCSS
- **PDF**: `github.com/go-pdf/fpdf`
- **Autenticación**: bcrypt + cookies de sesión

## 📦 Instalación

### Requisitos Previos

- Go 1.22 o superior
- Git

### Pasos

1. Clonar el repositorio:
```bash
git clone <url-del-repositorio>
cd recibosgo

# Puerto (default: 8090)
export PORT=8090
# Para producción con PostgreSQL
export DATABASE_URL="postgres://user:pass@localhost/dbname"

🏗️ Estructura del Proyecto
recibosgo/
├── cmd/
│   └── server/
│       └── main.go              # Punto de entrada
├── internal/
│   ├── config/                  # Configuración (env vars)
│   ├── database/                # Conexión y migraciones
│   ├── models/                  # Modelos GORM
│   ├── handlers/                # Handlers HTTP
│   ├── middleware/              # Auth y roles
│   └── pdf/                     # Generación de PDFs
├── templates/                   # Templates HTML
├── static/                      # CSS, imágenes, logos
│   ├── css/
│   ├── images/
│   └── js/
├── go.mod
├── go.sum
├── README.md
└── AGENTS.md

👥 Roles y Permisos
Rol	Descripción	Acceso
Administrador	Gestión completa	Todo
Propietario	Dueño de departamentos	Sus deptos, recibos y deudas
Inquilino	Ocupante de unidad	Su unidad, recibos y deudas
Operador	Genera recibos y consultas	Generación, consultas, reportes

🎨 Diseño
Colores: Azul corporativo (#1a56db) con acentos

Modo oscuro: Soporte nativo

Responsive: TailwindCSS con breakpoints (sm/md/lg/xl)

Menú móvil: Hamburguesa ☰ sin animación

Tareas Sugeridas (Orden de Implementación)
✅ Setup inicial: Estructura, configuración, conexión a DB
✅ Modelos: User, Departamento, Servicio, Recibo, Fecha
✅ Autenticación: Login, logout, middleware, roles
✅ Dashboard: Métricas por rol
✅ Departamentos: CRUD + listado con filtros
✅ Servicios: CRUD + listado
✅ Recibos: Flujo 3 pasos (seleccionar → agregar → confirmar)
✅ PDF: Generación y descarga
✅ Reportes: Básicos por período
✅ UI/UX: Menú responsive, modo oscuro, estilos finales