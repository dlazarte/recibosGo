# AGENTS.md - RecibosGo v2.1

**Última actualización**: 27/08/2026

## 📚 Documentación

La documentación técnica (arquitectura, modelo de datos, mapa de pantallas y migraciones) está en [docs/](./docs/README.md). Cada pantalla sigue una plantilla de especificación en [docs/pantallas/](./docs/pantallas/) (ruta, handler, template, roles, campos, acciones, navegación). Consultar estos archivos antes de modificar el sistema.

## 📋 Especificaciones del Proyecto

### Objetivo
Sistema web para gestionar expensas y servicios de edificios de departamentos.
- **Usuarios**: 150 clientes (propietarios/inquilinos) + 5 administradores/operadores
- **Contexto**: Red interna del edificio

### Stack Tecnológico
Backend:
  - Lenguaje: Go 1.26+
  - Router: github.com/go-chi/chi/v5
  - ORM: GORM (gorm.io/gorm)
  - Auth: bcrypt (golang.org/x/crypto/bcrypt) + cookies de sesión
Base de Datos:
  - PostgreSQL 14+ (desarrollo y producción) vía DATABASE_URL
  - Driver: github.com/jackc/pgx/v5/stdlib + gorm.io/driver/postgres
  - Migraciones: AutoMigrate de GORM + migración aditiva v2.1

Frontend:
  - Templates: html/template estándar
  - Interactividad: HTMX (sin JavaScript complejo)
  - Estilos: TailwindCSS (responsive nativo)
  - Íconos: FontAwesome o Material Icons

PDF:
  - Biblioteca: github.com/go-pdf/fpdf

⚠️ REGLAS FUNDAMENTALES
NO hacer:
	❌ Escribir HTML dentro de los handlers → usar templates en templates/
	❌ SQL crudo a mano → usar GORM (db.Where, db.First, etc.)
	❌ Usar MySQL/Oracle/otro dialecto → solo PostgreSQL
	❌ Guardar contraseñas en texto plano → siempre bcrypt
	❌ Importar drivers de DB en capas de negocio → solo en internal/database
	❌ Renombrar tablas/columnas del esquema de mp135.db → respetar nombres 1:1
	❌ JavaScript pesado → priorizar HTMX
	❌ Animaciones en menú móvil → mantener simple

SÍ hacer:
	✅ Modelos GORM con tags (gorm:"primaryKey", gorm:"uniqueIndex", etc.)
	✅ Handlers Chi con r.Get, r.Post, r.Put, r.Delete
	✅ Templates html/template parseados una sola vez al iniciar la app
	✅ Middleware de autenticación y roles en internal/middleware
	✅ Configuración vía variables de entorno (DATABASE_URL, PORT, CSV_PATH)
	✅ AutoMigrate + migrateV21 para crear tablas/columnas faltantes al iniciar
	✅ BackfillV21 desde CSV (idempotente, safe to run N times)
	✅ Diseño responsive con TailwindCSS
	✅ Modo oscuro/claro con clases de Tailwind
	✅ Menú hamburguesa sin animación en móvil

### Modelo de Datos (v2.1 — implementado)

```
persona ─┐
         ├── departamento (FKs directas, sin tabla puente)
departamento ─┘
```

- `departamento`: `id`, `nombre` (unidad legible: 1A, SS2A), `numero` (legacy), `piso`, `propietario` (legacy), `telefono`, `email` (legacy), `id_dueño` (FK→persona, NOT NULL), `id_inquilino` (FK→persona, NULLABLE), `coeficiente` (% de prorrateo), `activo`
- `persona`: `id`, `nombre`, `telefono`, `email`, `tipo` (propietario|inquilino), `activo`
- `recibo`: `id`, `numero_recibo`, `fecha`, `periodo` (AAAA-MM de la deuda generada), `tipo` (expensas|servicios|extraordinarias), `id_departamento`, `id_persona` (nullable, titular al generar), `observaciones`, `total`, `total_con_interes`, `fecha_corte`, `fecha_pago`, `monto_pagado`
- `detallerecibo`: sin cambios
- `consumo_luz`: `id`, `periodo`, `id_departamento`, `lectura_anterior`, `lectura_actual`, `consumo_kwh`, `precio_kw`, `monto_consumo`, `servicio_comun`, `total`
- `servicio`: sin cambios
- `users`: sin cambios (solo auth)

Detalle completo en docs/modelo-de-datos.md y docs/migraciones.md.

## 🔁 Migración de planillas a la base de datos

Mientras la app no esté validada, las planillas (`ExpensasMP2026.xlsx`, pestaña mensual más reciente) son la fuente de verdad. Directivas obligatorias:

