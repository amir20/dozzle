---
title: Umstieg von Watchtower
sourceHash: acec3dd8eb61
---

# Umstieg von Watchtower

Dozzle kann, was Watchtower kann: deine Container nach Zeitplan auf neuere Images prüfen und sie aktualisieren. Jedes Update wird überwacht und zurückgesetzt, wenn der neue Container nicht stabil läuft. Diese Seite ordnet die Einstellungen von Watchtower denen von Dozzle zu.

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

Du kannst die beiden `DOZZLE_AUTO_UPDATE`-Zeilen auch weglassen und den Zeitplan unter **Einstellungen → Updates** setzen. Stoppe danach Watchtower, damit nicht beide dieselben Container aktualisieren.

## Einstellungen

| Watchtower                                                        | Dozzle                                                                                                                                                                                                            |
| ----------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `--schedule` oder `--interval`                                    | `DOZZLE_AUTO_UPDATE` (`daily`, oder `weekly` am Sonntag) und `DOZZLE_AUTO_UPDATE_TIME`. Höchstens einmal am Tag                                                                                                   |
| Jeder Container (Standard)                                        | **Welche Container: Alles**                                                                                                                                                                                       |
| `--label-enable` mit `com.centurylinklabs.watchtower.enable=true` | **Welche Container: Dozzle und ausgewählte Container** (Standard) mit `dev.dozzle.update: auto`                                                                                                                   |
| `com.centurylinklabs.watchtower.enable=false`                     | `dev.dozzle.update: off`, oder **Manuell**, um die Update-Prüfung zu behalten                                                                                                                                     |
| `--cleanup`                                                       | Immer an. Das alte Image ohne Tag wird nach einem Update entfernt, ein vorheriges Image bleibt zum Zurücksetzen erhalten                                                                                          |
| `--monitor-only`                                                  | **Manuell**: Der Container wird geprüft und als Update angezeigt, aber nie von selbst aktualisiert                                                                                                                |
| `--rolling-restart`                                               | Immer: Jeder Host aktualisiert einen Container nach dem anderen                                                                                                                                                   |
| `--notification-url`                                              | [Alarme und Webhooks](/de/guide/alerts-and-webhooks) für Container-Ereignisse, oder [Dozzle Cloud](/de/guide/dozzle-cloud), das jedes Update beobachtet und dir sagt, wenn eine neue Version anfängt zu scheitern |
| `--run-once`                                                      | **Aktualisieren** in der Updates-Leiste                                                                                                                                                                           |
| Zugangsdaten für private Registries                               | Nicht unterstützt. Container aus einer privaten Registry werden übersprungen                                                                                                                                      |

## Labels

Die Labels von Watchtower werden nicht gelesen. Ersetze `com.centurylinklabs.watchtower.enable` durch `dev.dozzle.update`:

| `dev.dozzle.update` | Was passiert                                                     |
| ------------------- | ---------------------------------------------------------------- |
| `auto`              | Wird nach Zeitplan aktualisiert                                  |
| _(kein Label)_      | Manuell: wird geprüft und als Update angezeigt, das du anwendest |
| `off`               | Wird nie geprüft und nie aktualisiert                            |

Ein Label hat Vorrang vor einer Wahl in der UI. Ältere Dozzle-Labels werden weiterhin akzeptiert: `dev.dozzle.auto-update=true` gilt als `auto` und `dev.dozzle.update-check=false` als `off`.

## Was anders ist

- Der alte Container bleibt erhalten, bis der neue stabil läuft, und healthy ist, falls er einen Healthcheck hat. Sonst wird der alte zurückgesetzt.
- Container, die unhealthy sind, werden übersprungen.
- Ein Container, den jemand [zurückgesetzt](/de/guide/actions#rolling-back) hat, wird erst wieder aktualisiert, wenn ein neueres Image erscheint.
- Images mit festem Digest und lokal gebaute Images werden übersprungen, weil es nichts Neueres zum Vergleichen gibt.
