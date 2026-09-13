---
title: Agent-Modus
sourceHash: a73c7ed62d8a
---

# Agent-Modus

<Badge type="warning" text="Nur Docker" />

Dozzle kann im Agent-Modus laufen und damit Docker-Hosts für andere Dozzle-Instanzen zugänglich machen. Die gesamte Kommunikation läuft über eine mit TLS gesicherte Verbindung. Du kannst Dozzle also auf einem entfernten Host betreiben und dich von deinem lokalen Rechner aus damit verbinden.

> [!WARNING] Ein Agent ist nur so privat wie das Netzwerk, in dem er lauscht
> Das Zertifikat, das Dozzle mitbringt, ist in jeder Kopie des Images dasselbe. Es verschlüsselt die Verbindung, belegt aber nicht, wer am anderen Ende sitzt. Wer Port `7007` erreichen kann, verbindet sein eigenes Dozzle mit deinem Agent, liest jedes Log auf diesem Host und führt Befehle in dessen Containern aus. Der Agent schaut dabei nicht auf `DOZZLE_ENABLE_SHELL` oder `DOZZLE_ENABLE_ACTIONS`, diese Flags steuern nur, was die UI anbietet.
>
> Halte Port `7007` in einem privaten Netzwerk, und wenn der Agent von außerhalb erreichbar ist, [erzeuge dein eigenes Zertifikat](#eigene-zertifikate), damit Agents nur deinen Hub akzeptieren.

> [!NOTE] Du nutzt Docker Swarm?
> Im Docker-Swarm-Modus brauchst du keine Agents. Dozzle erkennt sich selbst und bildet über den Swarm-Modus einen Cluster. Mehr dazu unter [Swarm-Modus](/de/guide/swarm-mode).

## <Icon icon="mdi:plus-box-outline" inline /> Einen Agent erstellen

Um einen Dozzle-Agent zu erstellen, startest du Dozzle mit dem Unterbefehl `agent`. Hier ein Beispiel:

::: code-group

```sh
docker run -v /var/run/docker.sock:/var/run/docker.sock -p 7007:7007 amir20/dozzle:latest agent
```

```yaml [docker-compose.yml]
services:
  dozzle-agent:
    image: amir20/dozzle:latest
    command: agent
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    ports:
      - 7007:7007
```

:::

> [!NOTE] Nutzer eines Docker-Socket-Proxy
> Bei einem entfernten Agent **KANNST** du **KEINEN** Socket-Proxy vor den Agent setzen. Dozzle-Agents **ERSETZEN** den Proxy, siehe [Entfernte Hosts](/de/guide/remote-hosts) für mehr Infos und dazu, wie du einen Socket-Proxy statt eines Agents nutzt.

Der Agent startet und lauscht auf Port `7007`. Über die Dozzle-Oberfläche verbindest du dich mit ihm, indem du IP-Adresse und Port des Agents angibst. Der Agent zeigt nur die Container, die auf dem Host verfügbar sind, auf dem er läuft.

> [!TIP]
> Du musst Port 7007 nicht freigeben, wenn du ein Docker-Netzwerk nutzt. Der Agent ist für andere Container im selben Netzwerk erreichbar. Das ist die sicherste Art, einen Agent zu betreiben, denn von außerhalb dieses Netzwerks kommt niemand an ihn heran.

## <Icon icon="mdi:connection" inline /> Mit einem Agent verbinden

Um dich mit einem Agent zu verbinden, gibst du dessen IP-Adresse und Port an. Hier ein Beispiel:

::: code-group

```sh
docker run -p 8080:8080 amir20/dozzle:latest --remote-agent agent:7007
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      - DOZZLE_REMOTE_AGENT=agent:7007
    ports:
      - 8080:8080 # Port der Dozzle-Oberfläche
```

:::

Beachte, dass du den lokalen Docker-Socket nicht einhängen musst, wenn du dich mit Agents verbindest. In dem Fall zeigt die Oberfläche nur die Container, die auf den Agents verfügbar sind.

> [!TIP]
> Wenn du die Container des Hosts ebenfalls in der Oberfläche sehen willst, hänge den Socket `docker.sock` ein, wie im Beispiel unter [Erste Schritte](/de/guide/getting-started) gezeigt.

> [!TIP]
> Du kannst dich mit mehreren Agents verbinden, indem du mehrere `DOZZLE_REMOTE_AGENT`-Umgebungsvariablen angibst. Zum Beispiel `DOZZLE_REMOTE_AGENT=agent1:7007,agent2:7007`.

## <Icon icon="mdi:group" inline /> Host-Gruppen

Wenn du viele Agents über verschiedene Umgebungen hinweg betreibst, kannst du jeden Agent einer benannten Gruppe zuordnen. Gruppen erscheinen als aufklappbare Abschnitte in der Seitenleiste, und jede Gruppe hat einen Button zum Zusammenführen, der die Logs aller Hosts der Gruppe gemeinsam anzeigt.

Das Format der Verbindungszeichenfolge ist `endpoint|name|group`, wobei alle drei Teile optional sind:

| Format                          | Ergebnis                        |
| ------------------------------- | ------------------------------- |
| `agent:7007`                    | Kein eigener Name, keine Gruppe |
| `agent:7007\|web-1`             | Eigener Name, keine Gruppe      |
| `agent:7007\|web-1\|Production` | Eigener Name + Gruppe           |
| `agent:7007\|\|Production`      | Standard-Hostname + Gruppe      |

::: code-group

```sh
docker run -p 8080:8080 amir20/dozzle:latest \
  --remote-agent agent1:7007|web-1|Production \
  --remote-agent agent2:7007|web-2|Production \
  --remote-agent agent3:7007|dev-1|Development
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      - DOZZLE_REMOTE_AGENT=agent1:7007|web-1|Production,agent2:7007|web-2|Production,agent3:7007|dev-1|Development
    ports:
      - 8080:8080
```

:::

Die Seitenleiste zeigt dann:

```
▾ Production
    web-1
    web-2
▾ Development
    dev-1
  ungrouped-host   ← agents without a group appear below
```

Ein Klick auf das Zusammenführen-Symbol neben einem Gruppennamen öffnet eine kombinierte Logansicht, die von allen Hosts dieser Gruppe streamt. Die zusammengeführte Ansicht ist auch direkt unter `/host-group/<group-name>` erreichbar.

Agents ohne Gruppe funktionieren weiterhin genau wie zuvor und erscheinen unterhalb der gruppierten Abschnitte.

## <Icon icon="mdi:alert-circle-outline" inline /> Häufige Probleme

### Agent taucht nicht auf

Wenn du `An agent with an existing ID was found. Removing the duplicate host.` siehst, hast du zwei Hosts mit derselben Server-ID.

Dozzle nutzt die Docker-API, um Informationen über Hosts zu sammeln. Jeder Agent braucht eine eindeutige Host-ID, die über Neustarts hinweg gleich bleibt, damit er korrekt identifiziert wird. Derzeit identifizieren Agents den Host entweder über die System-ID oder die Node-ID von Docker.

In einer Swarm-Umgebung wird dafür die Node-ID verwendet. Wenn dir auffällt, dass nicht alle Hosts sichtbar sind, kann das an doppelten Hosts mit derselben Host-ID liegen.

Um das zu beheben, solltest du `/var/lib/docker/engine-id` von deinem System entfernen und neu starten. Damit verschwinden Konflikte durch doppelte Host-IDs. Weitere Informationen und Tipps zur Fehlersuche findest du in der [FAQ](/de/guide/faq#i-am-seeing-duplicate-hosts-error-in-the-logs-how-do-i-fix-it).

## <Icon icon="mdi:cog-outline" inline /> Erweiterte Optionen

### Healthcheck einrichten

Du kannst für den Agent einen Healthcheck einrichten, ähnlich wie für die Haupt-Dozzle-Instanz. Im Agent-Modus prüft der Healthcheck die Verbindung des Agents zu Docker. Ist Docker nicht erreichbar, gilt der Agent als ungesund und wird in der Oberfläche nicht angezeigt.

Für den Healthcheck nutzt du den Unterbefehl `healthcheck`. Hier ein Beispiel:

```yml
services:
  dozzle-agent:
    image: amir20/dozzle:latest
    command: agent
    healthcheck:
      test: ["CMD", "/dozzle", "healthcheck"]
      interval: 5s
      retries: 5
      start_period: 5s
      start_interval: 5s
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    ports:
      - 7007:7007
```

### Namen des Agents ändern

Wie bei einer Dozzle-Instanz kannst du den Namen des Agents über die Umgebungsvariable `DOZZLE_HOSTNAME` ändern. Hier ein Beispiel:

::: code-group

```sh
docker run -v /var/run/docker.sock:/var/run/docker.sock -p 7007:7007 amir20/dozzle:latest agent --hostname my-special-name
```

```yaml [docker-compose.yml]
services:
  dozzle-agent:
    image: amir20/dozzle:latest
    command: agent
    environment:
      - DOZZLE_HOSTNAME=my-special-name
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    ports:
      - 7007:7007
```

:::

Damit heißt der Agent `my-special-name`, und dieser Name erscheint in der Oberfläche, wenn du dich mit dem Agent verbindest.

### Filter einrichten

Du kannst für den Agent Filter einrichten, um die Container einzuschränken, auf die er zugreifen kann. Diese Filter werden direkt an Docker weitergegeben und begrenzen, was Dozzle sehen kann.

```yaml
services:
  dozzle-agent:
    image: amir20/dozzle:latest
    command: agent
    environment:
      - DOZZLE_FILTER=label=color
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
```

Damit zeigt der Agent nur Container mit dem Label `color`. Beachte, dass diese Filter mit den Filtern der Oberfläche kombiniert werden, um die Container weiter einzugrenzen. Mehr zu den verschiedenen Filtertypen steht in der [Dokumentation zu Filtern](/de/guide/filters#ui-agents-and-user-filters).

### Eigene Zertifikate

Dozzle bringt ein selbstsigniertes Zertifikat mit, das beide Seiten einander vorzeigen, und jede Seite vertraut genau diesem einen Zertifikat und keinem anderen. Es ist in die Binary kompiliert, also in jeder Dozzle-Installation identisch, und jeder kann es aus dem öffentlichen Image auslesen.

Damit bekommst du Verschlüsselung, aber keine Authentifizierung. Ein Agent kann deinen Hub nicht von einem fremden unterscheiden. Das Einzige, was Fremde aus einem mit Standardwerten gestarteten Agent heraushält, ist, dass sie Port `7007` nicht erreichen.

> [!WARNING] Wann du ein eigenes Zertifikat brauchst
> Ist Port `7007` für irgendetwas erreichbar, das du nicht kontrollierst, etwa weil er auf einem Host mit öffentlicher IP veröffentlicht wird, erzeuge dein eigenes Paar. Jede Installation, die das tut, bekommt ein Geheimnis, das sonst niemand hat.

Mit `generate-certs` schreibst du ein eindeutiges Paar:

```sh
docker run --rm -v "$PWD":/out amir20/dozzle:latest generate-certs --cert-out /out/dozzle_cert.pem --key-out /out/dozzle_key.pem
```

Kopiere beide Dateien auf den Hub und auf jeden Agent. Dozzle sucht sie unter `/dozzle_cert.pem` und `/dozzle_key.pem`, es genügt also, sie dorthin zu mounten. Über die Flags `--cert` und `--key` oder die Umgebungsvariablen `DOZZLE_CERT` und `DOZZLE_KEY` kannst du sie auch anderswo ablegen.

Behandle den Schlüssel wie ein Passwort. Wer ihn hat, kann sich mit deinen Agents verbinden, und ein Agent weist jeden Hub ab, der etwas anderes vorzeigt. Hub und Agents müssen deshalb dasselbe Paar bekommen und zusammen neu gestartet werden.

Hier ein Beispiel mit den Standardpfaden:

```yml
services:
  agent:
    image: amir20/dozzle:latest
    command: agent
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    secrets:
      - source: cert
        target: /dozzle_cert.pem
      - source: key
        target: /dozzle_key.pem
    ports:
      - 7007:7007
secrets:
  cert:
    file: ./cert.pem
  key:
    file: ./key.pem
```

Oder mit eigenen Pfaden über Umgebungsvariablen:

```yml
services:
  agent:
    image: amir20/dozzle:latest
    command: agent
    environment:
      - DOZZLE_CERT=/certs/my-cert.pem
      - DOZZLE_KEY=/certs/my-key.pem
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./certs:/certs
    ports:
      - 7007:7007
```

Oder über Kommandozeilen-Flags:

::: code-group

```sh
docker run -v /var/run/docker.sock:/var/run/docker.sock -v ./certs:/certs -p 7007:7007 amir20/dozzle:latest agent --cert /certs/my-cert.pem --key /certs/my-key.pem
```

```yaml [docker-compose.yml]
services:
  agent:
    image: amir20/dozzle:latest
    command: agent --cert /certs/my-cert.pem --key /certs/my-key.pem
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./certs:/certs
    ports:
      - 7007:7007
```

:::

> [!TIP]
> Für Zertifikate sind Docker Secrets die bessere Wahl. Du kannst sie mit dem Befehl `docker secret create` anlegen oder wie im Beispiel oben über `docker-compose.yml`. Dieselben Zertifikate müssen auch der Dozzle-Instanz bereitgestellt werden, die sich mit dem Agent verbindet.

Damit werden Zertifikat und Schlüsseldatei in den Agent eingehängt. Der Agent nutzt diese Zertifikate für die Kommunikation. Dieselben Zertifikate müssen auch der Dozzle-Instanz bereitgestellt werden, die sich mit dem Agent verbindet.

Wenn du sie lieber mit openssl erzeugst als mit `generate-certs`:

```sh
$ openssl genpkey -algorithm Ed25519 -out key.pem
$ openssl req -new -key key.pem -out request.csr -subj "/C=US/ST=California/L=San Francisco/O=My Company"
$ openssl x509 -req -in request.csr -signkey key.pem -out cert.pem -days 365
```

## <Icon icon="mdi:compare-horizontal" inline /> Agents im Vergleich zu entfernten Verbindungen

Agents ähneln entfernten Verbindungen, haben aber einige Vorteile. Aus Gründen der Performance und Sicherheit sind Agents in der Regel die bessere Wahl. Hier ein Vergleich:

| Merkmal        | Agent                                     | Entfernte Verbindung                  |
| -------------- | ----------------------------------------- | ------------------------------------- |
| Performance    | Besser durch verteilte Last               | Schlechter in der Oberfläche          |
| Sicherheit     | Verschlüsselt, eigenes Zertifikat möglich | Unsicher oder Docker TLS              |
| Bedienung      | Ohne Aufwand einsatzbereit                | Erfordert Freigabe des Docker-Sockets |
| Berechtigungen | Voller Zugriff auf Docker                 | Über einen Proxy steuerbar            |
| Reconnect      | Verbindet sich automatisch neu            | Erfordert Neustart der Oberfläche     |
| Healthcheck    | Eingebauter Healthcheck                   | Kein Healthcheck                      |
| Filter         | Unterstützt Filter                        | Keine Unterstützung für Filter        |

Wenn du entfernte Verbindungen nutzen willst, sichere die Verbindung unbedingt mit Docker TLS oder einem Reverse Proxy ab.
