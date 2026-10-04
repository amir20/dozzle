---
title: Umstieg von Watchtower
sourceHash: 3c48dbce256f
---

# Umstieg von Watchtower

Dozzle kann, was Watchtower kann: deine Container nach Zeitplan auf neuere Images prüfen und sie aktualisieren. Jedes Update wird beobachtet, und bleibt der neue Container nicht stabil, kommt der alte zurück. Diese Seite ordnet die Einstellungen von Watchtower denen von Dozzle zu.

## Einschalten

Dozzle braucht eingeschaltete Aktionen und `/data` auf einem Volume:

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - dozzle-data:/data
    ports:
      - 8080:8080
    environment:
      DOZZLE_ENABLE_ACTIONS: true
      DOZZLE_AUTO_UPDATE: daily
      DOZZLE_AUTO_UPDATE_TIME: "04:00"
volumes:
  dozzle-data:
```

Du kannst die beiden `DOZZLE_AUTO_UPDATE`-Zeilen weglassen und den Zeitplan stattdessen unter **Einstellungen → Aktualisierungen** festlegen. Dort wählst du auch **Welche Container**. Stoppe danach Watchtower, damit nicht beide dieselben Container aktualisieren.

## Einstellungen

| Watchtower                                                        | Dozzle                                                                                                                                                                                                        |
| ----------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `--schedule` oder `--interval`                                    | `DOZZLE_AUTO_UPDATE` (`daily` oder `weekly` am Sonntag) und `DOZZLE_AUTO_UPDATE_TIME`. Höchstens einmal am Tag                                                                                                |
| Alle Container (Standard)                                         | **Welche Container: Alles**                                                                                                                                                                                   |
| `--label-enable` mit `com.centurylinklabs.watchtower.enable=true` | **Welche Container: Container mit Label** (Standard) mit `dev.dozzle.update: auto`                                                                                                                            |
| `com.centurylinklabs.watchtower.enable=false`                     | `dev.dozzle.update: off`                                                                                                                                                                                      |
| `--cleanup`                                                       | Immer an. Das alte Image ohne Tag wird nach einem Update entfernt, ein vorheriges Image bleibt zum Zurückrollen erhalten                                                                                      |
| `--monitor-only`                                                  | Den Container bei **Container mit Label** ohne Label lassen: Er wird geprüft und als Update angezeigt, aber nie von selbst aktualisiert                                                                       |
| `--rolling-restart`                                               | Immer: Jeder Host aktualisiert einen Container nach dem anderen                                                                                                                                               |
| `--notification-url`                                              | [Alarme und Webhooks](/de/guide/alerts-and-webhooks) für Container-Ereignisse oder [Dozzle Cloud](/de/guide/dozzle-cloud), das jedes Update beobachtet und Bescheid gibt, wenn eine neue Version Fehler zeigt |
| `--run-once`                                                      | **Aktualisieren** in der Update-Leiste                                                                                                                                                                        |
| Zugangsdaten für private Registries                               | Nicht unterstützt. Container aus einer privaten Registry werden übersprungen                                                                                                                                  |

## Labels

Die Labels von Watchtower werden nicht gelesen. Ersetze `com.centurylinklabs.watchtower.enable` durch `dev.dozzle.update`:

| `dev.dozzle.update` | Was passiert                                                                                                    |
| ------------------- | --------------------------------------------------------------------------------------------------------------- |
| `auto`              | Wird nach Zeitplan aktualisiert, außer **Welche Container** steht auf **Nur Dozzle**                            |
| _(kein Label)_      | Wird bei **Alles** nach Zeitplan aktualisiert. Sonst geprüft und als Update angezeigt, das du selbst einspielst |
| `off`               | Wird nie geprüft und nie aktualisiert                                                                           |

Ältere Dozzle-Labels werden weiter akzeptiert: `dev.dozzle.auto-update=true` gilt als `auto` und `dev.dozzle.update-check=false` als `off`.

## Was anders ist

- Der alte Container bleibt erhalten, bis der neue stabil läuft und, falls er einen Healthcheck hat, gesund ist. Sonst kommt der alte zurück.
- Ungesunde Container werden übersprungen.
- Gestoppte Container werden nie aktualisiert, auch nicht mit dem Label `auto`.
- Mit [Dozzle Cloud](/de/guide/dozzle-cloud) lässt sich ein Update aus dem Zeitplan, das Fehler zeigt, [zurückrollen](/de/guide/actions#rolling-back). Der Zeitplan lässt diesen Container dann in Ruhe, bis ein neueres Image erscheint.
- Auf einen Digest gepinnte und lokal gebaute Images werden übersprungen, weil es nichts Neueres zum Vergleichen gibt.
