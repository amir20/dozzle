---
title: Dozzle vs Grafana Loki
description: Ob du Grafana und Loki brauchst, um Docker-Logs zu lesen, oder ob Dozzle reicht, und wie du beide zusammen nutzt.
sourceHash: 95a7cee5f63a
---

# Dozzle vs Grafana Loki

Loki ist ein Log-Speicher. Ein Collector schickt jede Log-Zeile hinein, Loki behält sie so lange, wie du es konfigurierst, und in Grafana fragst du sie ab. Dozzle speichert nichts: Es liest die Logs direkt aus Docker, dieselben, die `docker logs` anzeigt, und streamt sie in deinen Browser.

Die eigentliche Frage ist also, ob du Logs anderswo als in Docker aufbewahren musst.

## Auf einen Blick

|                               | Dozzle                                                                    | Grafana + Loki                                      |
| ----------------------------- | ------------------------------------------------------------------------- | --------------------------------------------------- |
| Was du betreibst              | Einen Container                                                           | Loki, Grafana und einen Collector wie Grafana Alloy |
| Wo die Logs liegen            | In Docker, so wie Docker sie ohnehin aufbewahrt                           | Kopiert in Lokis eigenen Speicher                   |
| Wie weit zurück               | So weit, wie Dockers Log-Rotation sie behält                              | So lange, wie du Lokis Aufbewahrung einstellst      |
| Live-Tail                     | Die Standardansicht                                                       | Verfügbar in Grafana Explore                        |
| Abfragen                      | Suche, reguläre Ausdrücke, [SQL](/de/guide/sql-engine) auf JSON           | LogQL, einschließlich aus Logs berechneter Metriken |
| Dashboards                    | CPU und Speicher der Container                                            | Jedes Dashboard, das du in Grafana baust            |
| Alarme                        | [Alarme](/de/guide/alerts-and-webhooks) auf Logs, Metriken und Ereignisse | Grafana-Alerting auf LogQL-Abfragen                 |
| Logs von außerhalb von Docker | Nein                                                                      | Ja: Dateien, systemd-Journal, Kubernetes und mehr   |

## Wann Loki die bessere Wahl ist

Nimm Loki, wenn du Logs brauchst, nachdem Docker sie wegrotiert hat, wenn ein Container entfernt wird und seine Logs erhalten bleiben müssen oder wenn du über Wochen an Verlauf suchen musst. Es passt auch besser, wenn du schon Grafana betreibst und Logs neben deinen Metriken haben willst oder Logs von Dingen brauchst, die keine Container sind.

Plane die Einrichtung ein. Promtail, der Collector, den die meisten älteren Anleitungen verwenden, hat im März 2026 das Ende seines Lebenszyklus erreicht, deshalb nutzen neue Installationen Grafana Alloy oder den Loki-Docker-Treiber.

## Wann Dozzle besser passt

Nimm Dozzle, wenn du sehen willst, was ein Container gerade tut oder in den letzten Stunden oder Tagen getan hat, ohne eine Log-Pipeline zu betreiben. Es gibt keinen Speicher zu dimensionieren, keine Aufbewahrung einzustellen und keine Abfragesprache zu lernen. Es startet in Sekunden. Siehe [Erste Schritte](/de/guide/getting-started).

Wie viel Verlauf Dozzle anzeigen kann, hängt von Dockers Log-Rotation ab. Um mehr zu behalten, erhöhe die Grenzen in `/etc/docker/daemon.json`:

```json
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "50m",
    "max-file": "5"
  }
}
```

Die neuen Grenzen gelten für Container, die nach dem Neustart von Docker erstellt werden.

## Beide nutzen

Viele Setups betreiben Loki für den Verlauf und Dozzle fürs alltägliche Debugging. Die beiden kommen sich nicht in die Quere: Dozzle liest aus Docker, und Alloy liest aus Docker. Stellst du Container stattdessen auf den Loki-Docker-Treiber um, lass Dockers [Dual Logging](https://docs.docker.com/engine/logging/dual-logging/) eingeschaltet (der Standard), damit `docker logs` und damit auch Dozzle sie weiterhin lesen können. Docker hält diese Kopie dann in einem Cache, dessen Größe `cache-max-size` und `cache-max-file` bestimmen (standardmäßig 5 Dateien zu 20 MB), nicht das `max-size` von oben.
