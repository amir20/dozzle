---
title: Shell-Zugriff auf Container
sourceHash: 267b2a6665a0
---

# Anhängen und Shell-Befehle ausführen

<Badge type="tip" text="Docker" />
<Badge type="tip" text="K8s" />

Dozzle kann sich an Container anhängen oder Befehle darin ausführen. Es bietet eine webbasierte Oberfläche für die Interaktion mit Docker-Containern: Du kannst dich direkt aus dem Browser an laufende Container anhängen und Befehle ausführen. Besonders nützlich ist das beim Debuggen und bei der Fehlersuche in containerisierten Anwendungen. Die Funktion ist standardmäßig **deaktiviert**, da sie ein Sicherheitsrisiko darstellen kann. Setze die Umgebungsvariable `DOZZLE_ENABLE_SHELL` auf `true`, um sie zu aktivieren.

::: code-group

```sh
docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --enable-shell
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    ports:
      - 8080:8080
    environment:
      DOZZLE_ENABLE_SHELL: true
```

:::

> [!NOTE]
> Der Shell-Zugriff sollte mit allen Container-Typen funktionieren, also mit Docker, Kubernetes und anderen Orchestrierungsplattformen.

## <Icon icon="mdi:shield-lock-outline" inline /> Sicherheit

Jeder, der die Dozzle-Oberfläche erreicht, kann eine Shell in deinen Containern öffnen, gleichbedeutend mit `docker exec`. Bevor du `--enable-shell` bei einem öffentlich erreichbaren Dozzle aktivierst, stelle eine [Authentifizierung](/de/guide/authentication) davor. Mit rollenbasierten Berechtigungen lässt sich der Shell-Zugriff auf bestimmte Benutzer beschränken.

## <Icon icon="mdi:kubernetes" inline /> Kubernetes

Im k8s-Modus läuft der Shell-Zugriff über die Kubernetes-API statt über `docker exec`. Der Ziel-Pod muss eine ausführbare Shell enthalten (`/bin/sh`, `/bin/bash` usw.). Bei minimalen Images auf Basis von `FROM scratch` oder Distroless-Images ohne Shell ist kein Zugriff möglich.
