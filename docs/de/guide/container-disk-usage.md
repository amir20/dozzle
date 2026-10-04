---
title: Speicherbelegung der Container
sourceHash: db32da6a38bf
---

# Speicherbelegung der Container

Die Spalte **Datenträger** in der Containerliste zeigt, wie viel jeder Container in sein eigenes Dateisystem geschrieben hat, für laufende und gestoppte Container gleichermaßen. Es ist derselbe Wert wie der erste in der Spalte `SIZE` von `docker ps -s`, nur in binären Einheiten, sodass Dockers `20.5kB` hier als `20 KB` erscheint.

## Was gezählt wird

Docker nennt das die beschreibbare Schicht des Containers: jede Datei, die ein Container außerhalb seiner Mounts anlegt oder ändert. Ein Container, der temporäre Dateien, Caches oder Logs in sein eigenes Dateisystem schreibt, wächst hier, und ebenso einer, der nach dem Start `apt install` ausführt.

Nicht gezählt werden:

- **Volumes**, benannte wie anonyme
- **Bind Mounts**, etwa `./data:/var/lib/postgresql/data`
- **Das Image**, auf dem der Container läuft und das sich alle Container dieses Images teilen
- **Dockers eigene Logdatei** für den Container

Die meisten Datenbanken und Log-Speicher legen ihre Daten in einem Volume oder Bind Mount ab, deshalb kann ein Postgres mit 30 GB hier nur ein paar KB zeigen. Das stimmt so: Die 30 GB liegen im Mount, nicht im Container.

## Den Rest finden

Docker kann Volumes messen, weiß aber nichts über den Inhalt eines Host-Ordners, den du per Bind Mount einbindest. Um einen zu messen, muss jede Datei darin durchlaufen werden, und das macht Dozzle nicht. Um zu sehen, wo der restliche Platz hingeht, führe auf dem Host Folgendes aus:

```sh
# Volumes, mit der Anzahl der Container, die sie nutzen
docker system df -v

# ein per Bind Mount eingebundener Ordner
du -sh /data/postgres
```

Um den Datenträger im Blick zu behalten, auf dem diese Ordner liegen, binde ihn auf der Host-Karte ein, wie unter [Host-Metriken](/de/guide/host-metrics#weitere-laufwerke) beschrieben.

## Wann der Wert aktualisiert wird

Docker speichert diesen Wert nirgends. Es ermittelt ihn bei jeder Anfrage, indem es die Schicht durchläuft, deshalb fragt Dozzle selten:

- einmal für jeden Container, kurz nachdem Dozzle gestartet ist
- noch einmal, wenn ein Container stoppt, weil sich seine Schicht danach nicht mehr ändern kann
- bei einem laufenden Container, nachdem er etwa 100 MB geschrieben hat, oder alle 5 Minuten, wenn er überhaupt etwas geschrieben hat

Die Prüfungen laufender Container finden nur statt, solange jemand Dozzle geöffnet hat. Ein Container zeigt `–`, bis er zum ersten Mal gemessen wurde.

## Einschränkungen

- Kubernetes hat keine beschreibbare Schicht, die sich melden ließe, deshalb ist die Spalte im [k8s-Modus](/de/guide/k8s) ausgeblendet.
- Ein [Agent](/de/guide/agent) misst seine eigenen Container. Ein Agent, der älter ist als das Dozzle, an das er meldet, sendet keine Größe, und seine Container zeigen `–`.
