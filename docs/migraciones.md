# Migración — de legacy (recibos2026.sql) a v2.1

## Contexto

- La fuente de producción es el dump legacy **`recibos2026.sql`** (tablas `categorias`, `clientes`, `detallefactura`, `facturas`).
- La base real en producción es `mp135.db` / `recibos.db`, que ya contiene las tablas del sistema actual (`departamento`, `servicio`, `recibo`, `detallerecibo`, `users`) creadas por AutoMigrate + importación manual.
- La v2.1 es **aditiva**: no se renombra ni elimina ninguna columna/tabla existente (regla 1:1 de AGENTS.md).

## Regla de oro

> Migrar SIEMPRE sobre una copia de la base, nunca sobre producción. Backfill idempotente: ejecutable N veces sin duplicar datos.

## Paso 0 — Preparación

1. Copiar `recibos.db` → `recibos_mig_test.db`.
2. Levantar la app apuntando a la copia (`DATABASE_URL=recibos_mig_test.db`) y verificar que arranca.
3. Exportar un snapshot de `departamento`, `recibo`, `detallerecibo` para comparar después (conteos y totales).

## Paso 1 — Estructura (aditiva, vía GORM Migrator)

En `internal/database` (función `Migrate`, idempotente):

| Cambio | API |
|---|---|
| Agregar `departamento.nombre` | `Migrator().AddColumn(&Departamento{}, "Nombre")` (o `HasColumn` + `AddColumn`) |
| Crear `persona` | `Migrator().CreateTable(&Persona{})` si no existe |
| Crear `departamento_persona` | `Migrator().CreateTable(&DepartamentoPersona{})` si no existe |
| Agregar `recibo.id_persona` | `Migrator().AddColumn(&Recibo{}, "IDPersona")` |

Columna propuesta para la puente (según DB):

```sql
UNIQUE (departamento_id, persona_id, rol)  -- evita duplicados activos
```

## Paso 2 — Backfill de `departamento.nombre` (unidad)

Parsear la unidad desde `departamento.propietario` (texto combinado actual). Orden de reglas:

1. **Sufijo `*XX*`**: si el texto termina en `*<unidad>*`, usar esa unidad y quitar el sufijo del nombre.
   - `1C  DR CARAN *1C*` → `1C` / `DR CARAN`
   - `3G CECILIA GUTIRREZ  *3G*(silvana)` → `3G` / `CECILIA GUTIRREZ (silvana)`
2. **Prefijo `^\d{1,2}[A-Ha-h]`** → unidad de departamento.
   - `1A DONADIO` → `1A` / `DONADIO`
   - `8A VERO ANGELILLI *8A*` → `8A` / `VERO ANGELILLI`
3. **Prefijo `SS`** (semi-subsuelo): `^SS\s*\d{1,2}[A-Za-z]`.
   - `SS2A` → `SS2A` / (sin nombre)
   - `ss1f  gettar` → `SS1F` (normalizar mayúsculas) / `GETTAR`
   - `SS 2 D ALDERETE` → `SS2D` / `ALDERETE`
   - `SS1J -RAMONA DEL CARMEN ORTIZ` → `SS1J` / `RAMONA DEL CARMEN ORTIZ`
4. **Prefijo `PB`**: `^PB\s*[A-Da-d]?`.
   - `PB C ALDERETE` → `PB C` / `ALDERETE`
5. **Sin unidad** (locales comerciales): `ILIAL CONSULTORIO`, `GRANDI`, `LOC COMERC` → `nombre` = texto completo, unidad vacía o derivada de `clientes.direccion` si se dispone.

> Reglas a implementar con expresiones regulares en Go (`regexp`), no a mano. Las filas que no matcheen ninguna regla se listan en un **reporte de revisión manual** (log o archivo) para corregirlas a mano.

## Paso 3 — Backfill de `persona` + `departamento_persona` (propietarios)

1. Para cada `departamento`:
   - Limpiar el nombre (aplicar las reglas del Paso 2 para separar unidad del nombre).
   - Normalizar para deduplicar (trim, mayúsculas, quitar signos): `SS2C` duplicado en ids 81/85, `SS2A` en 79/84 deben resolver a la MISMA persona.
2. Crear `persona` con `tipo = 'propietario'` (si el nombre normalizado no existe aún).
3. Crear vínculo en `departamento_persona` con `rol = 'propietario'`, `activo = 1`, `fecha_desde = fecha de creación` (o `NULL`).
4. **Inquilinos:** no hay dato legacy; quedan para carga manual (o CSV).

## Paso 4 — Backfill de `recibo.id_persona` (titular por recibo)

- Best-effort: cruzar `recibo.id_departamento` → `departamento_persona` (rol propietario) → `persona.id`.
- Los recibos sin vínculo (o legacy) dejan `id_persona = NULL` (la UI hace fallback a `departamento.propietario`).

## Paso 5 — Verificación

| Verificación | Consulta / método |
|---|---|
| No se perdieron recibos | `SELECT COUNT(*) FROM recibo` igual antes/después |
| Todas las unidades con `nombre` | `SELECT COUNT(*) FROM departamento WHERE nombre = ''` → 0 (o lista del reporte) |
| Personas deduplicadas | `SELECT nombre, COUNT(*) FROM persona GROUP BY nombre HAVING COUNT(*)>1` → vacío |
| Puente coherente | todo `departamento_persona.rol='propietario'` con persona existente |
| Totales de recibo intactos | `SELECT SUM(total) FROM recibo` igual antes/después |
| Idempotencia | ejecutar `Migrate` 2 veces → sin duplicados en `persona`/`departamento_persona` |

## Paso 6 — Puesta en producción

1. Backup de la base real.
2. Desplegar código con `Migrate` (se ejecuta al iniciar).
3. Correr, revisar logs/reporte de filas no parseadas y corregirlas vía la pantalla de Departamentos/Titulares.

## Riesgos y mitigaciones

| Riesgo | Mitigación |
|---|---|
| Parser erróneo en nombres extraños | Reporte de revisión manual + corrección desde UI |
| Duplicados de persona | Deduplicación por nombre normalizado + índice único parcial activo |
| Recibos con `id_departamento = 0` (239) | Quedan con `id_persona NULL`; documentados, no se migran |
| Conteo en `nextNumeroRecibo` | En v2.1 pasar a `MAX(numero_recibo)` |
