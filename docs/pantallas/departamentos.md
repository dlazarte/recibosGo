# Pantalla: Departamentos

## 1. Identificación

| Campo | Valor |
|---|---|
| Ruta | `GET /departamentos` (listado), `POST /departamentos` (crear), `POST /departamentos/{id}` (editar) |
| Handler | `DepartamentosList`, `DepartamentoCreate`, `DepartamentoUpdate` (`internal/handlers/departamentos.go`) |
| Template | `templates/departamentos.html` |
| Fragmentos | `deptos_table.html`, `paginador.html` |
| Roles | administrador, dueño, dueno |
| Propósito | CRUD de unidades del edificio con búsqueda y paginación |
| Acceso | rol (middleware `RequireRole("administrador","dueño","dueno")`) |

## 2. Datos involucrados

- Modelo: `models.Departamento`
- Queries:
  - Listado: búsqueda `q` sobre `numero`, `piso`, `propietario`; orden por `numero ASC`; paginado (`perPage = 25`).
  - Editar: `GET /departamentos?edit={id}` carga el registro en `data.Edit`.

## 3. Formulario / Filtros

| Campo | Tipo | Obligatorio | Notas |
|---|---|---|---|
| Búsqueda `q` | text | no | numero, piso, propietario |
| `numero` | text | sí | **legacy**: código de la unidad |
| `piso` | text | no | |
| `propietario` | text | no | **legacy**: texto combinado |
| `telefono` | text | no | |
| `email` | email | no | |
| `nombre` | text | sí | unidad legible (1A, SS2A, PB C) |
| `coeficiente` | number | sí | % de prorrateo de expensas |
| `id_dueño` | select | no | FK → `persona` (tipo propietario) |
| `id_inquilino` | select | no | FK → `persona` (tipo inquilino); vacío = sin inquilino |
| `activo` | checkbox | no | default checked |

## 4. Acciones y navegación

| Acción | Destino | Método | HTMX | Target |
|---|---|---|---|---|
| Buscar | `/departamentos?q=` | GET | sí | `#deptos-table` |
| Crear | `/departamentos` | POST | no | — |
| Editar | `/departamentos/{id}` | POST | no | — |
| Editar (formulario) | `/departamentos?edit={id}` | GET | no | — |
| Ver recibos | `/departamentos/{id}/recibos` | — | — | planificado en AGENTS.md, no implementado |

## 5. Flujo

```mermaid
flowchart LR
  Deptos -->|Crear/Editar| Deptos
  Deptos -->|Nuevo| Form
  Form -->|POST| Deptos
```

## 6. Estado actual vs objetivo

- **Actual (v2.2):** unidad legible `nombre`, búsqueda/orden por `nombre`, `coeficiente`, y selectores de dueño e **inquilino**. El listado muestra columnas Unidad, Propietario, **Inquilino**, Coef. %, Estado y acciones Editar/Historial. `numero`, `piso`, `propietario`, `telefono`, `email` se conservan como legacy.
- Los vínculos dueño/inquilino se corrigen directamente en PostgreSQL (única fuente de verdad); no hay backfill desde CSV al reiniciar.
- **Objetivo:** acción "Asignar titulares" por fila → `/departamentos/{id}/titulares`.
  - Cargar `propietario` de la unidad desde `persona` (vía `departamento_persona`).
