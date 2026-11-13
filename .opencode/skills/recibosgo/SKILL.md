
## 📝 Versión Final Mejorada del Skill

---
name: recibosgo
description: Use when working on the RecibosGo codebase (sistema de expensas, Go/Chi/GORM/HTMX). Covers departamentos, unidades (nombre), titulares/personas, recibos multi-unidad, migración desde recibos2026.sql, y el workflow de documentar pantallas siguiendo docs/pantallas/_plantilla.md.
---

# RecibosGo — Skill de desarrollo

Contexto y reglas para modificar el sistema de gestión de expensas RecibosGo (Go 1.22+, Chi, GORM, html/template + HTMX, SQLite dev / Postgres prod).

## Regla de oro

**Leer la documentación ANTES de tocar código.** La fuente de verdad está en `docs/` (ver `docs/README.md`). Cada pantalla tiene una especificación en `docs/pantallas/`. Si una pantalla no tiene spec, crear una copiando `docs/pantallas/_plantilla.md` y completándola ANTES de implementar.

## Orden de lectura obligatorio

1. `AGENTS.md` — reglas fundamentales, roles, menú, matriz de permisos.
2. `docs/arquitectura.md` — stack, flujo de request, carga de templates.
3. `docs/modelo-de-datos.md` — ERD actual y objetivo (v2.1), anomalías.
4. `docs/mapa-de-pantallas.md` — rutas, handlers, templates, roles.
5. `docs/migraciones.md` — reglas de backfill y migración.
6. `docs/pantallas/<pantalla>.md` — spec de la pantalla a modificar.

## Stack y convenciones

- **Router**: Chi (`r.Get`, `r.Post`, `r.Put`, `r.Delete`) en `cmd/server/main.go`.
- **ORM**: GORM siempre. Nada de SQL crudo. Drivers de DB SOLO en `internal/database`.
- **Templates**: `html/template` en `templates/`; nunca HTML dentro de handlers. Fragmentos HTMX en `templates/fragments/`.
- **Templates parseados una vez** en `main.loadTemplates()`: agregar páginas nuevas a `pages` y fragmentos a `fragments`.
- **FuncMap disponible**: `fmtMoney`, `fmtFecha`, `fmtFechaD`, `trimNumero`, `mul`.
- **Rendering**: `app.render(w, r, "página", PageData)` para páginas, `app.renderFrag(w, "fragmento", data)` cuando `HX-Request: true`.
- **Paginación**: `perPage = 25`, usar `paginar(r, total, "#target")` y el fragmento `paginador`.
- **Roles** (campo `User.Role`): `administrador`, `dueño`/`dueno`, `inquilino`, `receptor` (operador). Middleware `RequireRole(...)`.
- **Auth**: bcrypt + cookie `recibosgo_session` (HMAC), middleware en `internal/middleware`.
- **JS**: priorizar HTMX; sin animaciones de menú; sin JS pesado.

## Modelo de datos

### Actual (v2.0, NO renombrar/eliminar columnas — regla 1:1)

- `departamento.numero` guarda el **código legacy** (1..104), NO la unidad.
- `departamento.propietario` mezcla unidad+nombre en texto libre (ej. `1A DONADIO`, `SS2A`, `PB C ALDERETE`).
- `recibo` → FK a un único `id_departamento` (1 recibo = 1 unidad). `recibo.numero_recibo` = `REC-YYYY-NNNN` (numeración por conteo anual, frágil).
- `piso/telefono/email` casi siempre vacíos. 239 recibos con `id_departamento = 0` (huérfanos, no migrar).

### Objetivo (v2.1, cambios planificados en docs/)

- `departamento` gana `nombre` = unidad legible (1A, SS2A, PB C). Campos legacy se conservan pero quedan deprecados.
- Nuevas tablas `persona` (tipo propietario/inquilino) y `departamento_persona` (puente con rol, activo, fechas). Un titular con varias unidades = UNA sola fila en `persona`.
- `recibo` gana `id_persona` (nullable, titular al generar). `id_departamento` se conserva.
- Generación de recibos: **un recibo por unidad**, agrupando en un solo paso las unidades del MISMO titular; prohibido mezclar titulares.

## Workflow para crear/modificar una pantalla

1. Buscar la spec en `docs/pantallas/`. Si no existe, crearla desde `_plantilla.md`.
2. Identificar la **ruta** a agregar en `cmd/server/main.go` (grupo autenticado + `RequireRole`).
3. Handler en `internal/handlers/` siguiendo el patrón del dominio (ej. `departamentos.go`).
4. Template en `templates/` + fragmento HTMX en `templates/fragments/`.
5. Registrar página/fragmento en `main.loadTemplates()`.
6. Actualizar `docs/mapa-de-pantallas.md` y `docs/pantallas/<pantalla>.md`.
7. Completar la sección "Estado actual vs objetivo" de la spec.

Plantilla resumida (ver completa en `docs/pantallas/_plantilla.md`): identificación (ruta/handler/template/fragmentos/roles), datos, formulario, acciones y navegación, flujo mermaid, actual vs objetivo, pendientes.

## 🎨 Estilos y UI Responsive

### Clases Tailwind para Modo Oscuro
```html
<!-- Fondo -->
<div class="bg-gray-50 dark:bg-gray-900">
  
<!-- Tarjetas -->
<div class="bg-white dark:bg-gray-800">
  
<!-- Texto -->
<h1 class="text-gray-900 dark:text-white">
<p class="text-gray-600 dark:text-gray-300">

<!-- Bordes -->
<div class="border border-gray-200 dark:border-gray-700">
