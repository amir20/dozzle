---
title: Dozzle Cloud
sourceHash: 34c0056128a5
---

# Dozzle Cloud

[Dozzle Cloud](https://cloud.dozzle.dev) ist eine optionale, verwaltete Ergänzung zum selbst gehosteten Dozzle. Es verbindet deine Instanzen miteinander, fasst Container-Ereignisse zusammen, verteilt Alarme über mehrere Kanäle und lässt dich aus dem Chat heraus Fragen zu deiner Infrastruktur stellen. Dozzle selbst bleibt vollständig Open Source und selbst gehostet; Cloud setzt obendrauf.

Das Ziel: Dozzle Cloud soll sich wie der persönliche SRE-Assistent anfühlen, von dem du nicht wusstest, dass du ihn willst. Es beobachtet deine Container, sagt dir Bescheid, wenn etwas wichtig ist, und hält sich raus, wenn nichts los ist.

## Funktionen

### <Icon icon="mdi:text-box-outline" inline /> Log-Zusammenfassungen

Container-Ereignisse werden gebündelt und von einem LLM zusammengefasst. Jede Zusammenfassung hält den Schweregrad und den Quell-Container fest und verlinkt zurück auf die vollständige Logzeile in deiner Dozzle-Instanz.

### <Icon icon="mdi:group" inline /> Muster-Clustering

Wiederkehrende Fehler werden gruppiert und gezählt statt einzeln zugestellt. Eine Schleife, die dieselbe Exception 200-mal ausgibt, erzeugt eine Benachrichtigung mit Häufigkeitsangabe, nicht 200.

### <Icon icon="mdi:robot-outline" inline /> KI-Agent

Ein Chat-Agent beantwortet Fragen zum Zustand deiner Container und zur jüngsten Log-Aktivität. Er ist in Telegram und Discord verfügbar.

In den Plänen Pro und Team kann der Agent auch direkt aus der Unterhaltung heraus auf Container einwirken (starten, stoppen, neu starten), ohne dass Shell-Zugriff auf den Host nötig ist.

### <Icon icon="mdi:calendar-clock" inline /> Tägliche Zusammenfassungen

Eine geplante Übersicht der jüngsten Aktivität über deine verbundenen Instanzen hinweg: die häufigsten Fehlermuster, Ereigniszahlen und der Gesamtzustand. Zustellung per E-Mail zu einer Uhrzeit und Zeitzone deiner Wahl.

### <Icon icon="mdi:bell-ring-outline" inline /> Benachrichtigungskanäle

Alarme lassen sich parallel an mehrere Kanäle senden. Jeder Kanal kann unabhängig aktiviert oder deaktiviert und auf bestimmte Dozzle-Instanzen eingeschränkt werden.

| Kanal                                                      | Alarme | Tägliche Zusammenfassung | Agent in beide Richtungen |
| ---------------------------------------------------------- | :----: | :----------------------: | :-----------------------: |
| <Icon icon="mdi:telegram" inline /> Telegram               |   ✓    |            ✓             |             ✓             |
| <Icon icon="ic:baseline-discord" inline /> Discord         |   ✓    |            ✓             |             ✓             |
| <Icon icon="mdi:email-outline" inline /> E-Mail            |   ✓    |            ✓             |                           |
| <Icon icon="mdi:slack" inline /> Slack                     |   ✓    |                          |                           |
| <Icon icon="simple-icons:ntfy" inline /> ntfy              |   ✓    |                          |                           |
| <Icon icon="mdi:webhook" inline /> Webhooks                |   ✓    |                          |                           |
| <Icon icon="mdi:bell-badge-outline" inline /> Browser-Push |   ✓    |                          |                           |

### <Icon icon="mdi:bell-sleep-outline" inline /> Benachrichtigungen stummschalten

Benachrichtigungen lassen sich für eine Stunde, acht Stunden, bis zum nächsten Morgen oder bis zur nächsten Woche stummschalten. Praktisch bei Störungen oder geplanter Wartung.

### <Icon icon="mdi:view-dashboard-outline" inline /> Dashboard für mehrere Instanzen

Verbundene Dozzle-Instanzen erscheinen in einem gemeinsamen Dashboard. Jede Instanz authentifiziert sich mit einem API-Schlüssel, ein zusätzlicher Agent auf dem Host ist nicht nötig. Das Dashboard zeigt den Online-Status, den Container-Bestand und Logs im Live-Stream.

### <Icon icon="mdi:database-search-outline" inline /> Volltextsuche in Logs

Jede Logzeile, die deine verbundenen Instanzen weiterleiten, landet in einem Volltextindex. Du kannst über alle Instanzen auf einmal suchen oder nach Container, Schweregrad oder Zeitraum filtern. Suchen liefern selbst über Wochen an Verlauf in Millisekunden Ergebnisse, und jeder Treffer verlinkt zurück auf den Kontext in der Quellinstanz. Die Aufbewahrung hängt vom Plan ab und reicht von 24 Stunden bis 30 Tagen.

### <Icon icon="mdi:shield-lock-outline" inline /> Sicherheit

- API-Schlüssel werden mit BLAKE2b gehasht und können ablaufen.
- Die Anmeldung läuft über GitHub- oder Google-OAuth.
- Logs und Ereignisinhalte werden nur so lange gespeichert, wie es das Aufbewahrungsfenster deines Plans vorsieht.

## Eine Instanz verbinden

So verbindest du ein selbst gehostetes Dozzle mit Dozzle Cloud:

1. Öffne deine Dozzle-Instanz und klicke auf das **Cloud**-Symbol in der oberen Leiste.
2. Klicke auf **Instanz verbinden**. Du wirst weitergeleitet, um dich anzumelden und die Verbindung zu bestätigen.
3. Sobald die Verbindung steht, konfigurierst du in Dozzle die Alarm-Abonnements, um festzulegen, welche Ereignisse weitergeleitet werden.

## Steuern, was weitergeleitet wird

Standardmäßig streamt jeder laufende Container seine Logs an Dozzle Cloud, solange die Verbindung besteht. Bei geschwätzigen Containern, deren Info-Meldungen diagnostisch nichts hergeben, kannst du pro Container mit einem einzigen Label filtern oder ganz abschalten.

### `dev.dozzle.cloud.min_level`

| Wert                                          | Wirkung                                                                                           |
| --------------------------------------------- | ------------------------------------------------------------------------------------------------- |
| _(nicht gesetzt)_                             | Alle Logzeilen werden weitergeleitet. Standard.                                                   |
| `disabled`                                    | Der Container wird komplett übersprungen. Es werden keine Logs an Cloud weitergeleitet.           |
| `trace`                                       | Wie nicht gesetzt, da trace das niedrigste Level ist. Alles wird weitergeleitet.                  |
| `debug` / `info` / `warn` / `error` / `fatal` | Nur Zeilen ab diesem Level werden weitergeleitet. Zeilen ohne erkanntes Level kommen immer durch. |

Ein unbekannter Wert (ein Tippfehler wie `warning` oder `wran`) wird als Fehler protokolliert und ignoriert, der Container streamt dann alles, als wäre das Label nicht gesetzt.

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

Der Filter läuft auf deiner Dozzle-Instanz, bevor die Logs den Host verlassen, verworfene Zeilen berühren also nie das Netzwerk und zählen nicht gegen deinen Plan. Die lokale Log-Ansicht in Dozzle bleibt davon unberührt.

## Preise

Die kostenlose Stufe ist bewusst großzügig; du solltest Dozzle Cloud in einem Homelab oder kleinen Team wirklich nutzen können, ohne an eine Grenze zu stoßen. Kostenpflichtige Pläne gibt es für höhere Ereignisvolumen, längere Aufbewahrung und die Container-Aktionen des Agents. Aktuelle Limits und Plandetails findest du unter [cloud.dozzle.dev](https://cloud.dozzle.dev).

## Feedback

Dozzle Cloud stammt von derselben Person, die Dozzle gebaut hat, und der Anspruch ist derselbe: Dinge, die Leute wirklich nutzen wollen. Wenn du es ausprobierst und etwas schief wirkt, fehlt oder richtig nützlich ist, [eröffne bitte eine Diskussion](https://github.com/amir20/dozzle/discussions). Dieses Feedback bestimmt, was als Nächstes gebaut wird.
