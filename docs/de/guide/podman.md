---
title: Podman
sourceHash: bbfd1745527f
---

# Podman

Dozzle unterstützt Podman über dessen Docker-kompatible Socket-Schnittstelle. Ein bekannter Unterschied zu Docker wirkt sich auf die Einrichtung aus: In rootless- und Quadlet-Deployments fehlen häufig die Speicher-Statistiken (cgroup-Delegation). Diese Anleitung behandelt den Standalone-Modus (lokale Überwachung) und den Agent-Modus (entfernte Überwachung über einen zentralen Dozzle-Server).

## Deployment-Varianten

| Modus          | Anwendungsfall                      | Aufwand |
| -------------- | ----------------------------------- | ------- |
| **Standalone** | Logs eines einzelnen Hosts ansehen  | Einfach |
| **Agent**      | Zentrale Überwachung mehrerer Hosts | Mittel  |

### Startmethoden

Podman bietet mehrere Wege, Container zu starten:

| Methode           | Autostart | Speicher-Stats | Healthchecks | Am besten für |
| ----------------- | --------- | -------------- | ------------ | ------------- |
| CLI               | Manuell   | ✓              | ✓            | Entwicklung   |
| `podman-compose`  | ✗         | ✓              | ✗            | Tests         |
| Quadlet (systemd) | ✓         | ✗\*            | ✓            | Produktion    |

\*Speicher-Statistiken sind im rootless-Modus in der Regel nicht verfügbar, solange die cgroup-v2-Delegation für `memory` nicht aktiviert ist. Siehe die FAQ am Ende dieser Seite.

---

# <Icon icon="mdi:monitor-dashboard" inline /> Standalone-Modus

Betreibe Dozzle als eigenständigen Dienst, um lokale Podman-Container zu überwachen.

## <Icon icon="mdi:shield-account-outline" inline /> Rootful-Einrichtung

Für den systemweiten Podman-Daemon:

```bash
# Podman-Socket aktivieren und starten
sudo systemctl enable podman.socket
sudo systemctl start podman.socket

# Dozzle kann sich über den Docker-Socket verbinden
podman run -v /run/podman/podman.sock:/var/run/docker.sock:ro \
  -p 3000:8080 \
  ghcr.io/amir20/dozzle:latest
```

## <Icon icon="mdi:account-outline" inline /> Rootless-Einrichtung

Rootless-Podman isoliert Container in einem User-Namespace:

```bash
# Socket auf Benutzerebene starten (läuft automatisch mit der Benutzersitzung)
systemctl --user enable podman.socket
systemctl --user start podman.socket

# Für einen Benutzer namens 'appuser' verbindet sich Dozzle so:
podman run -v /run/user/$(id -u appuser)/podman/podman.sock:/var/run/docker.sock:ro \
  -p 3000:8080 \
  ghcr.io/amir20/dozzle:latest
```

**Wichtig**: Ein Dozzle, das am rootless-Socket eines Benutzers hängt, sieht nur dessen Container. Die rootless-Container anderer Benutzer liegen in eigenen Namespaces und tauchen nicht auf.

## <Icon icon="mdi:rocket-launch-outline" inline /> Quadlet-Deployment

Quadlet ermöglicht Container-Verwaltung nativ über systemd. Lege eine `.container`-Datei unter `~/.config/containers/systemd/dozzle.container` an:

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

Aktivieren und starten:

```bash
systemctl --user daemon-reload
systemctl --user enable --now dozzle.service
```

Auf Mehrbenutzersystemen legst du dieselbe Datei in das Verzeichnis `~/.config/containers/systemd/` jedes Benutzers und wählst pro Benutzer einen eigenen Host-Port (z. B. `PublishPort=3001:8080`). Jede Instanz sieht nur die rootless-Container des jeweiligen Benutzers.

