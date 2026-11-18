# Actualizar recibosGo en `rpiq1`

Guía para trabajar **desde este proyecto** (`/home/daniel/Proyectos/recibosGo`) y actualizar la app que corre en la Raspberry Pi 5.

---

## Datos del despliegue

| Dato | Valor |
|---|---|
| Host RPi | `rpiq1` (Tailscale: `100.100.199.53`) |
| Usuario SSH | `daniel` |
| Arquitectura | `aarch64` / ARM64 |
| URL de la app | <http://rpiq1:8082/> |
| Servicio | `recibosgo.service` (**systemd de usuario**, no root) |
| Directorio remoto | `/home/daniel/recibosGo` |
| Archivo de entorno | `/home/daniel/recibosGo/recibosgo.env` (`chmod 600`) |
| Base en la RPi | contenedor Docker `postgres`, base `recibosgo`, usuario `recibosgo` |
| Base local de referencia | contenedor Docker `recibosgo-db`, base `recibosgo`, `127.0.0.1:5432` |

> **PostgreSQL es la única fuente de verdad**: contenedor `recibosgo-db` en local y `postgres` en la RPi. Los `*.db` de SQLite y los `*.csv` del repo son artefactos históricos: la app no los lee al arrancar y no se sincronizan.

Última sincronización: **2026-11-18** → `persona 114`, `departamento 118`, `servicio 4`, `recibo 5918`, `detallerecibo 8200`.

---

## 0. Preparación (solo la primera vez)

Verificar acceso:

```bash
ssh daniel@rpiq1 'uname -m'
# debe devolver: aarch64
```

Herramientas necesarias en esta máquina: `go`, `rsync`, `docker`, `ssh`, `curl`.

Para que el servicio de usuario sobreviva a un logout/reboot, ejecutar **una vez** (pide la password de la RPi):

```bash
ssh daniel@rpiq1 'sudo loginctl enable-linger daniel'
```

Comando útil para servicios de usuario (se usa varias veces abajo):

```bash
ssh daniel@rpiq1 'export XDG_RUNTIME_DIR=/run/user/1000 DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1000/bus; systemctl --user STATUS recibosgo.service'
```

Reemplazar `STATUS` por `restart`, `stop`, `start` o `status`.

---

## 1. Actualizar solo el código (sin tocar la base)

Se compila a ARM64 acá y se suben binario + templates + estáticos. El `recibosgo.env` remoto **no se toca**. No se suben `data/`, CSV ni SQLite: no son fuente de verdad.

```bash
set -e
cd /home/daniel/Proyectos/recibosGo

STAGE=$(mktemp -d /tmp/recibosgo-stage.XXXXXX)

CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
  go build -trimpath -o "$STAGE/recibos-arm64" ./cmd/server

cp -a templates static "$STAGE/"

rsync -avz --delete \
  --exclude='recibosgo.env' \
  --exclude='*.log' \
  -e 'ssh -o BatchMode=yes -o ConnectTimeout=15' \
  "$STAGE/" daniel@rpiq1:/home/daniel/recibosGo/

rm -rf "$STAGE"

ssh daniel@rpiq1 'export XDG_RUNTIME_DIR=/run/user/1000 DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1000/bus; systemctl --user restart recibosgo.service'
```

Al arrancar, la app ejecuta `AutoMigrate` + migraciones aditivas (`migrateV21`, `migrateServicios`, `migrateIndexes`), todas idempotentes. No hay backfill desde CSV.

Verificación rápida:

```bash
ssh daniel@rpiq1 'curl -sS -o /dev/null -w "%{http_code}\n" http://127.0.0.1:8082/'
# 302 = redirect al login, está bien
```

---

## 2. Actualizar los datos (Postgres local → RPi)

Usar cuando cargaste recibos/servicios en local y querés que se vean en la RPi.

### 2.1 Backup de la base actual de la RPi

```bash
# backup dentro del contenedor de la RPi
ssh daniel@rpiq1 'docker exec postgres pg_dump -U postgres -d recibosgo -w -Fc -f /tmp/recibosgo_before_sync.dump'

# copia del backup en esta máquina
ssh daniel@rpiq1 'docker exec postgres pg_dump -U postgres -d recibosgo -w -Fc' \
  > /tmp/rpi_before_sync_$(date +%F).dump
```

### 2.2 Detener la app en la RPi

```bash
ssh daniel@rpiq1 'export XDG_RUNTIME_DIR=/run/user/1000 DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1000/bus; systemctl --user stop recibosgo.service'
```

### 2.3 Dump del Postgres local y restore en la RPi

