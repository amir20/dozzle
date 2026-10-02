---
title: Host-Metriken
sourceHash: e10c0661efbb
---

# Host-Metriken

Die Host-Karte kann drei Werte für die Maschine anzeigen, auf der Docker läuft. Sie stehen in einem kleinen Kasten rechts im Kopf der Karte, gekennzeichnet mit dem Symbol des Hosts, damit man sie nicht mit den Container-Summen in den Anzeigen für CPU und Arbeitsspeicher darunter verwechselt:

- **Laufzeit**, wie lange der Host schon läuft
- **Last**, der Load Average über 1 Minute (beim Darüberfahren mit der Maus erscheinen die Werte für 5 und 15 Minuten)
- **Datenträger**, wie voll das Dateisystem mit dem Datenverzeichnis von Docker ist, als kleiner Balken, der ab 70% gelb und ab 90% rot wird (beim Darüberfahren mit der Maus erscheinen belegter und gesamter Speicher)

Solange ein Tab offen ist, werden sie alle 15 Sekunden aktualisiert. Jeder Wert erscheint nur, wenn Dozzle dafür einen echten Wert lesen kann. Bei einer Standardinstallation fehlt der Kasten also womöglich ganz oder zeigt nur einen Teil davon.

## Dozzle in einem Container betreiben

In einem Container beschreibt `/proc` den Container und nicht den Host. Dozzle gibt die Zahlen des Containers nicht als die des Hosts aus, deshalb bleiben Last und Laufzeit ausgeblendet, bis du das `/proc` des Hosts unter `/host/proc` einbindest.

Die Datenträgerbelegung wird aus dem Datenverzeichnis von Docker gelesen (`docker info --format '{{.DockerRootDir}}'`, meist `/var/lib/docker`). Binde es unter demselben Pfad ein, dann erscheint auch der Datenträgerwert.

::: code-group

```sh
docker run -d \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /proc:/host/proc:ro \
  -v /var/lib/docker:/var/lib/docker:ro \
  -p 8080:8080 amir20/dozzle:latest
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /proc:/host/proc:ro
      - /var/lib/docker:/var/lib/docker:ro
    ports:
      - 8080:8080
```

:::

Beide Mounts sind optional und unabhängig voneinander. Mit dem eingebundenen Datenverzeichnis bekommt Dozzle Lesezugriff auf das Dateisystem jedes Containers. Wenn du nur Last und Laufzeit willst, lass es also weg.

## Dozzle nativ betreiben

Ein Dozzle-Binary, das direkt auf dem Host läuft, liest `/proc` so wie es ist. Last und Laufzeit brauchen also keine Einrichtung. Der Datenträgerwert funktioniert, solange der Benutzer, unter dem Dozzle läuft, das Datenverzeichnis sehen kann.

## Einschränkungen

- Vorerst liefert nur der lokale Host Metriken. Hosts, die über einen [Agent](/de/guide/agent) oder als [entfernter Host](/de/guide/remote-hosts) verbunden sind, zeigen CPU und Arbeitsspeicher wie bisher, aber ohne diese Zeile.
- Wenn `DOCKER_HOST` auf eine andere Maschine zeigt (`tcp://` oder `ssh://`), lässt Dozzle die Werte weg, denn das eigene `/proc` und die eigenen Datenträger sagen nichts über diese Engine aus.
- Docker Desktop betreibt die Engine in einer VM. Ein natives Dozzle-Binary unter macOS oder Windows hat kein `/proc` zum Lesen, und Dozzle in einem Container meldet dort die Werte der VM, nicht die deines Rechners.
