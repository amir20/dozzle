---
title: Dozzle vs Portainer
description: Wie sich Dozzle und Portainer beim Lesen von Docker-Logs unterscheiden und wie du beide nebeneinander betreibst.
sourceHash: d147e79555a6
---

# Dozzle vs Portainer

Portainer ist dafür gebaut, Container zu verwalten: Stacks, Volumes, Netzwerke, Images und Benutzer. Dozzle ist dafür gebaut, ihre Logs zu lesen. Portainer hat einen Log-Viewer, und Dozzle kann Container starten, stoppen und aktualisieren, aber jedes der beiden ist auf eine andere Aufgabe ausgerichtet. Viele betreiben beide.

## Auf einen Blick

|                   | Dozzle                                                                                                     | Portainer                                                  |
| ----------------- | ---------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------- |
| Hauptaufgabe      | Logs lesen                                                                                                 | Container, Stacks, Volumes, Netzwerke und Images verwalten |
| Live-Logs         | Streamt Zeilen, sobald sie ankommen                                                                        | Automatische Aktualisierung                                |
| Mehrere Container | Ein Stack, ein Service oder eine eigene Gruppe in einer zusammengeführten Ansicht                          | Ein Container nach dem anderen                             |
| JSON-Logs         | Erkannt, formatiert und farbcodiert. [SQL](/de/guide/sql-engine) über JSON-Felder                          | Als Text angezeigt                                         |
| Suche             | Suchen und filtern, auch mit regulären Ausdrücken                                                          | Suchen und filtern nach Text                               |
| CPU und Speicher  | Live-Diagramme mit jüngstem Verlauf                                                                        | Live-Statistiken                                           |
| Alarme            | Bei Log-Zeilen, CPU oder Speicher und Container-Ereignissen. Siehe [Alarme](/de/guide/alerts-and-webhooks) | Nicht im Log-Viewer                                        |
| Image-Updates     | Prüft auf neue Images und aktualisiert nach Zeitplan. Siehe [Updates](/de/guide/updates)                   | Pull und erneutes Deployment von Hand                      |
| Mehrere Hosts     | [Agents](/de/guide/agent), [Swarm](/de/guide/swarm-mode), [Kubernetes](/de/guide/k8s)                      | Environments                                               |
| Einrichtung       | Ein Container, plus ein optionaler Einrichtungsassistent beim ersten Start                                 | Ein Container, plus ein Admin-Konto beim ersten Start      |

## Wann Portainer die bessere Wahl ist

Nimm Portainer, wenn du Stacks im Browser erstellen und bearbeiten, Volumes, Netzwerke und Images verwalten oder einem Team Zugriff auf manche Environments geben willst und auf andere nicht. Nichts davon versucht Dozzle zu leisten.

## Wann Dozzle besser passt

Nimm Dozzle, wenn die Frage lautet: „Was macht dieser Container gerade?“ Es öffnet direkt die Logs, verfolgt mehrere Container gleichzeitig und macht JSON ohne jede Einrichtung lesbar. Außerdem ist es standardmäßig schreibgeschützt: [Aktionen](/de/guide/actions) und [Shell-Zugriff](/de/guide/shell) sind aus, bis du sie einschaltest.

## Beide betreiben

Die beiden kommen sich nicht in die Quere. Beide lesen den Docker-Socket und laufen auf unterschiedlichen Ports:

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - dozzle_data:/data
    ports:
      - 8080:8080
  portainer:
    image: portainer/portainer-ce:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - portainer-data:/data
    ports:
      - 9443:9443
volumes:
  dozzle_data:
  portainer-data:
```

Wie du von Portainer direkt zu den Logs eines Containers in Dozzle springst, steht unter [Link zu einem Container über seinen Namen](/de/guide/faq#wir-haben-tools-die-dozzle-nutzen-sobald-ein-neuer-container-erstellt-wird-wie-bekomme-ich-einen-direkten-link-zu-einem-container-uber-seinen-namen).
