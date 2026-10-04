---
title: Container-Aktionen
sourceHash: 121d4b250806
---

# Container-Aktionen

<Badge type="warning" text="Docker Only" />

Dozzle unterstützt Container-Aktionen: Über das Dropdown-Menü rechts neben den Container-Statistiken kannst du Container `start`en, `stop`pen, neu starten (`restart`), entfernen (`remove`) und aktualisieren (`update`). Diese Funktion ist standardmäßig **deaktiviert** und lässt sich aktivieren, indem du die Umgebungsvariable `DOZZLE_ENABLE_ACTIONS` auf `true` setzt.

Die Aktion `update` lädt das neueste Image für den Container und erstellt ihn mit derselben Konfiguration neu — praktisch, um einen Container an Ort und Stelle zu aktualisieren, ohne seine Compose-Datei zu bearbeiten. `update` hat nur dann einen spürbaren Effekt, wenn das Image ein bewegliches Tag nutzt (z. B. `latest`, `stable`); bei einem fest gepinnten Tag wird schlicht dasselbe Image erneut geladen.

Der alte Container bleibt umbenannt erhalten, bis der neue 10 Sekunden lang ohne Neustart gelaufen ist und, falls sein Image einen Healthcheck hat, als gesund gemeldet wurde. Startet der neue Container nicht, beendet er sich, startet er neu oder wird er ungesund, entfernt Dozzle ihn und stellt den alten wieder her. Das Update meldet dann **zurückgerollt** mit dem Grund. Der neue Container bekommt das Label `dev.dozzle.previous-image` mit der Image-ID, die er ersetzt hat, und `dev.dozzle.previous-ref` mit dem Digest `repo@sha256:…` dieses Images (fehlt bei lokal gebauten Images). Ein gestoppter Container wird nie aktualisiert, denn er kann absichtlich gestoppt sein. Ein neueres Image wird für ihn weiterhin angezeigt, aber **Aktualisieren** wird nicht angeboten, der Zeitplan überspringt ihn, und ein Update, das trotzdem angefordert wird, etwa aus Dozzle Cloud, wird mit „Starten Sie zuerst den Container“ abgelehnt. Sobald er läuft, lässt er sich aktualisieren.

> [!WARNING]
> `remove` löscht den Container: Daten in seiner beschreibbaren Schicht gehen verloren, und seine anonymen Volumes bleiben losgelöst zurück. `update` erstellt den Container neu und behält jedes Volume, auch anonyme, sowie jeden Bind-Mount. Verloren gehen nur Daten, die in die beschreibbare Schicht des Containers geschrieben wurden.

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

