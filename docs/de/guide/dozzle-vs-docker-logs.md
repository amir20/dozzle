---
title: Dozzle vs docker logs
description: Was Dozzle gegenüber docker logs und Terminal-Tools bietet und wann das Terminal alles ist, was du brauchst.
sourceHash: d312eefe8524
---

# Dozzle vs `docker logs`

Dozzle liest dieselben Logs wie `docker logs`. Nichts wird kopiert oder gespeichert, es zeigt also nie etwas, das der Befehl nicht auch zeigen könnte. Was sich ändert, ist, wie du sie liest: im Browser, über Container und Hosts hinweg, ohne Shell auf dem Server.

## Auf einen Blick

|                   | Dozzle                                                                                    | `docker logs`                                             |
| ----------------- | ----------------------------------------------------------------------------------------- | --------------------------------------------------------- |
| Wo du sie liest   | Im Browser, auf jedem Gerät                                                               | In einer Shell auf dem Docker-Host                        |
| Mehrere Container | Beliebige Container in einer zusammengeführten Ansicht oder ein ganzer Stack oder Service | Ein Container. `docker compose logs` deckt ein Projekt ab |
| Mehrere Hosts     | Eine Oberfläche für jeden Host, über [Agents](/de/guide/agent)                            | Ein Host pro Shell oder ein `DOCKER_HOST` pro Befehl      |
| Suchen            | Suche während der Eingabe, mit regulären Ausdrücken                                       | Pipe an `grep`                                            |
| JSON-Logs         | Formatiert und farbcodiert, [SQL](/de/guide/sql-engine) über Felder                       | Rohe Zeilen, Pipe an `jq`                                 |
| Stacktraces       | Mehrzeilige Einträge zusammengefasst                                                      | Eine Zeile nach der anderen                               |
| CPU und Speicher  | Diagramme neben den Logs                                                                  | `docker stats` in einer anderen Shell                     |
| Teilen            | Ein Link zu einem Container, hinter [Authentifizierung](/de/guide/authentication)         | Kopieren und Einfügen                                     |
| Installation      | Ein Container                                                                             | Schon vorhanden                                           |

## Wann das Terminal reicht

Wenn du schon auf dem Host bist und die letzten hundert Zeilen eines Containers sehen willst, ist `docker logs --tail 100 -f <name>` der schnellste Weg. Für eine Live-Ansicht aller Container im Terminal zeigt [dtop](/de/guide/dtop) sie in einer Tabelle im Stil von `docker ps` und öffnet Dozzle, wenn du die vollständigen Logs brauchst.

## Wann Dozzle besser passt

Nutze Dozzle, wenn mehr als eine Person die Logs liest, wenn du keinen SSH-Zugang nur zum Lesen von Logs vergeben willst oder wenn du eine Anfrage durch mehrere Container gleichzeitig verfolgst. Auch auf dem Handy lassen sich Logs damit leichter lesen.

## Ausprobieren

```sh
docker run -d -v /var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle:latest
```

Öffne dann `http://localhost:8080`. Weitere Optionen findest du unter [Erste Schritte](/de/guide/getting-started).
