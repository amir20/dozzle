---
title: FAQ
sourceHash: c964c824dcab
---

# Häufig gestellte Fragen

## Dozzle startet nicht und meldet `client version 1.x is too new`. Was bedeutet das?

Dozzle benötigt Docker Engine 19.03 oder neuer (API-Version 1.40+). Ältere Daemons, zum Beispiel Docker 18.06 (API 1.38), werden vom zugrunde liegenden Docker SDK nicht unterstützt und scheitern beim Start mit einer Meldung wie `failed to create docker client: ... client version 1.54 is too new. Maximum supported API version is 1.38`.

Aktualisiere Docker Engine auf eine unterstützte Version. Als vorübergehende Notlösung kannst du Dozzle auf `v10.5.2` oder älter festnageln, weil dort noch ein Docker SDK verwendet wurde, das auf ältere API-Versionen heruntergehandelt hat.

## Wie aktualisiere ich Dozzle?

Dozzle folgt den üblichen Docker-Image-Praktiken. Zum Aktualisieren holst du das neue Image und erstellst den Container neu:

```sh
docker pull amir20/dozzle:latest
docker compose up -d dozzle
```

Benutzereinstellungen, Benachrichtigungsregeln und weiterer Zustand liegen in `/data` (siehe unten), lass dieses Volume also über Upgrades hinweg eingehängt. Im Produktivbetrieb solltest du ein konkretes Tag festnageln (z. B. `amir20/dozzle:v10.9.2`) statt `latest` zu verwenden, damit Upgrades bewusst passieren. Die Release Notes findest du auf der [GitHub-Releases-Seite](https://github.com/amir20/dozzle/releases). Ein Rollback ist so einfach wie das erneute Deployen eines älteren Tags.

## Meine Plattform umhüllt den Container-Entrypoint und Dozzle scheitert mit `no such file or directory`

Das Standard-Image wird `FROM scratch` gebaut und enthält damit nur das Dozzle-Binary und sonst nichts. Keine Shell, kein Interpreter.

Manche Plattformen ergänzen optionale Funktionen, indem sie einen `#!/bin/sh`-Wrapper über den Container-Entrypoint mounten und den ursprünglichen erneut ausführen. Der Tailscale-Schalter pro Container bei Unraid funktioniert so, ebenso einige Sidecar- und Init-Injectoren. Ohne `/bin/sh` im Image kann der Wrapper nicht ausgeführt werden, und der Container beendet sich mit einem Fehler, der den Wrapper nennt statt der fehlenden Shell:

```
exec /opt/unraid/tailscale: no such file or directory
```

Nutze in diesen Fällen die `alpine`-Variante, also dasselbe Binary auf einer Alpine-Basis:

```sh
docker run \
  --volume=/var/run/docker.sock:/var/run/docker.sock \
  -p 8080:8080 \
  amir20/dozzle:alpine
```

Versionierte Tags folgen demselben Muster (`amir20/dozzle:v10.9.2-alpine`). Für alles andere bleibt das auf scratch basierende `latest` das empfohlene Image, da es deutlich kleiner ist und keine Distributionspakete gepatcht werden müssen.

## Was wird in `/data` gespeichert und wie sichere ich es?

Im Verzeichnis `/data` legt Dozzle alles ab, was einen Neustart des Containers überleben muss:

- `users.yml` / `users.yaml` — Benutzerdatei für die einfache Authentifizierung (falls du eine angelegt hast)
- Benachrichtigungsregeln, Ziele und Zustellstatus
- Benutzerbezogene UI-Einstellungen (nur im Mehrbenutzermodus; im Einzelbenutzermodus liegen die Einstellungen im localStorage des Browsers)
- Ein paar interne Dateien, etwa der Status ausgeblendeter Ankündigungen

Das Verzeichnis ist klein (typischerweise deutlich unter 10 MB) und lässt sich mit einem einfachen `tar` oder `rsync` des eingehängten Volumes sichern. Beim Upgrade oder Umzug auf einen neuen Host nimmt das `/data`-Volume alle Einstellungen mit.

## Ich habe Dozzle installiert, aber die Logs sind langsam oder laden nie. Was kann ich tun?

Dozzle nutzt Server Sent Events (SSE) und verbindet sich über einen HTTP-Stream mit dem Server, ohne die Verbindung zu schließen. Puffert ein Proxy diese Verbindung, erhält Dozzle die Daten nie und wartet ewig darauf, dass der Reverse Proxy den Puffer leert. Seit Version `1.23.0` sendet Dozzle den Header `X-Accel-Buffering: no`, der das Puffern in Reverse Proxies unterbinden sollte. Manche Proxies ignorieren diesen Header allerdings. In diesen Fällen musst du das Puffern explizit abschalten.

### Puffern in nginx abschalten

Unten ein Beispiel mit nginx und `proxy_pass`, um das Puffern abzuschalten:

```
server {
    ...

    location / {
        proxy_pass                  http://<dozzle.container.ip.address>:8080;
    }

    location /api {
        proxy_pass                  http://<dozzle.container.ip.address>:8080;

        proxy_buffering             off;
        proxy_cache                 off;
    }
}
```

### Komprimierung in traefik abschalten

Der Reverse Proxy Traefik kann über Middlewares Komprimierung unterstützen. Die übliche Konfiguration sieht so aus:

```
http:
  middlewares:
    middlewares-compress:
      compress: {}
```

Mit dieser Einrichtung kann es passieren, dass bestimmte Container keine Logs mehr in dozzle anzeigen, wenn du dozzle über traefik öffnest (z. B. dozzle.mydomain.com).
Dieselbe dozzle-Instanz zeigt die Logs jedoch an, wenn du sie direkt aufrufst (z. B. localhost:8080).

Container, bei denen das beobachtet wurde (unvollständige Liste), sind: dozzle, homepage, glances, filebrowser.

Damit die Logs wieder fließen, schließe `text/event-stream` von der Komprimierungs-Middleware aus:

```
http:
  middlewares:
    middlewares-compress:
      compress:
        excludedContentTypes:
          - text/event-stream
```

## Wir haben Tools, die Dozzle nutzen, sobald ein neuer Container erstellt wird. Wie bekomme ich einen direkten Link zu einem Container über seinen Namen?

Dozzle hat eine spezielle [Route](https://github.com/amir20/dozzle/blob/master/assets/pages/show.vue), mit der sich Container über den Namen suchen und anschließend dorthin weiterleiten lassen. Hast du zum Beispiel einen Container mit dem Namen `"foo.bar"` und der ID `abc123`, kannst du deine Benutzer auf `/show?name=foo.bar` schicken, was auf `/container/abc123` weiterleitet.

## Ich habe Dozzle installiert, aber der Speicherverbrauch wird nicht angezeigt!

_Das ist ein Problem, das nur ARM-Geräte betrifft._

Dozzle nutzt die Docker-API, um Informationen über den Speicherverbrauch der Container zu sammeln. Wird der Speicherverbrauch nicht angezeigt, liefert die Docker-API ihn wahrscheinlich nicht.

Du kannst das prüfen, indem du docker info ausführst; du solltest Folgendes sehen:

```
WARNING: No memory limit support
WARNING: No swap limit support
```

In diesem Fall musst du deiner Datei `/boot/cmdline.txt` die folgende Zeile hinzufügen und das Gerät neu starten:

```
cgroup_enable=cpuset cgroup_enable=memory cgroup_memory=1
```

## Ich sehe einen Fehler über doppelte Hosts in den Logs. Wie behebe ich das?

Wenn du den folgenden Fehler in den Logs siehst, hast du womöglich mehrere Hosts mit derselben Host-ID konfiguriert:

```
time="2024-07-10T13:35:53Z" level=warning msg="duplicate host ID: *********, Endpoint: 1.1.1.1:7007 found, skipping"
```

Dozzle nutzt die Docker-API, um Informationen über die Hosts zu sammeln. Jeder Host braucht eine eindeutige ID. Über diese ID wird der Host in der Oberfläche identifiziert. Im Swarm-Modus verwendet Dozzle die Node-ID aus `docker system info`. Ohne Swarm-Modus nutzt Dozzle die System-ID aus `docker system info` als Host-ID.

Manchmal werden VMs aus Backups mit derselben Host-ID wiederhergestellt. Dann hält Dozzle den Host für bereits vorhanden und nimmt ihn nicht in die Hostliste auf. Um das zu beheben, musst du die Datei `/var/lib/docker/engine-id` löschen. Sie enthält die Host-ID und wird beim Start des Docker-Daemons erzeugt.

## Ich sehe einen Fehler über einen nicht gefundenen Host in den Logs. Wie behebe ich das?

Das sollte hauptsächlich ein Podman-Fehler sein: Podman erzeugt keine engine-id wie Docker.
Wenn du Docker nutzt, prüfe, ob die Datei `engine-id` mit den richtigen Rechten in `/var/lib/docker` existiert und eine UUID enthält.

So behebst du den Fehler:

1. Lege die Ordner an: `mkdir -p /var/lib/docker`
2. Installiere bei Bedarf uuidgen
3. Erzeuge mit uuidgen eine UUID: `uuidgen > engine-id`

Die Datei engine-id sollte jetzt eine UUID enthalten.

Ein Beispiel-Setup für Ansible findest du unter [Podman](/de/guide/podman)

Möglicherweise musst du dein bestehendes Dozzle-Deployment unter Podman aufräumen, den Container stoppen und die zugehörigen Daten (Container/Volumes) entfernen. Danach kannst du den Dozzle-Container erneut deployen, und deine Logs sollten jetzt erscheinen.

## Warum sehe ich nur laufende Container? Wie sehe ich gestoppte Container?

Standardmäßig zeigt Dozzle nur laufende Container. Um gestoppte Container zu sehen, musst du die Option `Show Stopped Containers` in den Einstellungen aktivieren. Sie ist standardmäßig deaktiviert, damit weniger Container in der Oberfläche erscheinen.

## Kann ich meine Einstellungen über mehrere Dozzle-Instanzen hinweg synchronisieren?

Im Einzelbenutzermodus speichert Dozzle die Einstellungen im Local Storage des Browsers. Damit stehen sie nur in dem Browser zur Verfügung, in dem sie gesetzt wurden. Damit Dozzle Einstellungen über mehrere Instanzen hinweg synchronisieren kann, muss es wissen, wer der Benutzer ist. Im Mehrbenutzermodus nutzt Dozzle den Benutzernamen, um die Einstellungen auf der Festplatte zu speichern und über mehrere Instanzen hinweg zu synchronisieren. Diese Informationen liegen im Verzeichnis `/data`. Wenn du Einstellungen über mehrere Instanzen synchronisieren willst, musst du den Mehrbenutzermodus [aktivieren](/de/guide/authentication) und einen Benutzernamen angeben.

## Warum unterstützt Dozzle keine Benachrichtigungen für Slack, Discord, Telegram, E-Mail usw. direkt?

Dozzle legt sich bewusst nicht darauf fest, wohin deine Alarme gehen. Statt Integrationen für einzelne Benachrichtigungsplattformen mitzuliefern, bietet Dozzle **Webhooks** mit anpassbaren Payload-Templates. Damit kannst du Alarme an _jeden_ Dienst schicken, der HTTP-Anfragen annimmt: Slack, Discord, Telegram, ntfy, PagerDuty, Opsgenie oder deine eigenen internen Tools, ohne darauf zu warten, dass Dozzle explizite Unterstützung ergänzt.

Für diesen Ansatz gibt es mehrere Gründe:

- **Universalität.** Webhooks funktionieren mit praktisch jeder Benachrichtigungsplattform. Anbieterspezifische Integrationen würden nur einen Bruchteil dessen abdecken, was Nutzer brauchen, Webhooks decken alles ab.
- **Wartung.** Jede Anbieterintegration bringt eigene API-Eigenheiten, Authentifizierungsabläufe, Rate Limits und Breaking Changes mit. Sie zu unterstützen hieße, dass die Dozzle-Maintainer Probleme mit Drittanbieterdiensten debuggen müssten, und das liegt außerhalb dessen, was ein Log-Viewer leisten sollte.
- **Einfachheit.** Dozzle ist ein leichtgewichtiges, fokussiertes Werkzeug zum Betrachten von Docker-Logs. Eine generische Benachrichtigungsschicht hält die Codebasis klein und das Projekt tragfähig.

Wenn du eine stärker vorgegebene Erfahrung mit umfangreicheren Anbieterintegrationen brauchst (z. B. Web-Push-Benachrichtigungen, ntfy-Aktionsbuttons), ist [Dozzle Cloud](/de/guide/dozzle-cloud) genau dafür gedacht.

Wie du Webhooks mit deinem bevorzugten Dienst einrichtest, steht im Leitfaden [Alarme & Webhooks](/de/guide/alerts-and-webhooks). Er enthält fertige Payload-Templates für Slack, Discord und ntfy, die du direkt nutzen oder anpassen kannst.

## Warum halten dockerd und containerd bei Dozzle eine leicht erhöhte CPU-Last, obwohl kein Browser verbunden ist?

Dozzle streamt Container-Statistiken bis zu 6 Stunden weiter (2 Stunden unter Kubernetes), nachdem sich der letzte Browser getrennt hat, und fährt den Stats-Collector dann von selbst herunter. Das ist Absicht. Die Statistiken werden fortlaufend gestreamt, damit du beim erneuten Öffnen der Oberfläche den bisherigen CPU- und Speicherverlauf siehst statt eines leeren Diagramms. Würde das Streaming beim Schließen des Tabs sofort enden, gäbe es keinen Verlauf zu zeigen.

Der Preis dafür ist eine kleine, gleichmäßige CPU-Last in dockerd und containerd, da die Stats-API von Docker auf Polling basiert. Ein Neustart des Dozzle-Containers setzt den Timer sofort zurück, deshalb fällt der Host nach einem Neustart wieder in den Leerlauf. Das ist bewusst nicht konfigurierbar. Ein kurzes Timeout würde andere Funktionen brechen, die davon ausgehen, dass die Statistiken weiter gestreamt werden, und würde damit den Sinn des Statistikverlaufs zunichtemachen.

## Meine Dozzle-Instanzen laufen im Swarm-Modus in Timeouts oder ich sehe hinter einem Load Balancer nicht alle Swarm-Nodes. Wie behebe ich das?

Im Swarm-Modus brauchen Dozzle-Instanzen unter Umständen ein eigenes Overlay-Netzwerk. Wenn du dich uneinheitlich verhaltende Verbindungen zu verschiedenen Dozzle-Nodes siehst, richte ein separates Overlay-Netzwerk ein, das nur die Dozzle-Instanzen enthält, wie unten gezeigt:

```
services:
  logs:
    ...
    networks: [ traefik, dozzle ]
    ...

networks:
  dozzle:
    driver: overlay
  traefik:
    external: true
```

Das externe Netzwerk `traefik` ist das Overlay-Netzwerk für die Service Discovery des Load Balancers, und wir haben ein neues Overlay-Netzwerk `dozzle` angelegt, damit die Dozzle-Nodes miteinander sprechen können.
