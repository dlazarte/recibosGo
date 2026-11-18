# Verificación de navegación — RecibosGo

**Fecha**: 2026-11-18 · **Servidor probado**: `http://localhost:8090` (Postgres local `recibosgo-db`) · **Estado**: sin enlaces huérfanos, sin fallos.

## 1. Procedimiento

1. **Inventario**: `grep -rnoE '<a [^>]*href="[^"]*"'` + `<form`, `hx-get`, `hx-post`, `hx-target`, `onclick` sobre todo `templates/`. Se usó un patrón que captura `href` aunque no sea el primer atributo (los "Volver"/"Cancelar" tienen `class` antes de `href`).
2. **Rutas Chi**: contraste con `internal/server/server.go` (método + path exacto).
3. **Generación de URL**: extracción de los `href` reales del HTML renderizado; sin `//` ni IDs vacíos.
4. **Prueba real**: `curl` con 4 sesiones válidas (`administrador`, `dueno`, `inquilino`, `receptor`) creadas con bcrypt en la base, más sesión pública. Se verificó código HTTP, cabecera `Location` y ausencia de bucles de redirect.
5. **HTMX**: peticiones con `HX-Request: true`, comprobando que devuelven fragmento (no `<html>`) y que el `hx-target` existe en el DOM.
6. **Volver/Cancelar**: flujo lista → detalle → Volver, y edición → Cancelar, extrayendo el `href` del HTML y pidiéndolo después.

> Los POST se probaron con payloads que el handler rechaza o que son no-op (mismos valores), para no alterar datos. Se verificó después que `departamento.coeficiente`, `servicio.nombre`, `persona.nombre`, `recibo.fecha_pago`/`monto_pagado` y los registros pendientes quedaron intactos. Los usuarios de prueba se eliminaron al final.

## 2. Tabla de resultados

