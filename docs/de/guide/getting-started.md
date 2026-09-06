---
title: Erste Schritte
sourceHash: 6fa151a446d0
---

# Erste Schritte

Dozzle läuft als einzelner Container. Wähle unten Docker CLI, Docker Compose, Swarm oder Kubernetes.

## <Icon icon="mdi:docker" inline /> Docker als Einzelinstanz

Binde `docker.sock` ein, damit Dozzle die Container lesen kann, hänge ein Volume unter `/data` ein, damit die Einstellungen einen Neustart überstehen, und veröffentliche Port 8080.

::: code-group

```sh [docker run]
docker run -d -v /var/run/docker.sock:/var/run/docker.sock -v dozzle_data:/data -p 8080:8080 amir20/dozzle:latest
```

```yaml [docker-compose.yml]
# Mit docker compose up -d starten
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - dozzle_data:/data
    ports:
      - 8080:8080
    environment:
      # Auskommentieren, um Container-Aktionen zu aktivieren (stop, start, restart). Siehe https://dozzle.dev/guide/actions
      # - DOZZLE_ENABLE_ACTIONS=true
      #
      # Auskommentieren, um Zugriff auf Container-Shells zu erlauben. Siehe https://dozzle.dev/guide/shell
      # - DOZZLE_ENABLE_SHELL=true
      #
      # Auskommentieren, um die Authentifizierung zu aktivieren. Siehe https://dozzle.dev/guide/authentication
      # - DOZZLE_AUTH_PROVIDER=simple
      #
      # Diese Dozzle-Instanz benennen (erscheint im Header und im Multi-Host-Menü). Siehe https://dozzle.dev/guide/hostname
      # - DOZZLE_HOSTNAME=my-server
      #
      # Mit einem oder mehreren entfernten Agents verbinden, um andere Docker-Hosts zu überwachen. Siehe https://dozzle.dev/guide/agent
      # - DOZZLE_REMOTE_AGENT=192.168.1.10:7007,192.168.1.11:7007
      #
      # Nur Container anzeigen, die einem Filter entsprechen. Siehe https://dozzle.dev/guide/filters
      # - DOZZLE_FILTER=label=com.example.app
volumes:
  dozzle_data:
```

:::

Öffne `http://localhost:8080` und fertig. Alles andere, also Aktionen, Shell-Zugriff, Authentifizierung und entfernte Agents, ist optional und standardmäßig aus. Die auskommentierten Umgebungsvariablen in der Compose-Datei verweisen jeweils auf die passende Anleitung.

> [!WARNING]
> Das Einbinden von `docker.sock` gibt Dozzle root-äquivalenten Zugriff auf den Host. Wenn du Dozzle über dein privates Netzwerk hinaus erreichbar machen willst, lies zuerst [Sicherheitshinweise](/de/guide/authentication#security-considerations).

Dozzle benötigt Docker Engine 19.03 oder neuer (API-Version 1.40+). Falls Docker Hub in deinem Netzwerk blockiert ist, hole stattdessen `ghcr.io/amir20/dozzle:latest` aus der [GitHub Container Registry](https://ghcr.io/amir20/dozzle:latest).

## <Icon icon="mdi:hexagon-multiple-outline" inline /> Docker Swarm

Dozzle unterstützt den Swarm-Modus, indem es auf jedem Knoten bereitgestellt wird. Für den Swarm-Modus kannst du folgende Konfiguration nutzen:

```yaml [dozzle-stack.yml]
# Mit docker stack deploy -c dozzle-stack.yml <name> starten
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      - DOZZLE_MODE=swarm
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    ports:
      - 8080:8080
    networks:
      - dozzle
    deploy:
      mode: global
networks:
  dozzle:
    driver: overlay
```

Anschließend kannst du den Stack mit folgendem Befehl bereitstellen:

```bash
docker stack deploy -c dozzle-stack.yml <name>
```

Mehr dazu unter [Swarm-Modus](/de/guide/swarm-mode).

## <Icon icon="mdi:kubernetes" inline /> K8s

Dozzle läuft auch in Kubernetes. Es muss nur auf einem Knoten im Cluster bereitgestellt werden. Du musst `DOZZLE_MODE=k8s` setzen und RBAC für den Zugriff auf Pod-Logs einrichten.

Die vollständige Konfiguration inklusive RBAC-, Deployment- und Service-Manifesten findest du unter [Kubernetes-Modus](/de/guide/k8s).
