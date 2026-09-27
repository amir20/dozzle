---
title: Container-Aktionen
sourceHash: f51478b92485
---

# Container-Aktionen

<Badge type="warning" text="Docker Only" />

Dozzle unterstützt Container-Aktionen: Über das Dropdown-Menü rechts neben den Container-Statistiken kannst du Container `start`en, `stop`pen, neu starten (`restart`), entfernen (`remove`) und aktualisieren (`update`). Diese Funktion ist standardmäßig **deaktiviert** und lässt sich aktivieren, indem du die Umgebungsvariable `DOZZLE_ENABLE_ACTIONS` auf `true` setzt.

Die Aktion `update` lädt das neueste Image für den Container und erstellt ihn mit derselben Konfiguration neu — praktisch, um einen Container an Ort und Stelle zu aktualisieren, ohne seine Compose-Datei zu bearbeiten. `update` hat nur dann einen spürbaren Effekt, wenn das Image ein bewegliches Tag nutzt (z. B. `latest`, `stable`); bei einem fest gepinnten Tag wird schlicht dasselbe Image erneut geladen.

> [!WARNING]
> `remove` und `update` erstellen den Container neu. Daten in **anonymen Volumes** oder in der beschreibbaren Schicht des Containers gehen dabei verloren. Benannte Volumes und Bind-Mounts bleiben erhalten.

::: code-group

```sh
docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --enable-actions
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    ports:
      - 8080:8080
    environment:
      DOZZLE_ENABLE_ACTIONS: true
```

:::

## Update-Prüfung

Dozzle prüft, ob das Image, das ein Container ausführt, noch dasselbe ist, das seine Registry ausliefert. Weichen sie voneinander ab, erscheint ein Punkt am Container-Menü und das Menü weist auf ein verfügbares Update hin.

Die Prüfung fragt bei der Registry den Digest des Tags ab, aus dem der Container erstellt wurde, und vergleicht ihn mit dem Digest, den der Container tatsächlich ausführt. Das passiert über eine `HEAD`-Anfrage auf das Image-Manifest, es werden also keine Layer heruntergeladen und es zählt nicht gegen die Pull-Limits von Docker Hub. Antworten werden sechs Stunden zwischengespeichert, und dasselbe Image wird immer nur einmal abgefragt, egal wie viele Container oder Hosts es ausführen.

Da gegen das verglichen wird, was der Container _ausführt_, gilt ein Container so lange als veraltet, bis er neu erstellt wird, selbst wenn ein neueres Image bereits auf den Host geladen wurde.

Die Prüfung ist unabhängig von den Aktionen. Zu wissen, dass ein Container veraltet ist, ist nützlich, egal ob Dozzle etwas dagegen unternehmen darf oder nicht, der Hinweis erscheint also auch, wenn `DOZZLE_ENABLE_ACTIONS` aus ist. Nur die Schaltfläche `Update` setzt Aktionen voraus.

### Abschalten

`DOZZLE_IMAGE_CHECK_MODE` steuert, ob Dozzle überhaupt Registries kontaktiert.

| Wert        | Verhalten                                                                                      |
| ----------- | ---------------------------------------------------------------------------------------------- |
| `automatic` | Prüft im Hintergrund, sobald ein Container angesehen wird.                                     |
| `manual`    | Prüft nie von selbst. Das Menü bietet die Aktion "Nach Updates suchen" an.                     |
| `off`       | Die Funktion ist weg. Es wird kein Endpunkt registriert und es werden keine Anfragen gestellt. |

Standardmäßig übernimmt sie den Wert von `DOZZLE_RELEASE_CHECK_MODE`. Wenn du Dozzle also schon gesagt hast, dass es Releases nicht automatisch abrufen soll, prüft es auch Images nicht automatisch.

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      DOZZLE_IMAGE_CHECK_MODE: off
```

Um einen einzelnen Container ruhigzustellen, etwa einen bewusst auf eine Version gepinnten, versiehst du ihn mit einem Label:

```yaml [docker-compose.yml]
services:
  database:
    image: postgres:18-alpine
    labels:
      dev.dozzle.update-check: false
