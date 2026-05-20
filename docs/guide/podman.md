---
title: Podman
---

# Podman

Dozzle supports Podman through its Docker-compatible socket interface. However, Podman is not 100% compatible with Docker, key differences include memory stats reporting (especially in rootless/Quadlet deployments) and the lack of automatic engine-id generation. This guide covers both Web GUI mode (standalone, local monitoring) and Agent mode (remote monitoring via a central Dozzle server), with notes on these compatibility considerations.

## Deployment Options

| Mode | Use Case | Setup Complexity |
|------|----------|------------------|
| **Web GUI** | Single host log viewing | Simple |
| **Agent** | Multi-host centralized monitoring | Moderate |

### Deployment Methods

Podman offers several launch approaches:

| Method | Auto-start | Memory Stats | Healthchecks | Best For |
|--------|-----------|---------|------------|----------|
| CLI | Manual | ✓ | ✓ | Development |
| `podman-compose` | ✗ | ✓ | ✗* | Testing |
| Quadlet (systemd) | ✓ | ✗** | ✓ | Production |

*Note: Healthchecks reported incorrectly; manual runs succeed  
**Memory stats typically unavailable due to cgroups v2 delegation in rootless mode

---

# Web GUI Mode

Run Dozzle as a standalone service to monitor local Podman containers.

## Rootful Setup

For system-wide Podman daemon:

```bash
# Enable and start the Podman socket
sudo systemctl enable podman.socket
sudo systemctl start podman.socket

# Dozzle can connect via the Docker socket
podman run -v /run/podman/podman.sock:/var/run/docker.sock:ro \
  -p 3000:8080 \
  ghcr.io/amir20/dozzle:latest
```

## Rootless Setup

Rootless Podman isolates containers to a user namespace:

```bash
# Start user-level socket (runs automatically with user session)
systemctl --user enable podman.socket
systemctl --user start podman.socket

# For a user named 'appuser', Dozzle can connect via:
podman run -v /run/user/$(id -u appuser)/podman/podman.sock:/var/run/docker.sock:ro \
  -p 3000:8080 \
  ghcr.io/amir20/dozzle:latest
```

**Important**: Rootless mode can only access containers in the same user namespace. Root cannot see containers of other users.

## Per-User Namespace Deployment

For multi-user systems, run a separate Dozzle instance per user namespace. Each instance monitors only that user's containers with dedicated ports:

```bash
# For user 'appuser' - runs on port 3000
sudo -u appuser podman run -d \
  --name dozzle-appuser \
  -v /run/user/$(id -u appuser)/podman/podman.sock:/var/run/docker.sock:ro \
  -p 3000:8080 \
  ghcr.io/amir20/dozzle:latest

# For user 'webuser' - runs on port 3001
sudo -u webuser podman run -d \
  --name dozzle-webuser \
  -v /run/user/$(id -u webuser)/podman/podman.sock:/var/run/docker.sock:ro \
  -p 3001:8080 \
  ghcr.io/amir20/dozzle:latest
```

Or with Quadlet, create per-user `.container` files:

```ini
# ~/.config/containers/systemd/dozzle.container (for each user)
[Unit]
Description=Dozzle Log Viewer for %u
After=network-online.target
Wants=network-online.target

[Container]
Image=ghcr.io/amir20/dozzle:latest
Ports=3000:8080
Volumes=/run/user/%U/podman/podman.sock:/var/run/docker.sock:ro

HealthCmd=CMD /dozzle healthcheck
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

This approach ensures each user only sees and manages their own containers, with no cross-namespace visibility issues.

## Quadlet Deployment (Web GUI)

Quadlet enables systemd-native container management. Create a `.container` file in `~/.config/containers/systemd/`:

```ini
# dozzle.container
[Unit]
Description=Dozzle Log Viewer
After=network-online.target
Wants=network-online.target

[Container]
Image=ghcr.io/amir20/dozzle:latest
Ports=3000:8080
Volumes=/run/user/%U/podman/podman.sock:/var/run/docker.sock:ro

HealthCmd=CMD /dozzle healthcheck
HealthInterval=5s
HealthTimeout=10s
HealthRetries=5
HealthStartPeriod=15s

[Service]
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target default.target
```

Enable and start:

```bash
systemctl --user daemon-reload
systemctl --user enable dozzle.service
systemctl --user start dozzle.service
```

> [!NOTE] Quadlet generates healthcheck as a systemd timer. Healthchecks don't run automatically with `podman-compose`; use `podman healthcheck run` manually if needed.

---

# Agent Mode

Run Dozzle as an agent on remote Podman hosts for centralized monitoring via a main Dozzle server. Agents communicate with the main server via gRPC.

## Agent Setup

### Prerequisites

- Open port 7007 on agent host
- Network connectivity between main server and agent

### Start Dozzle Agent

Run Dozzle in agent mode on remote Podman hosts:

```bash
# Rootful agent
podman run -d \
  --name dozzle-agent \
  -v /run/podman/podman.sock:/var/run/docker.sock:ro \
  -p 7007:7007 \
  ghcr.io/amir20/dozzle:latest agent
