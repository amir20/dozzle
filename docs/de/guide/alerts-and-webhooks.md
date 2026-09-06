---
title: Alarme & Webhooks
sourceHash: daf372975955
---

# Alarme & Webhooks

Dozzle hat ein Alarmsystem, mit dem du Container-Logs, Ressourcenmetriken und Lebenszyklus-Ereignisse überwachst und benachrichtigt wirst, sobald bestimmte Bedingungen zutreffen. Alarme filtern Container und Auslösebedingungen über anpassbare Ausdrücke und können Benachrichtigungen an Webhooks, Slack, Discord, ntfy oder [Dozzle Cloud](/de/guide/dozzle-cloud) schicken.

## <Icon icon="mdi:format-list-bulleted-type" inline /> Alarmtypen

Dozzle unterstützt drei Arten von Alarmen, die alle gleich auf der Seite **Benachrichtigungen** konfiguriert werden:

| Typ                           | Löst aus bei                                     | Beispielhafter Anwendungsfall  |
| ----------------------------- | ------------------------------------------------ | ------------------------------ |
| [**Log**](#log-alerts)        | Einer Logmeldung, die auf ein Muster passt       | 5xx-Fehler, Stacktraces        |
| [**Metrik**](#metric-alerts)  | CPU / Speicher überschreitet einen Schwellenwert | Container über 90 % CPU        |
| [**Ereignis**](#event-alerts) | Lebenszyklus-Ereignissen von Docker              | OOM-Kills, ungesunde Container |

Jeder Alarm kombiniert einen **Container-Ausdruck** (welche Container überwacht werden) mit einem **Auslöse-Ausdruck** (die Bedingung, die den Alarm feuern lässt).

> [!IMPORTANT]
> Die Konfiguration von Alarmen und Zielen liegt im Verzeichnis `/data`. Du musst dieses Verzeichnis als Volume einhängen, damit deine Benachrichtigungseinstellungen Neustarts des Containers überleben.

::: code-group

```sh
docker run -v /var/run/docker.sock:/var/run/docker.sock -v /path/to/data:/data -p 8080:8080 amir20/dozzle:latest
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /path/to/data:/data
    ports:
      - 8080:8080
```

:::

## <Icon icon="mdi:send-outline" inline /> Ein Ziel einrichten

Bevor du Alarme anlegst, musst du mindestens ein Benachrichtigungsziel konfigurieren. Geh in Dozzle auf die Seite **Benachrichtigungen** und klicke auf **Ziel hinzufügen**.

### Webhook

Webhooks schicken eine HTTP-POST-Anfrage an eine URL deiner Wahl. Dozzle bringt fertige Payload-Templates für verbreitete Dienste mit:

- **Slack** — formatiert mit Blocks und Markdown
- **Discord** — formatiert für die Discord-Webhook-API
- **ntfy** — formatiert für Push-Benachrichtigungen von [ntfy.sh](https://ntfy.sh)
- **Custom** — generisches JSON-Payload, das du anpassen kannst

Du kannst auch ein eigenes Payload-Template mit der `text/template`-Syntax von Go schreiben. Diese Variablen stehen zur Verfügung:

<div v-pre>

| Variable                  | Beschreibung                                  |
| ------------------------- | --------------------------------------------- |
| `{{.Detail}}`             | Zusammenfassung (Logmeldung oder Metrikwerte) |
| `{{.Container.Name}}`     | Name des Containers                           |
| `{{.Container.Image}}`    | Image des Containers                          |
| `{{.Container.HostName}}` | Name des Docker-Hosts                         |
| `{{.Container.State}}`    | Status des Containers                         |
| `{{.Log.Message}}`        | Inhalt der Logmeldung                         |
| `{{.Log.Level}}`          | Loglevel                                      |
| `{{.Log.Timestamp}}`      | Zeitstempel des Logs                          |
| `{{.Log.Stream}}`         | Art des Streams (stdout/stderr)               |
| `{{.Stat.CPUPercent}}`    | CPU-Auslastung in Prozent                     |
| `{{.Stat.MemoryPercent}}` | Speicherauslastung in Prozent                 |
| `{{.Stat.MemoryUsage}}`   | Speicherverbrauch in Bytes                    |
| `{{.Subscription.Name}}`  | Name der Alarmregel                           |

</div>

> [!TIP]
> Nutze den Button **Test**, um deinen Webhook vor dem Speichern zu prüfen.

### Dozzle Cloud

Du kannst Alarme auch an [Dozzle Cloud](/de/guide/dozzle-cloud) schicken, um mehrere Dozzle-Instanzen zentral zu überwachen. Mehr Details stehen im [Leitfaden zu Dozzle Cloud](/de/guide/dozzle-cloud).

## <Icon icon="mdi:plus-circle-outline" inline /> Einen Alarm anlegen

Geh auf die Seite **Benachrichtigungen** und klicke auf **Alarm hinzufügen**. Jeder Alarm hat einen **Container-Ausdruck** plus einen Auslöse-Ausdruck für **Log**, **Metrik** oder **Ereignis**.

### Container-Ausdruck

Der Container-Ausdruck wählt aus, welche Container überwacht werden. Verfügbare Eigenschaften:

| Eigenschaft | Typ    | Beispiel                        |
| ----------- | ------ | ------------------------------- |
| `name`      | string | `name contains "api"`           |
| `image`     | string | `image == "nginx:latest"`       |
| `state`     | string | `state == "running"`            |
| `health`    | string | `health == "unhealthy"`         |
| `hostName`  | string | `hostName == "prod-host"`       |
| `labels`    | map    | `labels["env"] == "production"` |

Bedingungen kannst du mit `&&` (UND), `||` (ODER) und `!` (NICHT) kombinieren:

```
name contains "api" && labels["env"] == "production"
```

## <Icon icon="mdi:text-search" inline /> Log-Alarme

### Log-Ausdruck

Der Log-Ausdruck filtert, welche Logmeldungen den Alarm auslösen. Verfügbare Eigenschaften:

| Eigenschaft | Typ        | Beispiel                   |
| ----------- | ---------- | -------------------------- |
| `message`   | string/map | `message contains "error"` |
| `level`     | string     | `level == "error"`         |
| `stream`    | string     | `stream == "stderr"`       |
| `type`      | string     | `type == "complex"`        |

Bei JSON-Logs kannst du über die Punktnotation auf verschachtelte Felder zugreifen:

```
message.status >= 500 && message.path contains "/api"
```

Zu den unterstützten String-Operatoren gehören `contains`, `startsWith`, `endsWith` und `matches` (Regex).

### Log-Beispiele

**Alarm bei allen Fehlern aus Produktionscontainern:**

```
Container: labels["env"] == "production"
Log:       level == "error"
```

**Alarm bei HTTP-5xx-Fehlern aus API-Containern:**

```
Container: name contains "api"
Log:       message.status >= 500
```

**Alarm bei jeder stderr-Ausgabe eines bestimmten Images:**

```
Container: image startsWith "myapp/"
Log:       stream == "stderr"
```

**Alarm bei langsamen API-Antworten in der Produktion:**

```
Container: name contains "api" && labels["env"] == "production"
Log:       message.duration > 5000 && message.path contains "/api"
```

**Alarm bei fehlgeschlagenen Anmeldungen per Regex:**

```
Container: name contains "auth" || name contains "gateway"
Log:       message matches "(?i)(unauthorized|forbidden|invalid token)"
```

> [!NOTE]
> Der Alarm-Editor bietet Autovervollständigung und prüft die Eingabe direkt. Du kannst vor dem Speichern eine Vorschau der passenden Container und Logs sehen.

## <Icon icon="mdi:chart-line" inline /> Metrik-Alarme

Metrik-Alarme feuern, wenn die CPU- oder Speicherauslastung eines Containers einen Schwellenwert überschreitet. Der Auslöse-Ausdruck wird gegen einen geglätteten Durchschnitt der Messwerte aus einem gleitenden Zeitfenster ausgewertet, was Fehlalarme durch kurze Spitzen verhindert.

### Metrik-Ausdruck

Verfügbare Eigenschaften:

| Eigenschaft   | Typ    | Beschreibung                                             |
| ------------- | ------ | -------------------------------------------------------- |
| `cpu`         | number | CPU-Auslastung in Prozent (0–100), wie in der Oberfläche |
| `memory`      | number | Speicherauslastung in Prozent (0–100)                    |
| `memoryUsage` | number | Speicherverbrauch in Bytes                               |

### Abklingzeit & Messfenster

- **Messfenster** — wie viele Sekunden an Messwerten gemittelt werden, bevor der Ausdruck ausgewertet wird. Längere Fenster glätten Spitzen, kürzere reagieren schneller.
- **Abklingzeit** — Mindestabstand in Sekunden zwischen zwei Auslösungen für denselben Container. Verhindert eine Flut von Alarmen, wenn ein Container dauerhaft über dem Schwellenwert liegt.

### Metrik-Beispiele

**Hohe CPU-Last bei Produktionscontainern:**

```
Container: labels["env"] == "production"
Metric:    cpu > 90
```

**Speicherdruck bei einem bestimmten Dienst:**

```
Container: name contains "api"
Metric:    memory > 85
```

**Absoluter Speicherverbrauch (1 GiB):**

```
Container: name == "postgres"
Metric:    memoryUsage > 1073741824
```

## <Icon icon="mdi:bell-outline" inline /> Ereignis-Alarme

Ereignis-Alarme feuern bei Lebenszyklus-Ereignissen von Docker-Containern. Praktisch, um Abstürze, OOM-Kills und Änderungen des Gesundheitszustands zu erkennen, ohne Logs zu parsen.

### Ereignis-Ausdruck

Verfügbare Eigenschaften:

| Eigenschaft  | Typ    | Beschreibung                                                   |
| ------------ | ------ | -------------------------------------------------------------- |
| `name`       | string | Name des Ereignisses (siehe unten)                             |
| `actorId`    | string | Docker-Actor-ID (normalerweise die Container-ID)               |
| `attributes` | map    | Ereignisattribute von Docker (je nach Ereignistyp verschieden) |
| `timestamp`  | time   | Zeitpunkt des Ereignisses                                      |

Gängige Docker-Ereignisnamen sind `start`, `stop`, `die`, `kill`, `oom`, `restart`, `destroy` und `health_status`.

Bei `health_status`-Ereignissen stellt Dozzle den aktuellen Zustand als `attributes["healthStatus"]` bereit (`healthy` oder `unhealthy`).

### Ereignis-Beispiele

**Alarm, wenn ein Produktionscontainer stirbt:**

```
Container: labels["env"] == "production"
Event:     name == "die"
```

**Alarm bei OOM-Kills:**

```
Container: true
Event:     name == "oom"
```

**Alarm, wenn ein Container ungesund wird:**

```
Container: true
Event:     name == "health_status" && attributes["healthStatus"] == "unhealthy"
```

**Alarm bei unerwarteten Beendigungen (saubere und geordnete Shutdowns ausgenommen):**

Die Exit-Codes 0 (Erfolg), 130 (SIGINT), 143 (SIGTERM) und 137 (SIGKILL) treten bei `docker stop`, Strg+C und Update-Zyklen auf und werden deshalb ausgeschlossen, um Rauschen zu vermeiden. Echte Fehler-Exits (1, 2, 125, ...) lösen weiterhin einen Alarm aus.

```
Container: name contains "worker"
Event:     name == "die" && !(attributes["exitCode"] in ["0", "130", "143", "137"])
```

## <Icon icon="mdi:cog-outline" inline /> Alarme verwalten

Auf der Seite Benachrichtigungen kannst du:

- Alarme **aktivieren/deaktivieren**, ohne sie zu löschen
- Ausdrücke und Ziele von Alarmen **bearbeiten**
- **Statistiken ansehen**, darunter Anzahl der Auslösungen, passende Container und Zeitpunkt der letzten Auslösung
- Nicht mehr benötigte Alarme **löschen**