| Enlace / Botón | Plantilla | Ruta Chi | Método | Resultado (HTTP) | Estado |
|---|---|---|---|---|---|
| Dashboard | `layout.html` (×2: desktop/móvil) | `GET /dashboard` | GET | 200 | OK |
| Departamentos | `layout.html` (×2) | `GET /departamentos/` | GET | 200 (admin/dueno) | OK |
| Titulares | `layout.html` (×2) | `GET /personas/` | GET | 200 (admin/dueno) | OK |
| Servicios | `layout.html` (×2) | `GET /servicios/` | GET | 200 (admin/dueno) | OK |
| Reportes | `layout.html` (×2) | `GET /reportes` | GET | 200 (admin/dueno) | OK |
| Nuevo Recibo | `layout.html` (×2, `IsReceptor`) | `GET /recibos/new` | GET | 200 | OK |
| Recibos | `layout.html` (×2) | `GET /recibos/` | GET | 200 | OK |
| Deudas | `layout.html` (×2) | `GET /deudas/` | GET | 200 | OK |
| Salir | `layout.html` (×2, form) | `POST /logout` | POST | 302 → `/login` | OK |
| Modo oscuro / ☰ Menú | `layout.html` (`onclick`) | — (JS local) | — | sin navegación | N/A |
| Ingresar | `login.html` (form) | `POST /login` | POST | 302 → `/dashboard` | OK |
| ¿No tenés cuenta? Registrate | `login.html` | `GET /registro` | GET | 200 | OK |
| Registrarse | `registro.html` (form) | `POST /registro` | POST | 302 → `/registro?error=1` (inválido) · 302 → `/login?registro=1` (válido, vía tests) | OK |
| Ya tengo cuenta → Iniciar sesión | `registro.html` | `GET /login` | GET | 200 | OK |
| Ver (recibo) | `dashboard.html` (×2) | `GET /recibos/{id}` | GET | 200 | OK |
| Aprobar | `dashboard.html` (form dinámico) | `POST /registro/{id}/aprobar` | POST | 302 → `/dashboard?msg=Registro+aprobado` | OK |
| Buscar | `departamentos.html` (form + `hx-get`) | `GET /departamentos/` | GET | 200 · fragmento `deptos_table` | OK |
| Editar | `fragments/deptos_table.html` | `GET /departamentos/?edit={id}` | GET | 200 | OK |
| Historial | `fragments/deptos_table.html` | `GET /departamentos/{id}/historial` | GET | 200 | OK |
| Crear / Guardar cambios | `departamentos.html` (form) | `POST /departamentos/` · `POST /departamentos/{id}` | POST | 302 → `/departamentos?msg=…` | OK |
| **Cancelar** | `departamentos.html` | `GET /departamentos` | GET | 200 | OK |
| **Volver** | `departamento_historial.html` | `GET /departamentos` | GET | 200 | OK |
| Ver (recibo) | `departamento_historial.html` | `GET /recibos/{id}` | GET | 200 | OK |
| Editar | `servicios.html` | `GET /servicios/?edit={id}` | GET | 200 | OK |
| **Cancelar** | `servicios.html` | `GET /servicios` | GET | 200 | OK |
| Crear / Guardar cambios | `servicios.html` (form) | `POST /servicios/` · `POST /servicios/{id}` | POST | 302 → `/servicios?msg=…` | OK |
| Editar | `fragments/personas_table.html` | `GET /personas/?edit={id}` | GET | 200 | OK |
| Cuenta | `fragments/personas_table.html` | `GET /personas/{id}/cuenta` | GET | 200 | OK |
| **Cancelar** | `titulares.html` | `GET /personas` | GET | 200 | OK |
| Buscar | `titulares.html` (form) | `GET /personas/` | GET | 200 | OK |
| Crear / Guardar cambios | `titulares.html` (form) | `POST /personas/` · `POST /personas/{id}` | POST | 302 → `/personas?msg=…` | OK |
| **Volver** | `persona_cuenta.html` | `GET /personas` | GET | 200 | OK |
| Pendientes / Todos | `persona_cuenta.html` | `GET /personas/{id}/cuenta?estado=…` | GET | 200 | OK |
| Historial | `persona_cuenta.html` | `GET /departamentos/{id}/historial` | GET | 200 | OK |
| Ver (recibo) | `persona_cuenta.html` | `GET /recibos/{id}` | GET | 200 | OK |
| Nuevo recibo | `recibos_list.html` | `GET /recibos/new` | GET | 200 | OK |
| Filtrar | `recibos_list.html` (form + `hx-get`) | `GET /recibos/` | GET | 200 · fragmento `recibos_table` | OK |
| Ver | `fragments/recibos_table.html` | `GET /recibos/{id}` | GET | 200 | OK |
| PDF | `fragments/recibos_table.html` | `GET /recibos/{id}/pdf` | GET | 200 (`application/pdf`) | OK |
| Descargar PDF | `recibo_detail.html` | `GET /recibos/{id}/pdf` | GET | 200 | OK |
| **Volver** | `recibo_detail.html` | `GET /recibos` | GET | 200 | OK |
| Generar recibos | `recibo_create.html` (form) | `POST /recibos/` | POST | 302 → `/recibos/new?msg=…` | OK |
| Unidades del titular (`hx-get`) | `recibo_create.html` | `GET /recibos/new?id_persona=…` | GET | 200 · fragmento `recibo_unidades` · target `#unidades-container` OK | OK |
| + Agregar item / ✕ | `recibo_create.html`, `fragments/item_row.html` (`onclick`) | — (JS local) | — | sin navegación | N/A |
| Registrar pago | `fragments/recibo_detalle.html` (form + `hx-post`) | `POST /recibos/{id}/pagar` | POST | 200 fragmento `recibo_detalle` (HTMX) · 302 (normal) | OK |
| Ver mes | `reportes.html` (form + `hx-get`) | `GET /reportes` | GET | 200 · fragmento `reportes_content` | OK |
| Nº de recibo | `fragments/reportes_content.html` | `GET /recibos/{id}` | GET | 200 | OK |
| Generar expensas | `deudas.html` (form) | `POST /deudas/` | POST | 302 → `/deudas?msg=…` | OK |
| Generar servicios | `deudas.html` (form) | `POST /deudas/servicios` | POST | 302 → `/deudas?msg=…` | OK |
| Generar extraordinarias | `deudas.html` (form) | `POST /deudas/extraordinarias` | POST | 302 → `/deudas?msg=…` | OK |
| Filtrar | `deudas.html` (form + `hx-get`) | `GET /deudas/` | GET | 200 · fragmento `deudas_table` | OK |
| Ver | `fragments/deudas_table.html` | `GET /recibos/{id}` | GET | 200 | OK |
| Pagar | `fragments/deudas_table.html` (form) | `POST /recibos/{id}/pagar` | POST | 302 → `volver` | OK |
| ← Anterior / N / Siguiente → | `fragments/paginador.html` (`hx-get`) | `GET {path}?page=N` | GET | 200 · fragmento · target OK | OK |
| `/` sin sesión | — | `GET /` (grupo auth) | GET | 302 → `/login` | OK |
| `/` con sesión | — | `GET /` | GET | 302 → `/dashboard` | OK |
| `/dashboard` sin sesión | — | `GET /dashboard` | GET | 302 → `/login` | OK |
| `/static/style.css` | `layout.html` (`<link>`) | `GET /static/*` | GET | 200 | OK |

