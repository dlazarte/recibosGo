# RecibosGo — Documentación Técnica

Documentación de arquitectura, modelo de datos, mapa de pantallas y estrategia de migración del sistema de gestión de expensas.

## ¿Para quién es esto?

Estos documentos sirven a dos audiencias:

1. **Humanos**: equipo que mantiene RecibosGo.
2. **Agentes de IA**: la información está estructurada (rutas, handlers, templates, roles, fragmentos HTMX) para que un agente pueda analizar y modificar el sistema sin adivinar.

## Índice

| Documento | Contenido |
|---|---|
| [arquitectura.md](./arquitectura.md) | Stack, capas, flujo de una request, reglas de diseño |
| [modelo-de-datos.md](./modelo-de-datos.md) | ERD actual y objetivo (v2.1), diccionario de tablas |
| [mapa-de-pantallas.md](./mapa-de-pantallas.md) | Diagrama de navegación y tabla de rutas |
| [migraciones.md](./migraciones.md) | Registro histórico de la migración legacy → v2.1 |
| [pantallas/_plantilla.md](./pantallas/_plantilla.md) | Plantilla estándar de especificación de pantalla |
| [pantallas/login.md](./pantallas/login.md) | Login |
| [pantallas/dashboard.md](./pantallas/dashboard.md) | Dashboard |
| [pantallas/departamentos.md](./pantallas/departamentos.md) | Departamentos (listado + alta/edición) |
| [pantallas/titulares.md](./pantallas/titulares.md) | Titulares / Personas (nuevo en v2.1) |
| [pantallas/servicios.md](./pantallas/servicios.md) | Servicios |
| [pantallas/recibos-nuevo.md](./pantallas/recibos-nuevo.md) | Generación de recibos (multi-unidad en v2.1) |
| [pantallas/recibos-listado.md](./pantallas/recibos-listado.md) | Listado/filtrado de recibos |
| [pantallas/recibo-detalle.md](./pantallas/recibo-detalle.md) | Detalle y pago de recibo |
| [pantallas/reportes.md](./pantallas/reportes.md) | Reportes por período |
| [pantallas/deudas.md](./pantallas/deudas.md) | Deudas: generación mensual y consulta por unidad/persona |

## Convenciones de lectura

- **Ruta** → `METHOD /path` tal como está registrada en `cmd/server/main.go`.
- **Handler** → `handlers.NombreFuncion` (archivo en `internal/handlers/`).
- **Template** → archivo en `templates/`; **fragmento** → `templates/fragments/`.
- **Roles** → `administrador`, `dueño` (alias `dueno`), `inquilino`, `receptor` (equivalente a "Operador" de AGENTS.md).
- Cada pantalla sigue la [plantilla](./pantallas/_plantilla.md). Las secciones marcadas como **v2.1 (objetivo)** describen cambios aún NO implementados; el resto es estado actual.

## Estado del proyecto

- **v2.1 (implementado)**: unidad con `nombre`, `persona` + FKs directas, `coeficiente` por unidad, `recibo.periodo`, generación multi-unidad y deuda mensual.
- **v2.2 (en curso)**: PostgreSQL como única fuente de verdad, generación de expensas/servicios/extraordinarias, pagos parciales, cuenta por persona y auto-registro.
- Los `*.db` de SQLite y los `*.csv` del repo son artefactos históricos; las planillas Excel solo se usan para validar la transición.

## Fuente de verdad

**PostgreSQL** (`recibosgo-db` local, contenedor `postgres` en la RPi, base `recibosgo`). Todo cambio de datos se hace ahí. La app no lee SQLite ni CSV al arrancar; ver `AGENTS.md`.
