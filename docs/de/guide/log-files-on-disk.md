---
title: Logdateien auf der Festplatte verfolgen
sourceHash: e6e23f2438f9
---

# Logdateien auf der Festplatte verfolgen

Manche Container schreiben ihre Logs in Dateien statt nach `stdout` oder `stderr`. Dozzle kann nur lesen, was Docker selbst erfasst, also `stdout` und `stderr`, genau wie `docker logs`. Dateien innerhalb eines Containers sind für andere Container nicht sichtbar, Dozzle kommt also nicht an sie heran.

## Lieber in Streams loggen

Am besten hörst du auf, in Dateien zu schreiben. Die meisten Anwendungen haben eine Option, um auf die Konsole zu loggen, und die [Twelve-Factor-App](https://12factor.net/logs) erklärt, warum das der richtige Standard ist.

Lässt sich die Anwendung nicht umkonfigurieren, verlinke die Logdatei in deinem `Dockerfile` per Symlink auf stdout des Containers. Genau das macht das offizielle nginx-Image:

```dockerfile
RUN ln -sf /dev/stdout /var/log/nginx/access.log \
    && ln -sf /dev/stderr /var/log/nginx/error.log
```

## Eine Datei mit einem Sidecar verfolgen

Wenn beides nicht geht, starte einen kleinen Alpine-Container, der die Datei verfolgt und Docker die Ausgabe erfassen lässt. Dozzle zeigt sie dann wie bei jedem anderen Container an.

::: code-group

```sh [docker run]
docker run -d \
  --name system-log \
  --label dev.dozzle.name=system-log \
  --network none \
  --restart unless-stopped \
  --log-opt max-size=10m --log-opt max-file=3 \
  -v /var/log:/logs:ro \
  alpine tail -n 1000 -F /logs/system.log
```

```yaml [docker-compose.yml]
services:
  system-log:
    container_name: system-log
    image: alpine
    volumes:
      - /var/log:/logs:ro
    command:
      - tail
      - -n
      - "1000"
      - -F
      - /logs/system.log
    labels:
      dev.dozzle.name: system-log
    logging:
      options:
        max-size: 10m
        max-file: "3"
    network_mode: none
    restart: unless-stopped
```

:::

Die Compose-Variante ist praktisch, wenn der Log-Stream einen Server-Neustart überstehen soll. In Tests brauchte Alpine etwa `~50KB` Speicher.

### Warum `-F` und nicht `-f`

`tail -f` folgt dem offenen Dateihandle. Wird die Datei rotiert, zeigt das Handle auf die alte, umbenannte Datei und der Stream verstummt. `tail -F` folgt dem Pfad und öffnet die Datei nach einer Rotation erneut, läuft also weiter.

Aus demselben Grund solltest du das **Verzeichnis** einbinden statt der Datei. Ein Bind-Mount einer einzelnen Datei hängt an deren Inode, eine Rotation auf dem Host ersetzt die Datei und der Container schaut weiter auf die alte, selbst mit `-F`.

### Verlauf vorbefüllen

Docker speichert nur, was der Container seit dem Start ausgegeben hat, ein Neustart des Sidecars verwirft also alles, was Dozzle hatte. `-n 1000` gibt beim Start die letzten 1000 Zeilen aus, damit die Ansicht nicht leer ist.

### Mehrere Dateien

Bekommt `tail` mehr als eine Datei, stellt es jedem Block den Dateinamen voran. Globs brauchen eine Shell, da das Image keinen Entrypoint hat, der sie auflöst:

```sh
docker run -d -v /var/log:/logs:ro alpine sh -c 'tail -n 1000 -F /logs/*.log'
```

Das Label `dev.dozzle.name` oben gibt dem Sidecar einen lesbaren Namen in der Oberfläche. Mehr dazu unter [Container-Namen](/de/guide/container-names).