Um einen einzelnen Container nicht mehr zu prüfen, etwa einen bewusst auf eine Version gepinnten, versiehst du ihn mit einem Label. Damit bleibt er auch aus dem [Zeitplan für automatische Updates](#auto-updating-containers) heraus.

```yaml [docker-compose.yml]
services:
  database:
    image: postgres:18-alpine
    labels:
      dev.dozzle.update: off
```

Bei einem gefundenen Update kann auch eine Benachrichtigung erscheinen. Sie ist standardmäßig aus und findet sich unter den Einstellungen.

### Was sich nicht prüfen lässt

Bei manchen Containern gibt es nichts zu vergleichen, und Dozzle bleibt still statt zu raten:

- Lokal gebaute Images, die keinen Registry-Digest tragen
- Referenzen, die auf einen Digest gepinnt sind und sich daher nicht verändern können
- Private Registries, da Dozzle keine eigenen Zugangsdaten hat. In Kubernetes gilt das auch für Images, die über `imagePullSecrets` geladen werden.

### Kubernetes

Im Kubernetes-Modus vergleicht die Prüfung den Digest aus dem Pod-Status mit der Registry. Auf containerd-Clustern wie k3s, EKS und GKE funktioniert das ohne zusätzliche Berechtigungen. Der Hinweis dient nur zur Information und kommt nie mit einem `Update`-Button, weil das Image eines Pods zu seinem Workload gehört. Bei einem wandernden Tag wie `:latest` mit `imagePullPolicy: Always` holt ein [Rollout-Neustart](/de/guide/k8s#rollout-restart) das neue Image. Alles andere ist eine Änderung an der Spec des Workloads. Der Update-Button und die Update-Schublade im Dashboard werden im Kubernetes-Modus nicht angezeigt.

### Dozzle selbst aktualisieren

Die `Update`-Aktion am eigenen Container von Dozzle aktualisiert Dozzle an Ort und Stelle. Sie zieht das neue Image und übergibt den Austausch an einen kurzlebigen Hilfscontainer, Dozzle ist also ein paar Sekunden weg und kommt mit der neuen Version, derselben Konfiguration und denselben Volumes zurück. Das geht auch nach Zeitplan. Unter [So funktioniert das Selbst-Update](/de/guide/setup-wizard#self-update) steht, was erhalten bleibt und welche Setups nicht unterstützt werden. Läuft Dozzle als Swarm-Service, wird es über den Orchestrator aktualisiert. Dozzle-Agents auf anderen Hosts sind gewöhnliche Container und aktualisieren sich wie alles andere.

## Mehrere Container auf einmal aktualisieren

Mit eingeschalteten Aktionen prüft das Dashboard alle Container in einem Durchgang. Veraltete Container bekommen einen kleinen Ring neben ihrem Namen, und über der Containerliste erscheint ein Button **N Updates**. Beide öffnen die Update-Schublade, die jeden Container mit einem neueren Image auflistet, alle bereits ausgewählt. Entferne den Haken bei allem, was du in Ruhe lassen willst, und drücke dann **Aktualisieren**.

Hosts werden parallel aktualisiert, und jeder Host aktualisiert einen Container nach dem anderen, damit kein Daemon ein Dutzend Images gleichzeitig ziehen muss. Die Schublade zeigt, wie jeder Container die Phasen **Lädt**, **Neu erstellen** und **Aktualisiert** durchläuft, und ein Fehler bei einem Container hält die übrigen nicht auf. Das Update läuft auf dem Server, das Schließen des Tabs unterbricht es also nicht. Öffnest du die Schublade erneut, zeigt sie den aktuellen Stand.

Steht der eigene Container von Dozzle in der Liste, kommt er immer zuletzt dran, weil sein Update Dozzle neu startet.

Mit `DOZZLE_IMAGE_CHECK_MODE=manual` lautet der Button **Nach Updates suchen**, bis du ihn drückst. Mit ausgeschalteten Aktionen sieht das Dashboard genauso aus wie bisher, und das Menü jedes einzelnen Containers zeigt weiterhin an, wenn ein Update verfügbar ist.

## Container automatisch aktualisieren {#auto-updating-containers}

Dozzle kann Container nach Zeitplan aktualisieren. Eingerichtet wird das unter **Einstellungen → Aktualisierungen** oder im [Einrichtungsassistenten](/de/guide/setup-wizard#auto-update):

- **Wann:** aus, täglich oder wöchentlich am Sonntag, zu einer Uhrzeit. Entspricht `DOZZLE_AUTO_UPDATE` und `DOZZLE_AUTO_UPDATE_TIME`.
- **Welche Container:** **Nur Dozzle**, **Container mit Label** (Standard) oder **Alles**. Dozzle selbst folgt dem Zeitplan in allen drei Fällen. Entspricht `DOZZLE_UPDATE_CONTAINERS` (`off`, `labelled` oder `all`).

Ein Label am Container entscheidet den Rest:

| `dev.dozzle.update` | Was passiert                                                                                                    |
| ------------------- | --------------------------------------------------------------------------------------------------------------- |
| `auto`              | Wird nach Zeitplan aktualisiert, außer **Welche Container** steht auf **Nur Dozzle**                            |
| _(kein Label)_      | Wird bei **Alles** nach Zeitplan aktualisiert. Sonst geprüft und als Update angezeigt, das du selbst einspielst |
| `off`               | Wird nie geprüft und nie aktualisiert                                                                           |

```yaml
services:
  app:
    image: ghcr.io/example/app:latest
    labels:
      dev.dozzle.update: auto
```

Ältere Labels funktionieren weiter: `dev.dozzle.auto-update=true` gilt als `auto` und `dev.dozzle.update-check=false` als `off`.

Zur geplanten Zeit prüft Dozzle jeden Container im Zeitplan gegen seine Registry und aktualisiert nur die, für die es ein neueres Image gibt, Dozzle selbst zuletzt. Jedes Update ist der oben beschriebene sichere Austausch: Bleibt ein neuer Container nicht stabil, kommt der alte zurück. Übersprungen werden Container, die gestoppt oder ungesund sind, die Dozzle [nicht prüfen kann](#was-sich-nicht-prufen-lasst) oder die vom angebotenen Image [zurückgerollt](#rolling-back) wurden. **Einstellungen → Aktualisierungen** listet die Container, die der nächste Lauf aktualisiert.

Bei **Alles** kann eine Datenbank auf einem beweglichen Tag wie `postgres:latest` auf eine Hauptversion springen, deren Datendateien sie nicht lesen kann. Wählst du **Alles**, werden die Container aufgelistet, die Daten in benannten Volumes halten. Gib ihnen das Label `dev.dozzle.update: off`, um sie herauszunehmen.

**Welche Container** wird als `updateContainers` in der [`dozzle.yml`](/de/guide/setup-wizard#dozzle-yml) gespeichert. Eine Änderung in der Oberfläche braucht deshalb `/data` auf einem Volume. Automatische Updates laufen im Servermodus, auch für Container auf [Remote-Agents](/de/guide/agent), und setzen eingeschaltete Aktionen voraus. Du kommst von Watchtower? Siehe [Umstieg von Watchtower](/de/guide/moving-from-watchtower).

## Alte Images aufräumen {#cleaning-up-old-images}

Jedes Update lässt das ersetzte Image auf dem Host zurück, deshalb entfernt Dozzle nach einem Update alte Images, ähnlich wie Watchtowers `--cleanup`. Das passiert immer, bei jedem Update: geplant, über die Aktion `Update` eines Containers oder über die Update-Leiste. Es muss nichts eingeschaltet werden.

Dozzle behält das Image, mit dem der Container bisher lief, damit er noch dorthin zurück kann, und entfernt das davor. Ein Update von 1.4.1 auf 1.4.2 entfernt 1.4.0 und behält 1.4.1, sodass jeder Container höchstens ein Ersatz-Image behält. Welches Image entfernt wird, liest Dozzle aus dem Label `dev.dozzle.previous-image` des alten Containers. Das erste Update eines Containers entfernt deshalb nichts.

Aufgeräumt wird erst, wenn das Update durch ist und der alte Container entfernt ist. Ein zurückgerolltes Update entfernt nichts. Dozzle entfernt nur ein Image ohne Tag, das kein Container nutzt: Ein Image, das noch ein Tag hat, etwa eines, das du selbst gepullt oder gebaut hast, bleibt erhalten, und das Entfernen geschieht ohne Zwang, also verweigert Docker es, solange ein anderer Container es noch nutzt, ob laufend oder gestoppt. Eine Weigerung lässt das Update nie fehlschlagen.

Container auf [Remote-Agents](/de/guide/agent) werden genauso aufgeräumt, ebenso der eigene Container von Dozzle: Der Hilfscontainer des [Selbst-Updates](/de/guide/setup-wizard#self-update) entfernt das Image vor dem vorherigen, sobald das neue Dozzle stabil läuft. Swarm-Services, auch ein als Swarm-Service laufendes Dozzle, werden nicht aufgeräumt, da jeder Node seine eigenen Images hat und Swarm seinen Task-Verlauf selbst bereinigt.

## Zurückrollen {#rolling-back}

Ein Update zurückzurollen ist eine Funktion von [Dozzle Cloud](/de/guide/dozzle-cloud). Dozzle Cloud beobachtet jedes Update aus dem Zeitplan und bietet an, es zurückzurollen, wenn die neue Version Fehler zeigt. Dozzle tauscht den Container dann zurück auf das Image, das er vorher lief und das sein Label `dev.dozzle.previous-image` nennt, genauso wie bei einem Update: Der aktuelle Container bleibt erhalten, bis das vorherige Image stabil läuft, und kommt zurück, wenn nicht. Einstellungen und Volumes bleiben, wie sie sind. Es wird nichts heruntergeladen: Liegt das vorherige Image nicht mehr auf dem Host, schlägt das Zurückrollen fehl und der Container bleibt unverändert.

Der zurückgerollte Container bekommt das Label `dev.dozzle.rolled-back-from` mit dem Image, das er verlassen hat, und der Zeitplan für automatische Updates lässt ihn in Ruhe, bis sein Tag auf ein neueres Image zeigt. Sobald das Zurückrollen stabil läuft, wird das verlassene Image wie jedes alte Image [aufgeräumt](#cleaning-up-old-images), also nur entfernt, wenn kein Tag mehr darauf zeigt.

Zurückrollen funktioniert für eigenständige Container, auch auf [Remote-Agents](/de/guide/agent). Für Swarm-Services, Kubernetes und den eigenen Container von Dozzle ist es nicht verfügbar.

## Updates in der Log-Ansicht

Aktualisiert Dozzle einen Container, nach Zeitplan, aus der Dozzle-Oberfläche oder aus Dozzle Cloud, beginnen die Logs des neuen Containers mit einer Markierung. Sie nennt das Image vorher und nachher und wer das Update gestartet hat. Auch ein Zurückrollen und ein rückgängig gemachtes Update werden markiert. Ist Dozzle Cloud verbunden, zeigt die Markierung außerdem, wie Dozzle Cloud das Update einschätzt, mit einem Link dorthin. Dozzle hält seine letzten Updates im Speicher, nach einem Neustart von Dozzle ist die Markierung also weg.