> [!NOTE] Quadlet erzeugt einen systemd-Timer für Healthchecks. `podman-compose` tut das nicht, dort laufen Healthchecks also nicht nach Zeitplan; stoße sie bei Bedarf manuell mit `podman healthcheck run NAME` an.

---

# <Icon icon="mdi:lan-connect" inline /> Agent-Modus

Betreibe Dozzle als Agent auf entfernten Podman-Hosts, um sie zentral über einen Haupt-Dozzle-Server zu überwachen. Agents kommunizieren per gRPC mit dem Hauptserver.

## <Icon icon="mdi:cog-outline" inline /> Agent einrichten

### Voraussetzungen

- Port 7007 auf dem Agent-Host öffnen
- Netzwerkverbindung zwischen Hauptserver und Agent

### Dozzle-Agent starten

Starte Dozzle im Agent-Modus auf den entfernten Podman-Hosts:

```bash
# Rootful-Agent
podman run -d \
  --name dozzle-agent \
  -v /run/podman/podman.sock:/var/run/docker.sock:ro \
  -p 7007:7007 \
  ghcr.io/amir20/dozzle:latest agent
```

```bash
# Rootless-Agent (für den Benutzer 'appuser')
sudo -u appuser podman run -d \
  --name dozzle-agent \
  -v /run/user/$(id -u appuser)/podman/podman.sock:/var/run/docker.sock:ro \
  -p 7007:7007 \
  ghcr.io/amir20/dozzle:latest agent
```

### Quadlet-Deployment für den Agent

Lege eine `.container`-Datei für den Agent an:

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

> [!NOTE] Der Entrypoint des Dozzle-Images ist `/dozzle`, `agent` gehört also in `Exec=` (den Befehl), nicht in `Entrypoint=`.

Aktivieren und starten:

```bash
systemctl --user daemon-reload
systemctl --user enable dozzle-agent.service
systemctl --user start dozzle-agent.service
```

---

# <Icon icon="mdi:server-network" inline /> Hauptserver mit entfernten Agents

Konfiguriere den Haupt-Dozzle-Server so, dass er sich mit den Agents auf den entfernten Podman-Hosts verbindet.

## <Icon icon="mdi:cog" inline /> Serverkonfiguration

Starte den Haupt-Dozzle-Server mit den Agent-Endpunkten:

```bash
podman run -d \
  --name dozzle \
  -p 3000:8080 \
  ghcr.io/amir20/dozzle:latest \
  --remote-agent "host1.example.com:7007" \
  --remote-agent "host2.example.com:7007"
```

Oder mit Umgebungsvariablen:

```bash
podman run -d \
  --name dozzle \
  -e DOZZLE_REMOTE_AGENT="host1.example.com:7007,host2.example.com:7007" \
  -p 3000:8080 \
  ghcr.io/amir20/dozzle:latest
```

### Quadlet-Hauptserver mit Agents

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

> [!NOTE] `WantedBy=multi-user.target` gilt nur für System-Units. Für Units unter `systemctl --user` nimmst du `default.target`.

---

# <Icon icon="mdi:tune" inline /> Weitere Konfiguration

## <Icon icon="mdi:identifier" inline /> Host-IDs

Hier ist nichts einzurichten. Dozzle ermittelt Podman-Host-IDs selbst. Dieser Abschnitt existiert nur, weil frühere Versionen dieser Seite dazu aufgefordert haben, eine Datei anzulegen, die nie eine Wirkung hatte.

Docker identifiziert eine Engine über die UUID in `/var/lib/docker/engine-id`, die einmalig beim ersten Start des Daemons geschrieben wird. Podman läuft ohne Daemon und führt keine solche Identität, deshalb füllt der Docker-kompatible `/info`-Endpunkt dieses Feld bei jedem Aufruf mit einer neuen zufälligen UUID. Das lässt sich selbst nachprüfen:

```sh
curl -s --unix-socket /run/user/$(id -u)/podman/podman.sock "http://d/v1.40/info" | jq .ID
curl -s --unix-socket /run/user/$(id -u)/podman/podman.sock "http://d/v1.40/info" | jq .ID
```

