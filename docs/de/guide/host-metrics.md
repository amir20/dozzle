---
title: Host-Metriken
sourceHash: 5d6314385b49
---

# Host-Metriken

Die Host-Karte kann drei Werte für die Maschine anzeigen, auf der Docker läuft. Sie stehen in einem kleinen Kasten rechts im Kopf der Karte, gekennzeichnet mit einem Puls-Symbol, damit man sie nicht mit den Container-Summen in den Anzeigen für CPU und Arbeitsspeicher darunter verwechselt. Auf dem Handy wird der Speicherplatz zu einer dritten Anzeige neben CPU und Arbeitsspeicher, und Laufzeit und Last rücken in eine Zeile unter den Anzeigen:

- **Laufzeit**, wie lange der Host schon läuft
- **Last**, der Load Average über 1 Minute, der gelb wird, sobald er die Anzahl der Kerne übersteigt, und rot ab dem Doppelten davon (beim Darüberfahren mit der Maus erscheinen die Werte für 5 und 15 Minuten und die Anzahl der Kerne)
- **Datenträger**, wie voll das Dateisystem mit dem Datenverzeichnis von Docker ist, als kleiner Balken, der ab 70% gelb und ab 90% rot wird (beim Darüberfahren mit der Maus erscheinen belegter und gesamter Speicher)

Solange ein Tab offen ist, werden sie alle 15 Sekunden aktualisiert. Jeder Wert erscheint nur, wenn Dozzle dafür einen echten Wert lesen kann. Bei einer Standardinstallation fehlt der Kasten also womöglich ganz oder zeigt nur einen Teil davon.

## Dozzle in einem Container betreiben

In einem Container beschreibt `/proc` den Container und nicht den Host. Dozzle gibt die Zahlen des Containers nicht als die des Hosts aus, deshalb bleiben Last und Laufzeit ausgeblendet, bis du das `/proc` des Hosts unter `/host/proc` einbindest.

::: code-group

```sh
docker run -d \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /proc:/host/proc:ro \
  -v dozzle_data:/data \
  -p 8080:8080 amir20/dozzle:latest
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /proc:/host/proc:ro
      - dozzle_data:/data
    ports:
      - 8080:8080
volumes:
  dozzle_data:
```

:::

Für den Datenträgerwert brauchst du keinen zusätzlichen Mount. Dozzle misst das Dateisystem hinter seinem eigenen `/data`, und das liegt auf dem Datenträger von Docker, egal ob `/data` wie oben ein benanntes Volume ist oder gar nicht eingebunden wird. Bindest du dort stattdessen einen Ordner des Hosts ein (`./data:/data`), beschreibt der Wert den Datenträger, auf dem dieser Ordner liegt. Meist ist das derselbe.

## Dozzle nativ betreiben

Ein Dozzle-Binary, das direkt auf dem Host läuft, liest `/proc` so wie es ist. Last und Laufzeit brauchen also keine Einrichtung. Die Datenträgerbelegung wird aus dem Datenverzeichnis von Docker gelesen (`docker info | grep "Docker Root Dir"`, meist `/var/lib/docker`). Das klappt, solange der Benutzer, unter dem Dozzle läuft, dieses Verzeichnis sehen kann.

## Agents

Jeder [Agent](/de/guide/agent) liest seine eigene Maschine aus und schickt die Werte an das Dozzle, das du gerade ansiehst. So zeigt die Karte jedes Agents ihre eigene Laufzeit, Last und Datenträgerbelegung. Gib dem Agent-Container dieselben Mounts, die du auch Dozzle geben würdest:

```yaml [docker-compose.yml]
services:
  dozzle-agent:
    image: amir20/dozzle:latest
    command: agent
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /proc:/host/proc:ro
    ports:
      - 7007:7007
```

Weitere Laufwerke funktionieren genauso, eingebunden unter `/host/disks` auf dem Agent. Im [Swarm-Modus](/de/guide/swarm-mode) betreibt jeder Node Dozzle als seinen eigenen Agent. Füge die Mounts also dem Service hinzu, dann meldet sich jeder Node selbst.

Ein Agent, der älter ist als das Dozzle, an das er meldet, schickt keine Metriken, und seine Karte bleibt wie bisher. Aktualisiere den Agent, um sie zu sehen.

## Weitere Laufwerke

Der Datenträgerwert deckt von Haus aus den Datenträger von Docker ab. Um auch andere Laufwerke zu beobachten, bindest du jedes davon unter `/host/disks/<name>` ein. Der Ordnername wird zur Bezeichnung des Laufwerks.

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /proc:/host/proc:ro
      - /mnt/media:/host/disks/media:ro
      - /mnt/backup:/host/disks/backup:ro
```

Der Balken zeigt dann das vollste Laufwerk, denn das ist dasjenige, dem zuerst der Platz ausgeht. Beim Darüberfahren mit der Maus erscheint jedes Laufwerk mit belegtem und gesamtem Speicher.

Dozzle braucht zum Messen eines Laufwerks nur den Einhängepunkt, nicht seine Dateien. Bindest du das Wurzelverzeichnis eines Laufwerks ein, kann Dozzle trotzdem lesen, was darauf liegt. Wenn dir das wichtig ist, leg auf dem Laufwerk einen leeren Ordner an und binde stattdessen diesen ein (`/mnt/media/.dozzle:/host/disks/media:ro`). Er liegt auf demselben Dateisystem und meldet dieselben Zahlen.

Eine native Installation erreicht dasselbe mit symbolischen Links: `ln -s /mnt/media /host/disks/media`.

## Einschränkungen

- Ein [entfernter Host](/de/guide/remote-hosts), der über TCP verbunden ist, hat auf seiner Seite nichts, das die Maschine auslesen könnte. Er zeigt daher CPU und Arbeitsspeicher wie bisher, aber ohne den Kasten. Betreibe dort stattdessen einen Agent, um die Werte zu bekommen.
- Wenn `DOCKER_HOST` auf eine andere Maschine zeigt (`tcp://` oder `ssh://`), lässt Dozzle die Werte weg, denn das eigene `/proc` und die eigenen Datenträger sagen nichts über diese Engine aus.
- Docker Desktop betreibt die Engine in einer VM. Ein natives Dozzle-Binary unter macOS oder Windows hat kein `/proc` zum Lesen, und Dozzle in einem Container meldet dort die Werte der VM, nicht die deines Rechners.
