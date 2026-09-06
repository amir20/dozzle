---
title: Healthcheck
sourceHash: 4ab4dd21a26a
---

# Healthcheck aktivieren

Dozzle bringt den Unterbefehl `dozzle healthcheck` mit. Er ist im Image nicht standardmäßig eingebunden, weil er etwas CPU-Last verursacht. Aktiviere ihn in deiner Compose-Datei:

```yaml
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    ports:
      - 8080:8080
    healthcheck:
      test: ["CMD", "/dozzle", "healthcheck"]
      interval: 3s
      timeout: 30s
      retries: 5
      start_period: 30s
```

## Was geprüft wird

Läuft Dozzle als Server, schickt `dozzle healthcheck` ein HTTP-`GET` an den eigenen Endpunkt `/healthcheck`. Der Endpunkt pingt jeden **lokalen** Docker-Client an (bis zu 3s pro Client) und liefert:

- `200 OK` — mindestens ein lokaler Docker-Client hat geantwortet, **oder** es sind keine lokalen Clients konfiguriert, aber mindestens ein Remote-Agent-Host ist bekannt.
- `500 Internal Server Error` — alle lokalen Clients haben nicht auf den Ping reagiert und es sind keine Agent-Hosts bekannt.

Remote-Agents sind bewusst **kein** Teil des Server-Healthchecks. Ein nicht erreichbarer Agent soll den Haupt-Prozess von Dozzle nicht als ungesund markieren. Jeder Agent kann seinen eigenen Healthcheck anbieten, siehe [Agent-Healthcheck](/de/guide/agent#setting-up-healthcheck).

## Exit-Codes

- `0` — gesund (HTTP 200)
- ungleich null — ungesund, Netzwerkfehler oder eine Antwort ungleich 200. Die fehlgeschlagene URL und der Status werden nach stdout geloggt.

Der Befehl berücksichtigt `--addr` und `--base` und funktioniert damit ohne weitere Konfiguration auch mit eigenen Ports und Basispfaden.

> [!WARNING]
> Der Befehl `healthcheck` funktioniert wegen eines Fehlers in Docker nicht mit dem Flag `--health-cmd`. Nutze wie oben gezeigt den Block `healthcheck` in der `docker-compose.yml`. Details unter [docker/cli#3719](https://github.com/docker/cli/issues/3719).