**Totales**: 70 comprobaciones de la corrida principal + 18 de POST/HTMX + 4 de aprobación = **92 OK, 0 FALLA**.

## 3. Flujos "Volver" / "Cancelar" (prueba explícita)

| Flujo | Resultado |
|---|---|
| `/recibos` → `/recibos/5703` → Volver → `/recibos` | 200 → 200 → 200, sin redirect a login ni a raíz |
| `/recibos` (dueno) → `/recibos/5915` → Volver → `/recibos` | 200 → 200 → 200 (sesión propagada) |
| `/personas` → `/personas/1/cuenta` → Volver → `/personas` | 200 → 200 → 200 |
| `/departamentos` → `/departamentos/1/historial` → Volver → `/departamentos` | 200 → 200 → 200 |
| `/departamentos?edit=1` → Cancelar → `/departamentos` | 200 → 200 |
| `/servicios?edit=1` → Cancelar → `/servicios` | 200 → 200 |
| `/personas?edit=1` → Cancelar → `/personas` | 200 → 200 |
| `/registro` → "Ya tengo cuenta" → `/login` | 200 → 200 |
| `/login` → "Registrate" → `/registro` | 200 → 200 |

> La edición es **inline** (`?edit={id}` sobre la lista), no una página separada: por eso "Cancelar" vuelve a la lista y "Volver" del detalle vuelve a la lista. No hay página de edición intermedia.

## 4. HTMX

| Petición | Fragmento | `hx-target` en DOM |
|---|---|---|
| `GET /recibos` | `recibos_table` | `#recibos-table` OK |
| `GET /departamentos` | `deptos_table` | `#deptos-table` OK |
| `GET /deudas` | `deudas_table` | `#deudas-table` OK |
| `GET /reportes` | `reportes_content` | `#reportes-content` OK |
| `GET /recibos?page=2` | `recibos_table` | `#recibos-table` OK |
| `GET /recibos/new?id_persona=1` | `recibo_unidades` | `#unidades-container` OK (`hx-swap=innerHTML`) |
| `POST /recibos/5703/pagar` | `recibo_detalle` | `#recibo-detalle` OK |

Ninguna respondió una página completa (`<html>` ausente en todas).

## 5. Enlaces huérfanos y rutas sin enlace

- **Enlaces huérfanos**: 0. Cada `href`/`action`/`hx-*` coincide con una ruta Chi del método correcto.
- **Rutas sin enlace directo**: `GET /` (redirect de conveniencia, no se enlaza) y `GET /static/*` (enlazado vía `<link>`/`<script>`, no `<a>`). Ambas probadas: 302 y 200.
- Los `403` observados son **guardas de rol/propiedad esperadas**, no enlaces rotos: `inquilino`/`receptor` → `/departamentos` (403), `receptor` → `/personas/{id}/cuenta` (403), `dueno` → `/recibos/5703` (403, recibo de otra unidad, control S1). El `dueno` vinculado a persona 26 sí accede a `/recibos/5713` (200).

## 6. Generación de URL

Verificado en `/recibos`, `/dashboard`, `/deudas`, `/personas`, `/departamentos`, `/reportes`, `/personas/1/cuenta`, `/departamentos/1/historial`: ningún `href` con `//` ni con ID vacío (`/recibos//pdf`, `/personas//cuenta`). Los `href` del paginador extraídos del HTML (`/recibos?page=1`, `/departamentos?page=1`, `/deudas?page=1`, `/reportes?page=1`) devolvieron 200.

## 7. No probado / observaciones

- **Botones JS puro** (`toggleDark`, `toggleMobile`, `agregarItem`, quitar item): no navegan; no se prueban con curl. Se revisó que no tengan `href`/`action`.
- **Observación de diseño (no fallo de navegación)**: `GET /personas/{id}/cuenta` está dentro del grupo `RequireRole("administrador","dueño","dueno")`, por lo que un `dueño` puede abrirla aunque la spec `persona-cuenta.md` dice "solo admin". El enlace "Cuenta" se muestra en `/personas` (mismo grupo), así que **no hay enlace roto**; si se restringe a admin habría que ocultar también ese enlace para el `dueño`. Queda documentado, sin cambios.
- Los POST se probaron con payloads rechazados o no-op; los flujos de escritura reales (generar deuda, pago parcial, aprobar) están cubiertos por `go test ./...` contra `recibosgo_test`.

