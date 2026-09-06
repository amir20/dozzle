---
title: Podman
sourceHash: 6baf154c7545
---

# Podman

Dozzle funciona con Podman a través de su socket compatible con Docker. Hay dos diferencias conocidas respecto a Docker que afectan a la instalación: las estadísticas de memoria suelen faltar en despliegues rootless o con Quadlet (por la delegación de cgroups) y Podman no genera un engine-id. Esta guía cubre el modo independiente (monitorización local) y el modo agente (monitorización remota desde un servidor Dozzle central).

## Opciones de despliegue

| Modo              | Caso de uso                           | Complejidad |
| ----------------- | ------------------------------------- | ----------- |
| **Independiente** | Ver logs de un solo host              | Sencilla    |
| **Agente**        | Monitorización centralizada multihost | Media       |

### Métodos de despliegue

Podman ofrece varias formas de arrancar los contenedores:

| Método            | Autoarranque | Stats de memoria | Healthchecks | Ideal para |
| ----------------- | ------------ | ---------------- | ------------ | ---------- |
| CLI               | Manual       | ✓                | ✓            | Desarrollo |
| `podman-compose`  | ✗            | ✓                | ✗            | Pruebas    |
| Quadlet (systemd) | ✓            | ✗\*              | ✓            | Producción |

\*Las estadísticas de memoria no suelen estar disponibles en modo rootless salvo que se active la delegación de memoria en cgroup v2. Consulta la FAQ al final de esta página.

---

# <Icon icon="mdi:monitor-dashboard" inline /> Modo independiente

Ejecuta Dozzle como servicio independiente para monitorizar los contenedores locales de Podman.

## <Icon icon="mdi:shield-account-outline" inline /> Configuración rootful

Para el demonio de Podman de todo el sistema:

```bash
# Habilita e inicia el socket de Podman
sudo systemctl enable podman.socket
sudo systemctl start podman.socket

# Dozzle puede conectarse a través del socket de Docker
podman run -v /run/podman/podman.sock:/var/run/docker.sock:ro \
  -p 3000:8080 \
  ghcr.io/amir20/dozzle:latest
```

## <Icon icon="mdi:account-outline" inline /> Configuración rootless

Podman en modo rootless aísla los contenedores en un espacio de nombres de usuario:

```bash
# Inicia el socket a nivel de usuario (se ejecuta automáticamente con la sesión del usuario)
systemctl --user enable podman.socket
systemctl --user start podman.socket

# Para un usuario llamado 'appuser', Dozzle puede conectarse así:
podman run -v /run/user/$(id -u appuser)/podman/podman.sock:/var/run/docker.sock:ro \
  -p 3000:8080 \
  ghcr.io/amir20/dozzle:latest
```

**Importante**: un Dozzle enlazado al socket rootless de un usuario solo ve los contenedores de ese usuario. Los contenedores rootless de otros usuarios viven en espacios de nombres distintos y no aparecerán.

## <Icon icon="mdi:rocket-launch-outline" inline /> Despliegue con Quadlet

Quadlet permite gestionar contenedores de forma nativa con systemd. Crea un archivo `.container` en `~/.config/containers/systemd/dozzle.container`:

```ini
[Unit]
Description=Dozzle Log Viewer
After=network-online.target
Wants=network-online.target

[Container]
Image=ghcr.io/amir20/dozzle:latest
PublishPort=3000:8080
Volume=/run/user/%U/podman/podman.sock:/var/run/docker.sock:ro

HealthCmd=/dozzle healthcheck
HealthInterval=5s
HealthTimeout=10s
HealthRetries=5
HealthStartPeriod=15s

[Service]
Restart=on-failure
RestartSec=10

[Install]
WantedBy=default.target
```

Habilítalo e inícialo:

```bash
systemctl --user daemon-reload
systemctl --user enable --now dozzle.service
```

En sistemas multiusuario, copia el mismo archivo en el `~/.config/containers/systemd/` de cada usuario y elige un puerto distinto en el host para cada uno (por ejemplo, `PublishPort=3001:8080`). Cada instancia solo ve los contenedores rootless de su usuario.

> [!NOTE] Quadlet genera un temporizador de systemd para los healthchecks. `podman-compose` no lo hace, así que allí los healthchecks no se ejecutan de forma programada; lánzalos a mano con `podman healthcheck run NAME` si te hace falta.

---

# <Icon icon="mdi:lan-connect" inline /> Modo agente

Ejecuta Dozzle como agente en los hosts remotos de Podman para monitorizarlos de forma centralizada desde un servidor Dozzle principal. Los agentes se comunican con el servidor principal por gRPC.

## <Icon icon="mdi:cog-outline" inline /> Configuración del agente

### Requisitos previos

- Abrir el puerto 7007 en el host del agente
- Conectividad de red entre el servidor principal y el agente

### Iniciar el agente de Dozzle

Ejecuta Dozzle en modo agente en los hosts remotos de Podman:

```bash
# Agente rootful
podman run -d \
  --name dozzle-agent \
  -v /run/podman/podman.sock:/var/run/docker.sock:ro \
  -p 7007:7007 \
  ghcr.io/amir20/dozzle:latest agent
```

```bash
# Agente rootless (para el usuario 'appuser')
sudo -u appuser podman run -d \
  --name dozzle-agent \
  -v /run/user/$(id -u appuser)/podman/podman.sock:/var/run/docker.sock:ro \
  -p 7007:7007 \
  ghcr.io/amir20/dozzle:latest agent
```

### Despliegue del agente con Quadlet

Crea un archivo `.container` para el agente:

