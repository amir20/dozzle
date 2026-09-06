---
title: Hostname
sourceHash: 8769ba2c0e47
---

# Dozzles Hostnamen ändern

Dozzles Standardverbindung heißt localhost. Mit der Option `--hostname` lässt sich dieser Name beliebig ändern. Der Wert erscheint im Seitentitel und unter dem Dozzle-Logo.

Damit ändert sich auch die Bezeichnung der `localhost`-Verbindung im Multi-Host-Menü. Das folgende Beispiel setzt die Unterüberschrift mit `--hostname` auf `mywebsite.xyz`.

::: code-group

```sh
docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --hostname mywebsite.xyz
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
      DOZZLE_HOSTNAME: mywebsite.xyz
```

:::

## Multi-Host und Agents

`--hostname` benennt nur den Host um, auf dem **dieser** Dozzle-Prozess läuft. Entfernte [Agents](/de/guide/agent) melden ihre eigenen Namen. Setze `DOZZLE_HOSTNAME` (oder `--hostname`) auf jedem Agent, um zu bestimmen, wie er im Multi-Host-Menü erscheint. Im [Swarm-Modus](/de/guide/swarm-mode) läuft auf jedem Knoten ein eigener Agent, gib also jedem Knoten einen eigenen Hostnamen, um sie unterscheiden zu können.