Zwei verschiedene UUIDs, und `/var/lib/docker/engine-id` anzulegen ändert daran nichts, weil Podman die Datei nie liest. Dozzle leitet stattdessen eine stabile ID aus dem Hostnamen und dem Storage-Pfad der Container ab. So bleibt ein Host über Neustarts hinweg erkennbar, und zwei rootless-Benutzer auf derselben Maschine bleiben unterscheidbar.

> [!WARNING] Wenn du auf einem Podman-Host nach den alten Anweisungen `/var/lib/docker/engine-id` angelegt hast, kannst du die Datei löschen.

### Wenn IDs kollidieren

Zwei Podman-Hosts, die sowohl Hostnamen als auch Storage-Pfad teilen, erhalten dieselbe abgeleitete ID, und Dozzle verwirft einen davon als Duplikat. Hostnamen sind normalerweise verschieden, dafür braucht es also geklonte VMs oder eine Flotte, in der nie ein Hostname gesetzt wurde. Setze auf einem der beiden `DOZZLE_HOST_ID`, um den Konflikt aufzulösen:

```ini
# dozzle-agent.container
[Container]
Environment=DOZZLE_HOST_ID=web-01
```

Der Wert darf aus Buchstaben, Ziffern, Bindestrichen, Unterstrichen und Punkten bestehen. Er muss über alle Hosts hinweg eindeutig sein und für die Lebensdauer des Hosts gleich bleiben.

## <Icon icon="mdi:help-circle-outline" inline /> FAQ

### Speicher-Statistiken fehlen im Rootless-Modus

In rootless-Deployments fehlen die Speicher-Statistiken meist, weil der cgroup-Controller `memory` standardmäßig nicht an die User-Slice delegiert wird. Prüfe, was delegiert ist:

```bash
cat /sys/fs/cgroup/user.slice/user-$(id -u).slice/cgroup.controllers
```

Steht `memory` nicht in der Ausgabe, aktiviere die Delegation über ein Drop-in:

```bash
sudo mkdir -p /etc/systemd/system/user@.service.d
sudo tee /etc/systemd/system/user@.service.d/delegate.conf <<'EOF'
[Service]
Delegate=cpu cpuset io memory pids
EOF
sudo systemctl daemon-reload
```

Melde dich danach ab und wieder an (oder starte neu), damit die User-Slice die neue Delegation übernimmt. Details im [Podman-Rootless-Tutorial](https://github.com/containers/podman/blob/main/docs/tutorials/rootless_tutorial.md).

### Healthchecks werden als unhealthy gemeldet

**Problem mit podman-compose**: Healthchecks werden als unhealthy gemeldet, obwohl manuelle Läufe erfolgreich sind. Das ist Verhalten von Podman: ohne systemd-Timer werden Healthchecks nicht automatisch ausgewertet (Quadlet erzeugt einen solchen Timer automatisch).

Behelfslösung mit `podman-compose`:

```bash
# Healthcheck manuell ausführen
podman healthcheck run <container_id>
```

**Quadlet**: `HealthCmd=` erwartet eine einfache Befehlszeile, nicht die JSON-Form `CMD [...]` von Docker:

```ini
HealthCmd=/dozzle healthcheck
```

Ältere Versionen von `podman-compose` (< 1.5.0) führen alle Healthchecks über `sh` aus, das es im Dozzle-Image nicht gibt. Aktualisiere auf eine aktuelle Version.

### Sichtbarkeit von Containern über Benutzergrenzen hinweg

Rootless-Podman kann nur auf Container im selben User-Namespace zugreifen. Läuft Dozzle als ein Benutzer, sieht es keine Container aus der rootless-Sitzung eines anderen Benutzers.

**Lösung**: Betreibe Dozzle als denselben Benutzer oder nutze den Rootful-Modus.