```

```bash
# Rootless agent (for user 'appuser')
sudo -u appuser podman run -d \
  --name dozzle-agent \
  -v /run/user/$(id -u appuser)/podman/podman.sock:/var/run/docker.sock:ro \
  -p 7007:7007 \
  ghcr.io/amir20/dozzle:latest agent
```

### Quadlet Agent Deployment

Create a `.container` file for the agent:

```ini
# dozzle-agent.container
[Unit]
Description=Dozzle Agent
After=network-online.target
Wants=network-online.target

[Container]
Image=ghcr.io/amir20/dozzle:latest
Ports=7007:7007
Volumes=/run/user/%U/podman/podman.sock:/var/run/docker.sock:ro
Entrypoint=agent

HealthCmd=CMD /dozzle healthcheck
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

Enable and start:

```bash
systemctl --user daemon-reload
systemctl --user enable dozzle-agent.service
systemctl --user start dozzle-agent.service
```

---

# Main Server with Remote Agents

Configure the main Dozzle server to connect to agents on remote Podman hosts.

## Server Configuration

Run the main Dozzle server with agent endpoints:

```bash
podman run -d \
  --name dozzle \
  -p 3000:8080 \
  ghcr.io/amir20/dozzle:latest \
  --agent "host1.example.com:7007" \
  --agent "host2.example.com:7007"
```

Or with environment variables:

```bash
podman run -d \
  --name dozzle \
  -e DOZZLE_AGENT_FILTER="host1.example.com:7007,host2.example.com:7007" \
  -p 3000:8080 \
  ghcr.io/amir20/dozzle:latest
```

### Quadlet Main Server with Agents

```ini
# dozzle-server.container
[Unit]
Description=Dozzle Server with Remote Agents
After=network-online.target
Wants=network-online.target

[Container]
Image=ghcr.io/amir20/dozzle:latest
Ports=3000:8080

# Agent endpoints
Exec=--agent host1.example.com:7007 \
     --agent host2.example.com:7007

HealthCmd=CMD /dozzle healthcheck
HealthInterval=5s
HealthTimeout=10s
HealthRetries=5
HealthStartPeriod=15s

[Service]
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target default.target
```

---

# Additional Configuration

## <Icon icon="mdi:identifier" inline /> Engine-ID Setup

Podman doesn't create an engine-id like Docker. Create one to avoid "host not found" errors:

### Using uuidgen

```bash
# Create directory if needed
sudo mkdir -p /var/lib/docker

# Generate UUID
sudo sh -c 'uuidgen > /var/lib/docker/engine-id'

# Verify
cat /var/lib/docker/engine-id
```

### Using Ansible

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

> [!WARNING] Clean up existing Dozzle deployments (stop container, remove volumes) before recreating with the engine-id in place.

## FAQ

### Memory Stats Missing with Quadlet

Memory stats typically unavailable in rootless Quadlet deployments due to cgroups v2 delegation. Check if memory is delegated:

```bash
# For rootless, check user slice controllers
cat /sys/fs/cgroup/user.slice/user-$(id -u).slice/cgroup.controllers

# Memory must be in the output (e.g., "cpuset cpu.max io memory...")
systemctl --user status
```

If `memory` isn't listed, configure cgroup delegation in `/etc/systemd/system.conf`:

```ini
DefaultCgroupsMode=unified
DefaultMemoryAccounting=yes
```

Then reboot.

### Healthchecks Reported as Unhealthy

**podman-compose issue**: Healthchecks are reported as unhealthy even though manual runs pass. This is a Podman behavior where healthchecks aren't automatically evaluated without a systemd timer (Quadlet generates one automatically).

Workaround with `podman-compose`:

```bash
# Manual healthcheck run
podman healthcheck run <container_id>
```

**Quadlet**: Use the correct format in `.container` files:

```ini
HealthCmd=CMD /dozzle healthcheck
# NOT: HealthCmd=CMD ["executable", "param1", "param2"]
```

Older `podman-compose` (< 1.5.0) runs all healthchecks with `sh`, which may not exist in the Dozzle image. Update to the latest version.

### Cross-user Container Visibility

Rootless Podman can only access containers in the same user namespace. If running Dozzle as one user, it cannot see containers from another user's rootless session.

**Solution**: Either run Dozzle as the same user or use rootful mode.
