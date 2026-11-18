# Pantalla: Registro de acceso

## 1. Identificación

| Campo | Valor |
|---|---|
| Ruta | `GET /registro`, `POST /registro`, `POST /registro/{id}/aprobar` |
| Handler | `handlers.RegistroPage`, `handlers.RegistroCreate`, `handlers.RegistroAprobar` (`internal/handlers/registro.go`) |
| Template | `templates/registro.html` |
| Roles | Público (GET/POST `/registro`); `administrador` para aprobar |
| Propósito | Permitir que propietarios e inquilinos se den de alta con un link compartido, con rol asignado automáticamente. |

## 2. Flujo

1. El admin comparte el link `/registro`.
2. El usuario carga: unidad, nombre y apellido, email, teléfono (opcional) y contraseña (mín. 8).
3. El sistema busca coincidencia:
   - **Unidad + nombre**: contra `departamento.nombre` y los nombres de `Dueño`/`Inquilino`.
   - **Teléfono**: contra `persona.telefono` (últimos 10 dígitos, tolera prefijos).
4. Si hay coincidencia → **aprobación automática**: se crea `users` con rol derivado (`dueno` / `inquilino`), se completa `persona.email`/`telefono` si faltaban y se registra la operación. Redirige a `/login?registro=1`.
5. Si no hay coincidencia → queda **pendiente** en la tabla `registro`; se notifica al admin en el dashboard. Redirige a `/registro?pendiente=1`.
6. El admin aprueba desde el dashboard eligiendo el rol; se crea el `users` con el hash ya guardado.

## 3. Roles

| Origen del match | Rol asignado | Vínculo |
|---|---|---|
| Coincide con el dueño de la unidad | `dueno` | `users.id_persona` |
| Coincide con el inquilino de la unidad | `inquilino` | `users.id_persona` |
| Coincide por teléfono con `persona.tipo = propietario` | `dueno` | `users.id_persona` |
| Coincide por teléfono con `persona.tipo = inquilino` | `inquilino` | `users.id_persona` |
| Sin match, aprobación manual | el que elija el admin | `users.id_persona` si se encuentra persona |

`users.id_persona` garantiza que el usuario vea sus unidades aunque el email registrado difiera del que tenía la persona (los usuarios antiguos siguen resolviéndose por email).

El usuario **nunca elige su rol**. No puede auto-registrarse como administrador ni operador: esos los da de alta el admin directamente en la base.

## 4. Seguridad

- Rate limiting: 5 intentos por IP por hora (`internal/middleware/ratelimit.go`).
- Mensajes genéricos: no revela si el email/unidad existe.
- Contraseñas con bcrypt; el email es único en `users`.
- No se aceptan registros duplicados (usuario existente o solicitud pendiente con el mismo email).
- `POST /registro/{id}/aprobar` solo para `administrador`.
- `X-Forwarded-For` es spoofeable (ver `shortcut` en el código); detrás de un proxy confiable conviene usar la IP del proxy.

## 5. Notificación a admins

La tabla `registro` guarda todos los intentos. El dashboard del administrador muestra los últimos 10, el contador de pendientes y un botón **Aprobar** por cada pendiente. No se envía email (no hay SMTP configurado).

## 6. Datos

`models.Registro`: `id`, `email`, `nombre`, `unidad`, `telefono`, `password` (hash), `estado` (`auto` / `pendiente` / `aprobado` / `rechazado`), `rol`, `id_persona`, `ip`, `creado_en`.

## 7. Pendientes

- Pantalla `/usuarios` para que el admin cree operadores/administradores desde la UI.
- Envío de email/WhatsApp de confirmación.
- Tokens de invitación por persona (magic links).
- Aprobación/rechazo con motivo.