```bash
set -e
DUMP=/tmp/recibosgo_local_$(date +%F).dump

docker exec recibosgo-db pg_dump -U recibosgo -d recibosgo -w -Fc > "$DUMP"
ls -lh "$DUMP"

ssh daniel@rpiq1 "docker exec postgres psql -U postgres -d recibosgo -w -c 'DROP SCHEMA public CASCADE; CREATE SCHEMA public; GRANT ALL ON SCHEMA public TO recibosgo;'"

ssh daniel@rpiq1 'docker exec -i postgres pg_restore -U postgres -d recibosgo --no-privileges --exit-on-error' < "$DUMP"
```

> No agregar `-q` a `pg_restore`: en esta versión no es una opción válida y falla.

### 2.4 Regenerar el usuario admin con el password de la RPi

El dump trae el usuario de la base local. Se borra para que la app lo vuelva a crear usando `ADMIN_EMAIL` / `ADMIN_PASSWORD` del `recibosgo.env` de la RPi:

```bash
ssh daniel@rpiq1 "docker exec postgres psql -U postgres -d recibosgo -w -c 'DELETE FROM users;'"
```

### 2.5 Arrancar y verificar conteos

```bash
ssh daniel@rpiq1 'export XDG_RUNTIME_DIR=/run/user/1000 DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1000/bus; systemctl --user start recibosgo.service'

ssh daniel@rpiq1 "docker exec postgres psql -U postgres -d recibosgo -w -c \"select 'persona' t, count(*) n from persona union all select 'departamento', count(*) from departamento union all select 'servicio', count(*) from servicio union all select 'recibo', count(*) from recibo union all select 'detallerecibo', count(*) from detallerecibo union all select 'users', count(*) from users order by 1\""
```

---

## 3. Verificación funcional con login (opcional pero recomendada)

No imprime la contraseña: la lee del `.env` remoto y se la pasa a `curl` por stdin.

```bash
ssh daniel@rpiq1 bash -s <<'EOF'
set -e
set -a
. /home/daniel/recibosGo/recibosgo.env
set +a

printf 'url="http://127.0.0.1:8082/login"\ndata-urlencode="email=%s"\ndata-urlencode="password=%s"\n' \
  "$ADMIN_EMAIL" "$ADMIN_PASSWORD" \
  | curl -sS -c /tmp/recibosgo.cookies -b /tmp/recibosgo.cookies \
      -o /dev/null -w 'LOGIN_HTTP=%{http_code}\n' -K -

curl -sS -b /tmp/recibosgo.cookies http://127.0.0.1:8082/dashboard \
  | sed 's/<[^>]*>/ /g' | tr -s '[:space:]' ' ' \
  | grep -oE '[0-9]+ Departamentos|[0-9]+ Servicios|[0-9]+ Recibos totales'
EOF
```

Resultado esperado: `LOGIN_HTTP=302` y las tres cifras del dashboard.

---

## 4. Comandos útiles

Ver logs en vivo:

```bash
ssh daniel@rpiq1 'export XDG_RUNTIME_DIR=/run/user/1000 DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1000/bus; journalctl --user -u recibosgo.service -f'
```

Estado del servicio:

```bash
ssh daniel@rpiq1 'export XDG_RUNTIME_DIR=/run/user/1000 DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1000/bus; systemctl --user --no-pager status recibosgo.service'
```

Ver configuración remota **sin mostrar secretos**:

```bash
ssh daniel@rpiq1 'grep -E "^(PORT|DATABASE_URL|ADMIN_EMAIL)=" /home/daniel/recibosGo/recibosgo.env'
```

Cambiar la password del admin:

```bash
# 1) editar ADMIN_PASSWORD en /home/daniel/recibosGo/recibosgo.env en la RPi
ssh daniel@rpiq1 'nano /home/daniel/recibosGo/recibosgo.env'

# 2) borrar el usuario y reiniciar para que se regenere
ssh daniel@rpiq1 "docker exec postgres psql -U postgres -d recibosgo -w -c 'DELETE FROM users;'"
ssh daniel@rpiq1 'export XDG_RUNTIME_DIR=/run/user/1000 DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1000/bus; systemctl --user restart recibosgo.service'
```

---

## 5. Trampas conocidas

- **No usar** `recibosgo-linux-x86.zip`: es amd64 y SQLite viejo, no sirve para la RPi.
- **No sincronizar desde** `recibos.db`, `mp135.db`, otros `*.db` ni `*.csv`: son artefactos históricos. La fuente real es PostgreSQL (`recibosgo-db` / `postgres`).
- Los recibos con `id_departamento = 0` aparecen sin unidad/propietario; vienen así del sistema legacy (eran 239). No es un error nuevo.
- La app **no ejecuta backfill desde CSV** al iniciar; los vínculos dueño/inquilino se corrigen directamente en PostgreSQL.
- Si `curl http://rpiq1:8082/` falla desde esta máquina pero funciona dentro de la RPi, revisar Tailscale/firewall antes de tocar la app.