```ini
# dozzle-agent.container
[Unit]
Description=Dozzle Agent
After=network-online.target
Wants=network-online.target

[Container]
Image=ghcr.io/amir20/dozzle:latest
PublishPort=7007:7007
Volume=/run/user/%U/podman/podman.sock:/var/run/docker.sock:ro
Exec=agent

HealthCmd=/dozzle healthcheck
HealthInterval=5s
HealthTimeout=10s
HealthRetries=5
HealthStartPeriod=15s

[Service]
Restart=on-failure
RestartSec=10

[Install]
WantedBy=default.target
```

> [!NOTE] El entrypoint de la imagen de Dozzle es `/dozzle`, así que `agent` va en `Exec=` (el comando) y no en `Entrypoint=`.

Habilítalo e inícialo:

```bash
systemctl --user daemon-reload
systemctl --user enable dozzle-agent.service
systemctl --user start dozzle-agent.service
```

---

# <Icon icon="mdi:server-network" inline /> Servidor principal con agentes remotos

Configura el servidor Dozzle principal para que se conecte a los agentes de los hosts remotos de Podman.

## <Icon icon="mdi:cog" inline /> Configuración del servidor

Ejecuta el servidor Dozzle principal indicando los endpoints de los agentes:

```bash
podman run -d \
  --name dozzle \
  -p 3000:8080 \
  ghcr.io/amir20/dozzle:latest \
  --remote-agent "host1.example.com:7007" \
  --remote-agent "host2.example.com:7007"
```

O con variables de entorno:

```bash
podman run -d \
  --name dozzle \
  -e DOZZLE_REMOTE_AGENT="host1.example.com:7007,host2.example.com:7007" \
  -p 3000:8080 \
  ghcr.io/amir20/dozzle:latest
```

### Servidor principal con agentes usando Quadlet

```ini
# dozzle-server.container
[Unit]
Description=Dozzle Server with Remote Agents
After=network-online.target
Wants=network-online.target

[Container]
Image=ghcr.io/amir20/dozzle:latest
PublishPort=3000:8080
Environment=DOZZLE_REMOTE_AGENT=host1.example.com:7007,host2.example.com:7007

HealthCmd=/dozzle healthcheck
HealthInterval=5s
HealthTimeout=10s
HealthRetries=5
HealthStartPeriod=15s

[Service]
Restart=on-failure
RestartSec=10

[Install]
WantedBy=default.target
```

> [!NOTE] `WantedBy=multi-user.target` solo vale para unidades de sistema. Para las unidades de `systemctl --user`, usa `default.target`.

---

# <Icon icon="mdi:tune" inline /> Configuración adicional

## <Icon icon="mdi:identifier" inline /> Configurar el engine-id

Podman no crea un engine-id como hace Docker. Créalo tú para evitar errores de tipo «host not found»:

### Con uuidgen

```bash
# Crea el directorio si hace falta
sudo mkdir -p /var/lib/docker

# Genera el UUID
sudo sh -c 'uuidgen > /var/lib/docker/engine-id'

# Comprueba el resultado
cat /var/lib/docker/engine-id
```

### Con Ansible

```yaml
- name: Create /var/lib/docker
  ansible.builtin.file:
    path: /var/lib/docker
    state: directory
    mode: "755"

- name: Create engine-id and derive UUID from hostname
  ansible.builtin.lineinfile:
    path: /var/lib/docker/engine-id
    line: "{{ hostname | to_uuid }}"
    create: true
    mode: "0644"
    insertafter: "EOF"
```

> [!WARNING] Limpia los despliegues de Dozzle ya existentes (para el contenedor, elimina los volúmenes) antes de recrearlos con el engine-id ya puesto.

## <Icon icon="mdi:help-circle-outline" inline /> FAQ

### Faltan las estadísticas de memoria en modo rootless

Las estadísticas de memoria suelen faltar en despliegues rootless porque el controlador de cgroup `memory` no se delega al slice del usuario por defecto. Comprueba qué está delegado:

```bash
cat /sys/fs/cgroup/user.slice/user-$(id -u).slice/cgroup.controllers
```

Si `memory` no aparece en la salida, activa la delegación con un archivo drop-in:

```bash
sudo mkdir -p /etc/systemd/system/user@.service.d
sudo tee /etc/systemd/system/user@.service.d/delegate.conf <<'EOF'
[Service]
Delegate=cpu cpuset io memory pids
EOF
sudo systemctl daemon-reload
```

Después cierra la sesión y vuelve a entrar (o reinicia) para que el slice del usuario recoja la nueva delegación. Tienes los detalles en el [tutorial de Podman rootless](https://github.com/containers/podman/blob/main/docs/tutorials/rootless_tutorial.md).

### Los healthchecks aparecen como unhealthy

**Problema con podman-compose**: los healthchecks aparecen como unhealthy aunque al ejecutarlos a mano pasen. Es un comportamiento de Podman: los healthchecks no se evalúan automáticamente sin un temporizador de systemd (Quadlet crea uno solo).

Solución alternativa con `podman-compose`:

```bash
# Ejecución manual del healthcheck
podman healthcheck run <container_id>
```

**Quadlet**: `HealthCmd=` espera una línea de comando normal, no el formato JSON `CMD [...]` de Docker:

```ini
HealthCmd=/dozzle healthcheck
```

Las versiones antiguas de `podman-compose` (< 1.5.0) ejecutan todos los healthchecks con `sh`, que no existe en la imagen de Dozzle. Actualiza a una versión actual.

### Visibilidad de contenedores entre usuarios

Podman en modo rootless solo puede acceder a los contenedores del mismo espacio de nombres de usuario. Si ejecutas Dozzle con un usuario, no verá los contenedores de la sesión rootless de otro.

**Solución**: ejecuta Dozzle con ese mismo usuario o usa el modo rootful.
