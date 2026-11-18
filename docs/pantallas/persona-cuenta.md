# Pantalla: Cuenta de Persona

## 1. Identificación

| Campo | Valor |
|---|---|
| Ruta | `GET /personas/{id}/cuenta` |
| Handler | `handlers.PersonaCuenta` (`internal/handlers/personas.go`) |
| Template | `templates/persona_cuenta.html` |
| Roles | `administrador` (por ahora) |
| Propósito | Mostrar la cuenta corriente de una persona: unidades asociadas, recibos imputados, detalle de cada recibo y total a cobrar. |
| Acceso | autenticado + rol admin |

## 2. Datos involucrados

- Modelos: `models.Persona`, `models.Departamento`, `models.Recibo`, `models.DetalleRecibo`.
- Fuente de la deuda: `recibo.id_persona` (snapshot del deudor).
- Unidades actuales: `departamento.id_dueño = persona.id OR id_inquilino = persona.id`.
- Unidades históricas: unidades con deuda imputada a la persona que ya no ocupa.
- Totales calculados en Go con `MontoPagar()` (incluye recargo por vencimiento) y `MontoPagado`.

## 3. Pantalla

- Cabecera: nombre, tipo, teléfono, email, estado.
- Tarjetas: **A cobrar**, **Pagado**, **Saldo a favor**, **Unidades**.
- **Unidades**: unidad, rol (propietario/inquilino/histórica), coeficiente, pendiente, link al historial.
- **Recibos**: filtro `estado` (`pendiente` por defecto, `todos`).
- Tabla de recibos con tres columnas:
  - **Recibo**: número, período, unidad, estado (Pagado / Pago parcial / Pendiente / +20%) y link al recibo.
  - **Detalle**: `<details>` con los `detallerecibo` (concepto, importe, forma de pago).
  - **Importe**: monto a pagar, base, pagado, saldo a favor y fecha de pago.

## 4. Reglas

- Solo se listan recibos con `id_persona = persona.id`.
- Un propietario no ve la deuda del inquilino de su unidad; el admin puede consultar la cuenta del inquilino por separado.
- Un inquilino viejo con deuda sigue apareciendo por el snapshot de sus recibos.
- Pago parcial: el original aparece como “Pago parcial” y la deuda anterior hija aparece como pendiente.
- Recibos legacy sin `id_persona` no aparecen en la cuenta; se ven en el historial de la unidad.

## 5. Acciones

| Acción | Destino | Método |
|---|---|---|
| Ver recibo | `/recibos/{id}` | GET |
| Ver historial de unidad | `/departamentos/{id}/historial` | GET |
| Filtrar | `/personas/{id}/cuenta?estado=pendiente\|todos` | GET |
| Volver | `/personas` | GET |

## 6. Pendientes

- Habilitar el acceso para el operador (`receptor`) y para el propio titular.
- Paginación si una persona acumula muchos recibos.
- Botón “Cobrar todo lo pendiente” (multi-pago).
