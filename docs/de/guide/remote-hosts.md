---
title: Entfernte Hosts einrichten
sourceHash: 6ae09165c838
---

# Entfernte Hosts einrichten

<Badge type="warning" text="Docker Only" />

Dozzle kann sich mit entfernten Docker-Hosts verbinden. Das ist nützlich, wenn Dozzle in einem Container läuft und du einen anderen Docker-Host überwachen willst.

Mit Dozzle-Agents kannst du dich allerdings mit entfernten Hosts verbinden, ohne den Docker-Socket freizugeben. Mehr dazu auf der Seite [Agent](/de/guide/agent).

Dozzle-Agents machen es überflüssig, den Docker-Socket nach außen freizugeben, lassen sich aber nicht mit einem Docker-Socket-Proxy innerhalb des Agent-Stacks kombinieren. Wenn du einen Socket-Proxy allein ohne Agent nutzen willst, siehe den Abschnitt [Verbindung über einen Socket-Proxy](#connecting-with-a-socket-proxy).

> [!WARNING]
> Entfernte Hosts wurden durch Agents ersetzt. Agents bieten eine sicherere Möglichkeit, sich mit entfernten Hosts zu verbinden. Entfernte Hosts werden zwar weiterhin unterstützt, empfohlen sind aber Agents. Mehr Informationen und Beispiele findest du auf der Seite [Agent](/de/guide/agent). Für einen Vergleich siehe den Abschnitt [Agents im Vergleich zu entfernten Verbindungen](/de/guide/agent#comparing-agents-with-remote-connection). Problemen von Nutzern mit entfernten Hosts kann ich nicht nachgehen, das kostet zu viel Zeit.

## Verbindung zu entfernten Hosts mit TLS

Entfernte Hosts werden mit `--remote-host` oder `DOZZLE_REMOTE_HOST` konfiguriert. Alle Zertifikate müssen ins Verzeichnis `/certs` eingebunden werden. Das Verzeichnis `/certs` erwartet `/certs/{ca,cert,key}.pem` oder, bei mehreren Hosts, `/certs/{host}/{ca,cert,key}.pem`.

Beachte, dass der Wert `{host}` hier die konfigurierte IP oder FQDN meint, nicht die [optionale Bezeichnung](#adding-labels-to-hosts).

Mehrere `--remote-host`-Optionen können mehrere Hosts angeben. Bei `DOZZLE_REMOTE_HOST` muss der Wert dagegen kommagetrennt sein.

::: code-group

```sh [cli]
$ docker run -v /var/run/docker.sock:/var/run/docker.sock -v /path/to/certs:/certs -p 8080:8080 amir20/dozzle --remote-host tcp://167.99.1.1:2376 --remote-host tcp://167.99.1.2:2376
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /path/to/certs:/certs
    ports:
      - 8080:8080
    environment:
      DOZZLE_REMOTE_HOST: tcp://167.99.1.1:2376,tcp://167.99.1.2:2376
```

:::

## Verbindung über einen Socket-Proxy

In einem privaten Netzwerk kannst du den [Docker Socket Proxy](https://github.com/Tecnativa/docker-socket-proxy) nutzen, der `docker.sock` ohne TLS bereitstellt. Damit brauchst du keinen Dozzle-Agent, Dozzle verbindet sich stattdessen direkt mit dem Socket-Proxy. Dozzle wird nie versuchen, nach Docker zu schreiben, braucht aber Zugriff auf die List-APIs. Der folgende Befehl startet einen Proxy mit minimalen Rechten:

```sh
$ docker container run --privileged -e CONTAINERS=1 -e INFO=1 -v /var/run/docker.sock:/var/run/docker.sock -p 2375:2375 tecnativa/docker-socket-proxy
```

> [!TIP]
> `CONTAINERS=1` wird benötigt, um laufende Container aufzulisten. `EVENTS` wird ebenfalls gebraucht, ist aber standardmäßig aktiv. `INFO=1` wird für die Systeminformationen benötigt.

Dozzle sollte dann ohne Zertifikate laufen. Hier ein Beispiel:

::: code-group

```sh [cli]
$ docker run -p 8080:8080 amir20/dozzle --remote-host tcp://123.1.1.1:2375
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    ports:
      - 8080:8080
    environment:
      DOZZLE_REMOTE_HOST: tcp://123.1.1.1:2375
```

:::

Bei entfernten Hosts ist das Einbinden von `/var/run/docker.sock` optional. Du brauchst mindestens einen entfernten Host, mit dem du dich verbindest.

> [!WARNING]
> Der Docker Socket Proxy stellt die Docker-API ins Internet. Das kann ein Sicherheitsrisiko sein, wenn er nicht ordentlich abgesichert ist.

## Hosts mit Bezeichnungen versehen

`--remote-host` unterstützt Host-Bezeichnungen, die du mit `|` an die Verbindungszeichenfolge anhängst. Zum Beispiel nutzt `--remote-host tcp://123.1.1.1:2375|foobar.com` foobar.com als Bezeichnung in der Oberfläche. Ein vollständiges Beispiel mit CLI oder Compose:

::: code-group

```sh
docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --remote-host tcp://123.1.1.1:2375|foobar.com
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /path/to/certs:/certs
    ports:
      - 8080:8080
    environment:
      DOZZLE_REMOTE_HOST: tcp://167.99.1.1:2376|foo.com,tcp://167.99.1.2:2376|bar.com
```

:::

> [!WARNING]
> Dozzle nutzt die Docker-API, um Informationen über Hosts zu sammeln. Jeder Agent braucht eine eindeutige Host-ID. Dafür wird Dockers System-ID oder die Node-ID verwendet. Bei Swarm ist es die Node-ID. Wenn du nicht alle Hosts siehst, hast du möglicherweise doppelte Hosts mit derselben Host-ID konfiguriert. Entferne zur Behebung die Datei `/var/lib/docker/engine-id`. Mehr dazu in der [FAQ](/de/guide/faq#i-am-seeing-duplicate-hosts-error-in-the-logs-how-do-i-fix-it).

## Bezeichnung von localhost ändern

`localhost` ist eine besondere Verbindung und wird anders konfiguriert als `--remote-host`. Die Bezeichnung für localhost änderst du über `--hostname` oder die Umgebungsvariable `DOZZLE_HOSTNAME`. Beispiele zur Verwendung dieser Option findest du auf der Seite [Hostname](/de/guide/hostname).
