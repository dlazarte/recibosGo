# Modelo de Datos — RecibosGo

## Estado actual (v2.1 — implementado)

```mermaid
erDiagram
    persona ||--o{ departamento : "id_dueño"
    persona ||--o{ departamento : "id_inquilino"
    departamento ||--o{ recibo : "id_departamento"
    persona ||--o{ recibo : "id_persona"
    recibo ||--o{ detallerecibo : "id_recibo"
    servicio ||--o{ detallerecibo : "id_servicio"
    departamento ||--o{ consumo_luz : "id_departamento"

    departamento {
        int id PK
        string nombre "unidad legible: 1A, SS2A, PB C"
        string numero "legacy, deprecado"
        string piso "legacy"
        string propietario "legacy, deprecado"
        string telefono "legacy"
        string email "legacy"
        int id_dueño FK "NOT NULL → persona"
        int id_inquilino FK "NULLABLE → persona"
        float coeficiente "0-100, % de participación"
        bool activo
    }
    persona {
        int id PK
        string nombre
        string telefono
        string email
        string tipo "propietario | inquilino"
        bool activo
    }
    recibo {
        int id PK
        string numero_recibo "REC-YYYY-NNNN"
        date fecha
        string periodo "YYYY-MM de la deuda generada; '' si es manual"
        string tipo "expensas | servicios | extraordinarias"
        int id_departamento FK
        int id_persona FK "nullable, titular al generar"
        string observaciones
        float total
    }
    servicio {
        int id PK
        string nombre
        float precio_unitario
        bool activo
    }
    detallerecibo {
        int id PK
        int id_recibo FK
        int id_servicio FK
        float precio
        int cantidad
        string descripcion
        string forma_pago "E | T | ''"
    }
    consumo_luz {
        int id PK
        string periodo "YYYY-MM"
        int id_departamento FK
        float lectura_anterior
        float lectura_actual
        float consumo_kwh
        float precio_kw
        float monto_consumo
        float servicio_comun
        float total
    }
```

### Datos actuales (mp135.db)

| Tabla | Filas | Observaciones |
|---|---|---|
| `departamento` | 104 | 104 unidades registradas |
| `persona` | ~88 | Creadas desde CSV (propietarios + inquilinos) |
| `recibo` | 5.690 | 239 con `id_departamento = 0` (huérfanos) |
| `detallerecibo` | 7.860 | |
| `users` | 1+ | Solo auth (admin seed) |

### Diccionario de tablas

**`departamento`** — una fila por unidad física del edificio.
- `id` PK
- `nombre` — unidad legible (`1A`, `2B`, `SS2A`, `PB C`). Campo de identificación principal.
- `numero` — código legacy; conservado (regla 1:1) pero deprecado.
- `piso`, `propietario`, `telefono`, `email` — legacy; `propietario` deja de ser fuente primaria.
- `id_dueño` FK → `persona.id` — propietario de la unidad (NOT NULL).
- `id_inquilino` FK → `persona.id` — inquilino actual (NULLABLE).
- `activo` — baja lógica.

**`persona`** — titular normalizado (propietario o inquilino).
- `id` PK, `nombre`, `telefono`, `email`
- `tipo` — `propietario | inquilino` (clasificación principal de la persona).
- `activo`

**`recibo`** — documento por unidad.
- `id_departamento` FK (1 recibo = 1 unidad; se conserva).
- `id_persona` FK nullable — titular al momento de generar. NULL en recibos legacy.
- El resto no cambia.

**`detallerecibo`** — líneas de un recibo. Sin cambios.

**`consumo_luz`** — lectura y cálculo de luz EDET por unidad y período.
- `periodo` + `id_departamento` — clave única.
- `lectura_anterior`, `lectura_actual`, `consumo_kwh` — datos del medidor.
- `precio_kw`, `monto_consumo`, `servicio_comun`, `total` — resultado de la generación de servicios.

**`servicio`** — catálogo de servicios. Sin cambios.

**`users`** — solo autenticación. Sin cambios.

### Relaciones clave

| Relación | Tipo | Descripción |
|---|---|---|
| `persona` → `departamento` (id_dueño) | 1:N | Un propietario puede tener múltiples unidades |
| `persona` → `departamento` (id_inquilino) | 1:N | Un inquilino puede estar en múltiples unidades |
| `departamento` → `recibo` | 1:N | Cada recibo pertenece a una unidad |
| `persona` → `recibo` (id_persona) | 1:N | Titular al momento de generar (nullable) |

### Mapeo legacy → v2.1

| Fuente (recibos2026.sql / CSV) | Destino (v2.1) | Regla |
|---|---|---|
| CSV `DEPTO` | `departamento.nombre` | Copia directa |
| CSV `PROPIETARIO` | `persona.nombre` + `departamento.id_dueño` | Crear persona, vincular |
| CSV `INQUILINO` | `persona.nombre` + `departamento.id_inquilino` | Crear persona si existe, vincular |
| `departamento.propietario` | Legacy conservado | No se borra (regla 1:1) |
| `departamento.numero` | Legacy conservado | No se borra (regla 1:1) |

Detalle del parser en [migraciones.md](./migraciones.md).