```

Bei einem gefundenen Update kann auch eine Benachrichtigung erscheinen. Sie ist standardmäßig aus und findet sich unter den Einstellungen.

### Was sich nicht prüfen lässt

Bei manchen Containern gibt es nichts zu vergleichen, und Dozzle bleibt still statt zu raten:

- Lokal gebaute Images, die keinen Registry-Digest tragen
- Referenzen, die auf einen Digest gepinnt sind und sich daher nicht verändern können
- Private Registries, da Dozzle keine eigenen Zugangsdaten hat
- Kubernetes, wo das Ausrollen von Images Sache des Clusters ist

### Dozzle selbst aktualisieren

Die `Update`-Aktion am eigenen Container von Dozzle aktualisiert Dozzle an Ort und Stelle. Sie zieht das neue Image und übergibt den Austausch an einen kurzlebigen Hilfscontainer, Dozzle ist also ein paar Sekunden weg und kommt mit der neuen Version, derselben Konfiguration und denselben Volumes zurück. Das geht auch nach Zeitplan. Unter [So funktioniert das Selbst-Update](/de/guide/setup-wizard#self-update) steht, was erhalten bleibt und welche Setups nicht unterstützt werden. Läuft Dozzle als Swarm-Service, wird es über den Orchestrator aktualisiert. Dozzle-Agents auf anderen Hosts sind gewöhnliche Container und aktualisieren sich wie alles andere.

## Mehrere Container auf einmal aktualisieren

Mit eingeschalteten Aktionen prüft das Dashboard alle Container in einem Durchgang. Veraltete Container bekommen einen kleinen Ring neben ihrem Namen, und über der Containerliste erscheint ein Button **N Updates**. Beide öffnen die Update-Schublade, die jeden Container mit einem neueren Image auflistet, alle bereits ausgewählt. Entferne den Haken bei allem, was du in Ruhe lassen willst, und drücke dann **Aktualisieren**.

Hosts werden parallel aktualisiert, und jeder Host aktualisiert einen Container nach dem anderen, damit kein Daemon ein Dutzend Images gleichzeitig ziehen muss. Die Schublade zeigt, wie jeder Container die Phasen **Lädt**, **Neu erstellen** und **Aktualisiert** durchläuft, und ein Fehler bei einem Container hält die übrigen nicht auf. Das Update läuft auf dem Server, das Schließen des Tabs unterbricht es also nicht. Öffnest du die Schublade erneut, zeigt sie den aktuellen Stand.

Steht der eigene Container von Dozzle in der Liste, kommt er immer zuletzt dran, weil sein Update Dozzle neu startet.

Mit `DOZZLE_IMAGE_CHECK_MODE=manual` lautet der Button **Nach Updates suchen**, bis du ihn drückst. Mit ausgeschalteten Aktionen sieht das Dashboard genauso aus wie bisher, und das Menü jedes einzelnen Containers zeigt weiterhin an, wenn ein Update verfügbar ist.

## Container automatisch aktualisieren {#auto-updating-containers}

Dozzle kann Container nach Zeitplan aktualisieren. Einen Container nimmst du per Label mit auf:

```yaml [docker-compose.yml]
services:
  whoami:
    image: traefik/whoami:latest
    labels:
      dev.dozzle.auto-update: true
```

Container mit diesem Label folgen demselben Zeitplan wie [das automatische Update von Dozzle selbst](/de/guide/setup-wizard#_4-automatische-updates), den du im Einrichtungsassistenten oder mit `DOZZLE_AUTO_UPDATE` und `DOZZLE_AUTO_UPDATE_TIME` festlegst. Zu dieser Zeit prüft Dozzle jeden markierten Container gegen seine Registry und aktualisiert nur die, für die es ein neueres Image gibt. Erst kommen die Container dran, Dozzle zuletzt.

Das automatische Update ist bewusst Opt-in. Eine Datenbank auf einem beweglichen Tag wie `postgres:latest` kann auf eine neue Hauptversion springen, deren Datendateien sie nicht lesen kann. Versieh also nur Container mit dem Label, bei denen es dich nicht stört, wenn sie ohne dein Zutun ersetzt werden. Container, die Dozzle [nicht prüfen kann](#was-sich-nicht-prufen-lasst), etwa solche aus einer privaten Registry, werden nie automatisch aktualisiert.

Das automatische Update läuft im Server-Modus, auch für Container auf [Remote-Agents](/de/guide/agent). Es setzt eingeschaltete Aktionen voraus.
