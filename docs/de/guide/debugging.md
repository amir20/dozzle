---
title: Fehlersuche
sourceHash: 2cb9b9a633e2
---

# Fehlersuche mit Logs

Standardmäßig loggt Dozzle auf der Stufe `info` und ist damit bewusst zurückhaltend. Wenn etwas nicht funktioniert, erhöhe die Ausführlichkeit über das Flag `--level` oder die Umgebungsvariable `DOZZLE_LEVEL`.

| Stufe   | Wann sie sinnvoll ist                                                             |
| ------- | --------------------------------------------------------------------------------- |
| `info`  | Standard. Startdetails, Fehler und Warnungen.                                     |
| `debug` | Diagnose auf Request-Ebene, Auth-Entscheidungen, Agent-Verbindungen, Konfig-Dump. |
| `trace` | Alles. Einzelne Log-Events, Beacon-Payloads, gRPC-Frames. Sehr gesprächig.        |

Dozzle schreibt alle Logs nach `stdout`, `docker logs dozzle` ist also die richtige Stelle zum Mitlesen.

```yaml
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    ports:
      - 8080:8080
    environment:
      DOZZLE_LEVEL: debug
```

## Einen Fehler melden

Wenn du glaubst, auf einen Fehler gestoßen zu sein, erstelle bitte ein Issue unter [github.com/amir20/dozzle/issues](https://github.com/amir20/dozzle/issues). Gib dabei an:

- Dozzle-Version (sichtbar in der Fußzeile der Oberfläche oder über `dozzle --version`)
- Betriebsmodus: server, swarm, k8s oder agent
- Docker- oder Kubernetes-Version
- Relevante Log-Ausgabe auf Stufe `debug` oder `trace`
- Schritte zum Reproduzieren, idealerweise mit einer minimalen `docker-compose.yml`

Je mehr Kontext im ersten Bericht steht, desto schneller lässt er sich einordnen.
