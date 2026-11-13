# Pantalla: Titulares / Personas (v2.1 — implementada)

## 1. Identificación

| Campo | Valor |
|---|---|
| Ruta | `GET /personas`, `POST /personas`, `POST /personas/{id}` |
| Handler | `PersonasList`, `PersonaCreate`, `PersonaUpdate` (`internal/handlers/personas.go`) |
| Template | `templates/titulares.html` |
| Fragmentos | `personas_table.html`, `paginador.html` |
| Roles | administrador (gestión) |

## 2. Datos involucrados

- Modelo: `models.Persona`
- Queries:
  - Listado `persona` ordenado por `nombre`, con búsqueda por `nombre`, `telefono`, `email`.
  - `persona.tipo`: `propietario | inquilino`.

## 3. Formulario (alta/edición de persona)

| Campo | Tipo | Obligatorio | Notas |
|---|---|---|---|
| `nombre` | text | sí | |
| `telefono` | text | no | |
| `email` | email | no | |
| `tipo` | select | sí | propietario / inquilino |
| `activo` | checkbox | no | default checked |

## 4. Acciones y navegación

| Acción | Destino | Método | HTMX | Target |
|---|---|---|---|---|
| Buscar | `/personas?q=` | GET | sí | `#personas-table` |
| Crear | `/personas` | POST | no | — |
| Editar | `/personas/{id}` | POST | no | — |

## 5. Estado actual

- **Implementado (v2.1):** CRUD completo de personas con búsqueda y paginación.
- El vínculo persona ↔ departamento se realiza desde la pantalla de Departamentos (select de dueño).