## 8. Limpieza

Usuarios `nav.*@test.local` eliminados; `users` volvió a 1 (admin), `registro` a 0 filas. `departamento.coeficiente`, `servicio.nombre`, `persona.nombre` y `recibo 5703` verificados sin cambios. Scripts de prueba en `/tmp/opencode/nav_check*.sh`.

---

## 9. Apéndice — re-verificación tras el fix 8A (2026-11-18)

Se corrigió el inquilino de 8A (`FATIMA BLANCO` → `LILIANA`, dato del Excel vigente) y se agregó la columna **Inquilino** + el selector `id_inquilino` en el formulario de Departamentos. Plantillas tocadas: `departamentos.html`, `fragments/deptos_table.html`.

| Enlace / Botón | Plantilla | Ruta Chi | Método | Resultado | Estado |
|---|---|---|---|---|---|
| Listado Departamentos | `deptos_table.html` | `GET /departamentos/` | GET | 200 | OK |
| Búsqueda `?q=8A` | `departamentos.html` | `GET /departamentos/` | GET | 200 | OK |
| Editar | `deptos_table.html` | `GET /departamentos/?edit={id}` | GET | 200 | OK |
| Historial | `deptos_table.html` | `GET /departamentos/{id}/historial` | GET | 200 | OK |
| Paginador | `paginador.html` | `GET /departamentos/?page=N` | GET | 200 | OK |
| **Cancelar** (edición) | `departamentos.html` | `GET /departamentos` | GET | 200 | OK |
| **Volver** (historial) | `departamento_historial.html` | `GET /departamentos` | GET | 200 | OK |
| Buscar / fragmento (`hx-get`) | `departamentos.html` | `GET /departamentos/` | GET | 200 · fragmento `deptos_table`, target `#deptos-table` OK | OK |
| Crear / Guardar (form) | `departamentos.html` | `POST /departamentos/` · `/{id}` | POST | 302 → `?msg=…` (cubierto por tests) | OK |

**9 OK, 0 FALLA** (local). Repetido en la RPi: `LOGIN_HTTP=302`, `/departamentos?q=8A` muestra `VERONICA ANGELILLI` + `LILIANA`, columna `Inquilino` presente y `<option value="64" selected>LILIANA` en el formulario de edición.

Nuevos tests: `TestDepartamentoMuestraInquilino`, `TestDepartamentoFormularioInquilinoSeleccionado`, `TestDepartamentoUpdateConservaInquilino`, `TestDepartamentoUpdateLimpiaInquilino`, `TestDepartamentoCreateConInquilino`. `go test ./...` OK.

---

## 10. Apéndice — CSV/SQLite eliminados como fuente de verdad (2026-11-18)

Se quitó `CSV_PATH` de la configuración, se eliminó `BackfillV21` del arranque y el parser de CSV. Se des-versionaron los `*.csv` (`git rm --cached`, siguen en disco) y se añadió `*.csv` a `.gitignore`. PostgreSQL quedó declarado como única fuente de verdad en `AGENTS.md`, `README.md`, `docs/`, `rpiq1.md` y `data/REVISION_MANUAL.md`. El despliegue ya no sincroniza `data/` ni CSV.

**Re-verificación de navegación tras este cambio** (sin tocar plantillas, pero regla obligatoria): `nav_check.sh` → **70 OK / 0 FALLA**, `nav_deptos.sh` → **9 OK / 0 FALLA**. Log de arranque limpio (sin `record not found` ni mensajes de CSV). `go test ./...` OK.

**No probado / pendiente**: el despliegue a la RPi no pudo ejecutarse porque `ssh daniel@rpiq1` y `daniel@100.100.199.53` dieron `Connection timed out` (Tailscale/host caído). La RPi sigue con el binario anterior que aún incluía el backfill por CSV; al recuperar conectividad hay que correr la sección 1 de `rpiq1.md` (ya actualizada: no sube CSV). Hasta entonces, la declaración de fuente de verdad aplica al repo y a la base local/PostgreSQL; el dato 8A=LILIANA ya está corregido en la RPi desde el turno anterior vía SQL directo.


