---
title: Swarm-Modus
sourceHash: f73672e30c85
---

# Swarm-Modus

<Badge type="warning" text="Docker Only" />

Dozzle unterstützt den Docker-Swarm-Modus. Im Swarm-Modus erkennt Dozzle Services und eigene Gruppen automatisch. Intern nutzt Dozzle die Swarm-API nicht, da sie [eingeschränkt](https://github.com/moby/moby/issues/33183) ist. Stattdessen gruppiert Dozzle selbst anhand von Swarm-Labels. Zusätzlich führt Dozzle die Statistiken der Container einer Gruppe zusammen. Damit siehst du Logs und Statistiken aller Container einer Gruppe in einer Ansicht. Das bedeutet allerdings, dass Dozzle auf jedem Host eingerichtet sein muss.

## <Icon icon="mdi:cogs" inline /> Wie funktioniert das?

Im Swarm-Modus baut Dozzle ein gesichertes Mesh-Netzwerk zwischen allen Knoten des Swarms auf. Über dieses Netzwerk kommunizieren die Dozzle-Instanzen miteinander. Das Mesh-Netzwerk basiert auf [mTLS](https://www.cloudflare.com/learning/access-management/what-is-mutual-tls) mit einem privaten TLS-Zertifikat. Die gesamte Kommunikation zwischen den Dozzle-Instanzen ist damit verschlüsselt und kann überall betrieben werden.

Dozzle unterstützt Docker-[Stacks](https://docs.docker.com/reference/cli/docker/stack/deploy/), [Services](https://docs.docker.com/engine/swarm/how-swarm-mode-works/services/) und eigene Gruppen, um Logs zusammenzuführen. Zur Gruppierung der Container werden die Labels `com.docker.stack.namespace` und `com.docker.compose.project` genutzt. Bei Services verwendet Dozzle den Service-Namen als Gruppennamen, also `com.docker.swarm.service.name`.

## <Icon icon="mdi:rocket-launch-outline" inline /> Wie aktiviert man den Swarm-Modus?

Um Dozzle auf jedem Knoten im Swarm bereitzustellen, nutzt du `mode: global`. Damit landet Dozzle auf jedem Knoten des Swarms. Hier ein Beispiel mit Docker Stack:

```yml
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      - DOZZLE_MODE=swarm
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /opt/dozzle/data:/data
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

Beachte, dass die Umgebungsvariable `DOZZLE_MODE` auf `swarm` gesetzt ist. Damit weiß Dozzle, dass es andere Dozzle-Instanzen im Swarm automatisch finden soll. Das `overlay`-Netzwerk bildet das Mesh-Netzwerk zwischen den Dozzle-Instanzen.

Das Volume `/data` wird eingebunden, um Dozzles Konfiguration (Benachrichtigungen, Cloud-Einstellungen, eigene Stacks) dauerhaft zu speichern. Da Dozzle global auf jedem Knoten läuft, solltest du auf jedem Knoten einen Host-Pfad einbinden, damit jede Instanz ihren lokalen Zustand über Neustarts hinweg behält.

> [!WARNING]
> Socket-Proxy lässt sich im Docker-Swarm-Modus nicht verwenden. Diese Einschränkung kommt von Docker selbst, nicht von Dozzle. Im Swarm-Modus können Services nur mit anderen Services kommunizieren, Dozzle braucht aber direkte Verbindungen zu einzelnen Proxy-Instanzen, was nicht unterstützt wird. Wenn du eine Lösung für Socket-Proxy im Swarm-Modus hast, hören wir gerne davon!

## <Icon icon="mdi:shield-lock-outline" inline /> Einfache Authentifizierung im Swarm-Modus einrichten

Für die einfache Authentifizierung kannst du die Datei `users.yml` in einem Docker-Secret ablegen. Hier ein Beispiel mit Docker Stack:

```yml
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      - DOZZLE_LEVEL=debug
      - DOZZLE_MODE=swarm
      - DOZZLE_AUTH_PROVIDER=simple
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /opt/dozzle/data:/data
    secrets:
      - source: users
        target: /data/users.yml

    ports:
      - "8080:8080"
    networks:
      - dozzle
    deploy:
      mode: global

networks:
  dozzle:
    driver: overlay
secrets:
  users:
    file: users.yml
```

In diesem Beispiel liegt die Datei `users.yml` in einem Docker-Secret. Sie entspricht dem Beispiel unter [einfache Authentifizierung](/de/guide/authentication#generating-users-yml).

## <Icon icon="mdi:server-plus-outline" inline /> Eigenständige Agents zum Swarm-Modus hinzufügen

Dozzle kann im Swarm-Modus auch eigenständige [Agents](/de/guide/agent) einbinden.

Füge den [entfernten Agent](/de/guide/agent#how-to-connect-to-an-agent) einfach genauso zu deiner Swarm-Compose-Datei hinzu, wie du es sonst tun würdest.

> [!NOTE]
> Entfernte Agents werden unterstützt, entfernte Verbindungen wie Socket-Proxy dagegen nicht.

```yml
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      - DOZZLE_MODE=swarm
      - DOZZLE_REMOTE_AGENT=agent:7007
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /opt/dozzle/data:/data
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

Die entfernten Agents erscheinen nun neben den anderen Knoten in Dozzle.
