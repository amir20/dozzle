---
title: Podman
---

# Podman

Dozzle supports Podman through its Docker-compatible socket interface. Two known differences from Docker that affect setup: memory stats are often missing in rootless/Quadlet deployments (cgroup delegation), and Podman doesn't generate an engine-id. This guide covers standalone mode (local monitoring) and agent mode (remote monitoring via a central Dozzle server).

## Deployment Options

| Mode           | Use Case                          | Setup Complexity |
| -------------- | --------------------------------- | ---------------- |
| **Standalone** | Single host log viewing           | Simple           |
| **Agent**      | Multi-host centralized monitoring | Moderate         |

### Deployment Methods

Podman offers several launch approaches:

| Method            | Auto-start | Memory Stats | Healthchecks | Best For    |
| ----------------- | ---------- | ------------ | ------------ | ----------- |
| CLI               | Manual     | ✓            | ✓            | Development |
| `podman-compose`  | ✗          | ✓            | ✗            | Testing     |
| Quadlet (systemd) | ✓          | ✗\*          | ✓            | Production  |

\*Memory stats are typically unavailable in rootless mode unless cgroup v2 memory delegation is enabled. See the FAQ at the bottom of this page.

---

# <Icon icon="mdi:monitor-dashboard" inline /> Standalone Mode

Run Dozzle as a standalone service to monitor local Podman containers.

## <Icon icon="mdi:shield-account-outline" inline /> Rootful Setup

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

## <Icon icon="mdi:account-outline" inline /> Rootless Setup

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

**Important**: A Dozzle bound to one user's rootless socket only sees that user's containers. Other users' rootless containers live in separate namespaces and won't appear.

## <Icon icon="mdi:rocket-launch-outline" inline /> Quadlet Deployment

Quadlet enables systemd-native container management. Create a `.container` file at `~/.config/containers/systemd/dozzle.container`:

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

Enable and start:

```bash
systemctl --user daemon-reload
systemctl --user enable --now dozzle.service
```

For multi-user systems, drop the same file into each user's `~/.config/containers/systemd/` and pick a distinct host port per user (e.g. `PublishPort=3001:8080`). Each instance only sees that user's rootless containers.

> [!NOTE] Quadlet generates a systemd timer for healthchecks. `podman-compose` does not, so healthchecks won't run on a schedule there; trigger them manually with `podman healthcheck run NAME` if needed.

---

# <Icon icon="mdi:lan-connect" inline /> Agent Mode

Run Dozzle as an agent on remote Podman hosts for centralized monitoring via a main Dozzle server. Agents communicate with the main server via gRPC.

## <Icon icon="mdi:cog-outline" inline /> Agent Setup

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

> [!NOTE] The Dozzle image's entrypoint is `/dozzle`, so `agent` goes in `Exec=` (the command), not `Entrypoint=`.

Enable and start:

```bash
systemctl --user daemon-reload
systemctl --user enable dozzle-agent.service
systemctl --user start dozzle-agent.service
```

---

# <Icon icon="mdi:server-network" inline /> Main Server with Remote Agents

Configure the main Dozzle server to connect to agents on remote Podman hosts.

## <Icon icon="mdi:cog" inline /> Server Configuration

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
  -e DOZZLE_REMOTE_AGENT="host1.example.com:7007,host2.example.com:7007" \
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

> [!NOTE] `WantedBy=multi-user.target` only applies to system units. For `systemctl --user` units, use `default.target`.

---

# <Icon icon="mdi:tune" inline /> Additional Configuration

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

## <Icon icon="mdi:help-circle-outline" inline /> FAQ

### Memory Stats Missing in Rootless Mode

Memory stats are usually missing in rootless deployments because the `memory` cgroup controller isn't delegated to the user slice by default. Check what's delegated:

```bash
cat /sys/fs/cgroup/user.slice/user-$(id -u).slice/cgroup.controllers
```

If `memory` is not in the output, enable delegation via a drop-in:

```bash
sudo mkdir -p /etc/systemd/system/user@.service.d
sudo tee /etc/systemd/system/user@.service.d/delegate.conf <<'EOF'
[Service]
Delegate=cpu cpuset io memory pids
EOF
sudo systemctl daemon-reload
```

Then log out and back in (or reboot) for the user slice to pick up the new delegation. See the [Podman rootless tutorial](https://github.com/containers/podman/blob/main/docs/tutorials/rootless_tutorial.md) for details.

### Healthchecks Reported as Unhealthy

**podman-compose issue**: Healthchecks are reported as unhealthy even though manual runs pass. This is a Podman behavior where healthchecks aren't automatically evaluated without a systemd timer (Quadlet generates one automatically).

Workaround with `podman-compose`:

```bash
# Manual healthcheck run
podman healthcheck run <container_id>
```

**Quadlet**: `HealthCmd=` takes a plain command line, not the Docker `CMD [...]` JSON form:

```ini
HealthCmd=/dozzle healthcheck
```

Older `podman-compose` (< 1.5.0) runs all healthchecks via `sh`, which doesn't exist in the Dozzle image. Update to a current version.

### Cross-user Container Visibility

Rootless Podman can only access containers in the same user namespace. If running Dozzle as one user, it cannot see containers from another user's rootless session.

**Solution**: Either run Dozzle as the same user or use rootful mode.