- **Mapear antes de migrar**: documentar pestaña/columna de origen y destino (`departamento`, `persona`, `recibo`, `detallerecibo`). No migrar por posición de fila; usar claves naturales estables (unidad, período, nombre).
- **Migraciones idempotentes**: cada script debe correr N veces sin duplicar datos. Buscar por clave natural antes de insertar. Referencia: `BackfillV21` y `migrateV21`.
- **No romper el esquema legacy**: no renombrar ni borrar columnas de `mp135.db` (regla 1:1). Todo cambio es aditivo.
- **Validar contra la planilla**: comparar totales por período/unidad (expensas, SAT/CISI, total) antes de dar una migración por buena. Registrar diferencias.
- **Coeficientes**: cargar `departamento.coeficiente` desde la columna de porcentaje de la planilla. Cálculo: `monto = total * coeficiente / 100`.
- **Blanqueo final**: solo cuando la app se compruebe fiable. Antes: backup de la base, migrar sobre una base vacía, validar totales y recién entonces promover.
- **No borrar la planilla**: conservarla como respaldo/auditoría hasta cerrar el blanqueo.

🎨 Diseño y Estilo
Paleta de Colores
	Fontes	/* Modo Claro */
	--color-primary: #1a56db;    /* Azul corporativo */
	--color-secondary: #0ea5e9;  /* Azul cielo */
	--color-success: #059669;    /* Verde pagos/activo */
	--color-danger: #dc2626;     /* Rojo deudas/inactivo */
	--color-warning: #d97706;    /* Ámbar alertas */
	--color-bg: #f8fafc;         /* Fondo claro */
	--color-surface: #ffffff;    /* Tarjetas */
	--color-text: #1e293b;       /* Texto principal */
	--color-text-light: #64748b; /* Texto secundario */

	Fontes	/* Modo Oscuro */
	--color-bg-dark: #0f172a;    /* Fondo oscuro */
	--color-surface-dark: #1e293b; /* Tarjetas oscuras */
	--color-text-dark: #f1f5f9;  /* Texto claro */
	--color-border-dark: #334155; /* Bordes */
	
	
### Estructura del Menú Móvil (Hamburguesa ☰)

	Breakpoint: < 768px (md)

	☰ Menú
	├── Dashboard
	├── Departamentos
	│   ├── Listar
	│   └── Crear
	├── Titulares (solo admin)
	├── Servicios
	│   ├── Listar
	│   └── Crear
	├── Recibos
	│   ├── Generar
	│   └── Historial
	├── Reportes
	└── Configuración
		├── Usuarios (solo admin)
		└── Mi Perfil

### Estructura del Menú Desktop (Laptop)
Categorías estables: Dashboard, Departamentos, Titulares, Servicios, Recibos, Reportes, Configuración

### Estructura de Pantallas y Funcionalidades

1. Login
Logo: static/images/logoMP135.webp
Campos: Email, Contraseña
Extras: ☑ Recordarme
Acción: Iniciar Sesión

2. Dashboard
Métricas (solo números):
  - Total recibos del mes
  - Total expensas
  - Total servicios separados
  - Total deuda
  - Departamentos activos

Acciones:
  - Botón "Ver todos los recibos" → redirige a /recibos

3. Listado Departamentos
Columnas:
  - Unidad (nombre, ej: 1A, 2B, 3C)
  - Propietario (dueño desde persona)
  - Estado

Acciones por fila:
  - ✏️ Editar → /departamentos/{id}/editar

Filtros:
  - Búsqueda por nombre/propietario
  - Botón "Nuevo Departamento"

4. Titulares / Personas (NUEVO v2.1)
Columnas:
  - Nombre
  - Tipo (propietario/inquilino)
  - Teléfono, Email
  - Estado

Acciones:
  - Crear persona
  - Editar persona
  - Búsqueda por nombre/teléfono/email

👥 Roles y Permisos
Matriz de Permisos

Pantalla/Acción		Admin	Propietario		Inquilino			Operador
Dashboard			✅ Todo	✅ Su resumen	✅ Su resumen	✅ Todo
Ver Departamentos	✅ Todos	✅ Solo suyos	❌				✅ Todos
Edit. Departamentos	✅		❌				❌				❌
Ver Titulares		✅		❌				❌				❌
Crear/Editar Personas✅		❌				❌				❌
Ver Servicios		✅ Todos	✅ Todos			✅ Todos			✅ Todos
Editar Servicios	✅		❌				❌				❌
Generar Recibos		✅		❌				❌				✅
Ver Recibos			✅ Todos	✅ Solo suyos	✅ Solo suyos	✅ Todos
Ver Deuda/Resumen	✅ Todo	✅ Su deuda		✅ Su deuda		✅ Todo
Consultas/Reportes	✅		✅ (solo suyo)	❌				✅ 
Gestionar Usuarios	✅		❌				❌				❌

Pantallas por Rol:
	Administrador:
		Dashboard completo (todas las métricas)
		CRUD Departamentos, Servicios, Usuarios
		Generar y gestionar recibos
		Reportes completos

	Propietario:
		Dashboard resumido (solo sus departamentos)
		Ver sus departamentos
		Ver recibos de sus departamentos
		Consultar deudas

	Inquilino:
		Dashboard resumido (solo su unidad)
		Ver recibos de su unidad
		Ver deuda consolidada

	Operador:
		Dashboard operativo
		Generar recibos (flujo rápido)
		Consultas de propietarios/inquilinos
		Reportes básicos