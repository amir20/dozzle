---
title: Deine Daten
sourceHash: f11dd8e45cb5
---

# Deine Daten

Was deinen Host verlässt, wie du das stoppst, und was [Dozzle Cloud](/de/guide/dozzle-cloud) speichert, sobald es angekommen ist.

## Das Verbinden stellt dein Dozzle nicht ins Netz

Dein Dozzle baut eine **ausgehende** Verbindung zu Cloud auf. Es wird nichts Eingehendes geöffnet, kein Port weitergeleitet, und Cloud erreicht deine Instanz nur über die Verbindung, die deine Instanz gestartet hat. Trennst du sie, endet dieser Zugriff sofort.

Das Verbinden fügt deinem selbst gehosteten Dozzle auch keine Authentifizierung hinzu. Das ist eine eigene und wichtige Frage: **Standardmäßig hat Dozzle keinen Login.** Wer es im Netzwerk erreichen kann, kann deine Logs sehen. Wenn du Dozzle ins Internet gestellt hast oder dir das Netzwerk teilst, richte [Authentifizierung](/de/guide/authentication) auf der Instanz selbst ein. Das gilt unabhängig davon, ob du verbindest.

Container-Aktionen — starten, stoppen, neu starten — lehnt deine Instanz weiterhin ab, solange du sie nicht mit `DOZZLE_ENABLE_ACTIONS` aktivierst. Siehe [Aktionen](/de/guide/actions).

## Steuern, was weitergeleitet wird

Die wirksamste Datenschutzmaßnahme ist, etwas gar nicht erst zu senden. Standardmäßig streamt jeder laufende Container seine Logs an Cloud, solange die Instanz verbunden ist. Für Container, deren Info-Geplapper keinen diagnostischen Wert hat, oder die mit Material umgehen, das du lieber auf dem Host behältst, filtere oder schalte per Label ganz ab.

### `dev.dozzle.cloud.min_level`

| Wert                                          | Wirkung                                                                                          |
| --------------------------------------------- | ------------------------------------------------------------------------------------------------ |
| _(nicht gesetzt)_                             | Alle Log-Zeilen werden weitergeleitet. Standard.                                                 |
| `disabled`                                    | Der Container wird vollständig übersprungen. Es werden keine Logs an Cloud weitergeleitet.       |
| `trace`                                       | Wie nicht gesetzt, da trace die niedrigste Stufe ist. Alles wird weitergeleitet.                 |
| `debug` / `info` / `warn` / `error` / `fatal` | Nur Zeilen ab dieser Stufe werden weitergeleitet. Zeilen ohne erkannte Stufe kommen immer durch. |

Ein unbekannter Wert (ein Tippfehler wie `warning` oder `wran`) wird als Fehler geloggt und ignoriert, der Container streamt also alles, als wäre das Label nicht gesetzt.

Das Label wird beim Start des Log-Readers gelesen. Eine Änderung an einem laufenden Container wirkt erst nach dessen Neustart.

```yaml
services:
  zigbee2mqtt:
    image: koenkk/zigbee2mqtt
    labels:
      # Nur warn/error/fatal an Dozzle Cloud weiterleiten
      - dev.dozzle.cloud.min_level=warn

  noisy-debug-tool:
    image: example/debug
    labels:
      # Von diesem Container nichts senden
      - dev.dozzle.cloud.min_level=disabled
```

Der Filter läuft auf deiner Dozzle-Instanz, **bevor Logs den Host verlassen**. Verworfene Zeilen berühren also nie das Netzwerk und zählen nie auf dein Kontingent. Das lokale Betrachten der Logs in Dozzle ist davon nicht betroffen.

## Was Cloud speichert

- **Log-Zeilen**, die von deinen verbundenen Instanzen weitergeleitet wurden, für die Volltextsuche.
- **Ereignisse und Alarme**, die deine Regeln getroffen haben, samt Untersuchungen und Befunden.
- **Container- und Host-Metadaten** — Namen, Images, Zustände, Ressourcenverbrauch.
- **Dein Konto** — E-Mail-Adresse, Tarif, Einstellungen der Benachrichtigungskanäle.
- **Chat-Verlauf** mit dem Agent.

Alles ist auf dein Konto beschränkt; andere Nutzer sehen deine Daten nicht. Gespeicherte Daten bleiben für den Aufbewahrungszeitraum deines Tarifs und werden dann automatisch gelöscht. Siehe [Tarife & Limits](/de/guide/dozzle-cloud/plans).

## API-Schlüssel

Jede verbundene Instanz authentifiziert sich mit einem eigenen API-Schlüssel. Schlüssel werden mit BLAKE2b gehasht, unterstützen Ablaufdaten und werden nie im Klartext gespeichert.

Das Löschen eines Schlüssels auf der Instances-Seite trennt diese Instanz sofort und dauerhaft. Der Schlüssel lässt sich nicht wiederherstellen oder erneut anhängen — verbinde die Instanz neu, um einen neuen zu bekommen. Wenn du glaubst, dass ein Schlüssel offengelegt wurde, lösche ihn und verbinde neu. Das ist die vollständige Abhilfe: der alte Schlüssel funktioniert ab dem Löschen nicht mehr.

## Anmeldung

Cloud nutzt die Anmeldung mit GitHub oder Google. Es gibt kein separates Passwort anzulegen, und Cloud sieht dein GitHub- oder Google-Passwort nie. Registrierst du dich mit dem einen Anbieter und meldest dich später mit dem anderen unter derselben E-Mail-Adresse an, landest du im selben Konto.

## Datensammlung stoppen, ohne das Konto zu schließen

1. Lösche die API-Schlüssel deiner Instanzen auf der Instances-Seite. Die Weiterleitung endet sofort.
2. Deaktiviere deine Kanäle auf der Channels-Seite, damit nichts zugestellt wird.

Die bestehende Historie läuft dann innerhalb des Aufbewahrungszeitraums deines Tarifs von selbst aus.

## Konto schließen

Es gibt noch keinen Selbstbedienungs-Löschknopf. Schreib von der Konto-Adresse aus an **amir@dozzle.dev** und bitte um Löschung. Wenn du in einem bezahlten Tarif bist, kündige ihn vorher in den Einstellungen, damit dir nichts mehr berechnet wird.

Vorher oder stattdessen entfernst du mit den Schritten oben praktisch alles selbst: das Löschen deiner API-Schlüssel stoppt jede Sammlung, und gespeicherte Daten laufen mit der Aufbewahrung aus.
