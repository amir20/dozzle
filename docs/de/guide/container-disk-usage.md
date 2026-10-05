---
title: Speicherbelegung der Container
sourceHash: 36536624d20d
---

# Speicherbelegung der Container

Die Spalte **Datenträger** in der Containerliste zeigt, wie viel Speicherplatz jeder Container belegt, für laufende und gestoppte Container gleichermaßen. Sie addiert zwei Dinge, die Docker messen kann: was der Container in sein eigenes Dateisystem geschrieben hat, und die Docker-Volumes, die er einbindet. Fahre mit der Maus über den Wert, um jeden Teil zu sehen.

Die Größen sind in binären Einheiten, sodass Dockers `20.5kB` hier als `20 KB` erscheint.

## Was gezählt wird

- **Die beschreibbare Schicht**: jede Datei, die ein Container außerhalb seiner Mounts anlegt oder ändert, etwa temporäre Dateien, Caches oder Pakete, die nach dem Start installiert wurden. Das ist der erste Wert in der Spalte `SIZE` von `docker ps -s`.
- **Volumes**, benannte wie anonyme, die der Container einbindet. Das sind die Größen, die `docker system df -v` auflistet. Ein Volume, das mehrere Container nutzen, zählt bei jedem von ihnen, und der Tooltip weist darauf hin, dass es geteilt ist.

Nicht gezählt werden:

- **Bind Mounts**, etwa `./data:/var/lib/postgresql/data`
- **Das Image**, auf dem der Container läuft und das sich alle Container dieses Images teilen
- **Dockers eigene Logdatei** für den Container
- **Volumes, die kein Container nutzt**, da es für sie keine Zeile gibt, in der sie erscheinen könnten. Sie zählen stattdessen zum [freigebbaren Speicher](#freigebbarer-speicher) des Hosts.

## Bind Mounts

Docker weiß nichts über den Inhalt eines Host-Ordners, den du per Bind Mount einbindest, und um einen zu messen, muss jede Datei darin durchlaufen werden, was Dozzle nicht macht. Eine Datenbank, die ihre Daten in einem Bind Mount ablegt, zeigt hier ein paar KB, auch wenn sie viele GB enthält. Um einen zu messen, führe auf dem Host Folgendes aus:

```sh
du -sh /data/postgres
```

Um den Datenträger im Blick zu behalten, auf dem diese Ordner liegen, binde ihn auf der Host-Karte ein, wie unter [Host-Metriken](/de/guide/host-metrics#weitere-laufwerke) beschrieben.

## Freigebbarer Speicher

Die Host-Karte zeigt **Freigebbar** neben ihrer Datenträger-Anzeige: Speicher, den Dinge belegen, die kein Container nutzt, dieselbe Summe wie in der Spalte `RECLAIMABLE` von `docker system df`. Fahre mit der Maus darüber, um zu sehen, wie viel davon auf ungenutzte Images, ungenutzte Volumes, gestoppte Container und den Build-Cache entfällt. Der Wert wird zusammen mit den Volumes aktualisiert, also höchstens alle 20 Minuten.

## Wann der Wert aktualisiert wird

Docker speichert diese Werte nirgends. Es ermittelt sie bei jeder Anfrage, indem es die Dateien durchläuft, deshalb fragt Dozzle selten.

Die beschreibbare Schicht wird gemessen:

- einmal für jeden Container, kurz nachdem Dozzle gestartet ist
- noch einmal, wenn ein Container stoppt, weil sich seine Schicht danach nicht mehr ändern kann
- bei einem laufenden Container, nachdem er etwa 100 MB geschrieben hat, oder alle 5 Minuten, wenn er überhaupt etwas geschrieben hat

Volumes werden direkt nach diesem ersten Durchgang gemessen, danach höchstens alle 20 Minuten. Docker kann nur alle Volumes auf einmal messen, deshalb durchläuft jede Aktualisierung jedes Volume auf dem Host. Ein Container, der dazwischen erstellt wird, zeigt seine Volumes bei der nächsten Aktualisierung.

Alles nach dem ersten Durchgang passiert nur, solange jemand Dozzle geöffnet hat. Ein Container zeigt `–`, bis er zum ersten Mal gemessen wurde.

## Einschränkungen

- Kubernetes hat weder eine beschreibbare Schicht noch Docker-Volumes, die sich melden ließen, deshalb ist die Spalte im [k8s-Modus](/de/guide/k8s) ausgeblendet.
- Volumes eines Plugin-Treibers können meist keine Größe melden und werden weggelassen.
- Ein [Agent](/de/guide/agent) misst seine eigenen Container. Ein Agent, der älter ist als das Dozzle, an das er meldet, sendet keine Größen, und seine Container zeigen `–`.
