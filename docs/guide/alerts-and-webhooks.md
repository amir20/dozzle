---
title: Alerts & Webhooks
---

# Alerts & Webhooks

<Badge type="tip" text="New in v10" />

Dozzle v10 introduces a powerful alerting system that lets you monitor container logs, resource metrics, and lifecycle events, and receive notifications when specific conditions are met. Alerts use customizable expressions to filter containers and trigger conditions, and can send notifications to webhooks, Slack, Discord, ntfy, or [Dozzle Cloud](/guide/dozzle-cloud).

## Alert Types

Dozzle supports three kinds of alerts, all configured the same way from the **Notifications** page:

| Type                         | Triggers on                            | Example use case                |
| ---------------------------- | -------------------------------------- | ------------------------------- |
| [**Log**](#log-alerts)       | A log message matching a pattern       | 5xx errors, stack traces        |
| [**Metric**](#metric-alerts) | CPU / memory crossing a threshold      | Container exceeding 90% CPU     |
| [**Event**](#event-alerts)   | Container lifecycle events from Docker | OOM kills, unhealthy containers |

Each alert pairs a **container expression** (which containers to watch) with a **trigger expression** (the condition to fire on).

> [!IMPORTANT]
> Alert and destination configurations are stored in the `/data` directory. You must mount this directory as a volume to persist your notification settings across container restarts.

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

## Setting Up a Destination

Before creating alerts, you need to configure at least one notification destination. Navigate to the **Notifications** page in Dozzle and click **Add Destination**.

### Webhook

Webhooks send an HTTP POST request to a URL of your choice. Dozzle includes built-in payload templates for popular services:

- **Slack** — formatted with blocks and markdown
- **Discord** — formatted for Discord webhook API
- **ntfy** — formatted for [ntfy.sh](https://ntfy.sh) push notifications
- **Custom** — generic JSON payload you can customize

You can also write your own payload template using Go's `text/template` syntax. The following variables are available:

<div v-pre>

| Variable                  | Description                            |
| ------------------------- | -------------------------------------- |
| `{{.Detail}}`             | Summary (log message or metric values) |
| `{{.Container.Name}}`     | Container name                         |
| `{{.Container.Image}}`    | Container image                        |
| `{{.Container.HostName}}` | Docker host name                       |
| `{{.Container.State}}`    | Container state                        |
| `{{.Log.Message}}`        | Log message content                    |
| `{{.Log.Level}}`          | Log level                              |
| `{{.Log.Timestamp}}`      | Log timestamp                          |
| `{{.Log.Stream}}`         | Stream type (stdout/stderr)            |
| `{{.Stat.CPUPercent}}`    | CPU usage percentage                   |
| `{{.Stat.MemoryPercent}}` | Memory usage percentage                |
| `{{.Stat.MemoryUsage}}`   | Memory usage in bytes                  |
| `{{.Subscription.Name}}`  | Alert rule name                        |

</div>

> [!TIP]
> Use the **Test** button to verify your webhook is working before saving.

### Dozzle Cloud

You can also send alerts to [Dozzle Cloud](/guide/dozzle-cloud) for centralized monitoring across multiple Dozzle instances. See the [Dozzle Cloud guide](/guide/dozzle-cloud) for more details.

## Creating an Alert

Navigate to the **Notifications** page and click **Add Alert**. Every alert has a **container expression** plus one of a **log**, **metric**, or **event** trigger expression.

### Container Expression

The container expression selects which containers to monitor. Available properties:

| Property   | Type   | Example                         |
| ---------- | ------ | ------------------------------- |
| `name`     | string | `name contains "api"`           |
| `image`    | string | `image == "nginx:latest"`       |
| `state`    | string | `state == "running"`            |
| `health`   | string | `health == "unhealthy"`         |
| `hostName` | string | `hostName == "prod-host"`       |
| `labels`   | map    | `labels["env"] == "production"` |

You can combine conditions with `&&` (AND), `||` (OR), and `!` (NOT):

```
name contains "api" && labels["env"] == "production"
```

## Log Alerts

### Log Expression

The log expression filters which log messages trigger the alert. Available properties:

| Property  | Type       | Example                    |
| --------- | ---------- | -------------------------- |
| `message` | string/map | `message contains "error"` |
| `level`   | string     | `level == "error"`         |
| `stream`  | string     | `stream == "stderr"`       |
| `type`    | string     | `type == "complex"`        |

For JSON logs, you can access nested fields using dot notation:

```
message.status >= 500 && message.path contains "/api"
```

Supported string operators include `contains`, `startsWith`, `endsWith`, and `matches` (regex).

### Log Examples

**Alert on all errors from production containers:**

```
Container: labels["env"] == "production"
Log:       level == "error"
```

**Alert on HTTP 5xx errors from API containers:**

```
Container: name contains "api"
Log:       message.status >= 500
```

**Alert on any stderr output from a specific image:**

```
Container: image startsWith "myapp/"
Log:       stream == "stderr"
```

**Alert on slow API responses from production:**

```
Container: name contains "api" && labels["env"] == "production"
Log:       message.duration > 5000 && message.path contains "/api"
```

**Alert on authentication failures using regex:**

```
Container: name contains "auth" || name contains "gateway"
Log:       message matches "(?i)(unauthorized|forbidden|invalid token)"
```

> [!NOTE]
> The alert editor includes autocomplete and real-time validation. You can preview matched containers and logs before saving.

## Metric Alerts

Metric alerts fire when a container's CPU or memory usage crosses a threshold. The trigger expression is evaluated against a smoothed average of stats sampled over a rolling window, which avoids false alarms from short spikes.

### Metric Expression

Available properties:

| Property      | Type   | Description                                     |
| ------------- | ------ | ----------------------------------------------- |
| `cpu`         | number | CPU usage percentage (0–100, per core-adjusted) |
| `memory`      | number | Memory usage percentage (0–100)                 |
| `memoryUsage` | number | Memory usage in bytes                           |

### Cooldown & Sample Window

- **Sample window** — how many seconds of stats are averaged before the expression is evaluated. Longer windows smooth out spikes; shorter windows react faster.
- **Cooldown** — minimum seconds between consecutive triggers for the same container. Prevents alert floods when a container stays above threshold.

### Metric Examples

**High CPU on production containers:**

```
Container: labels["env"] == "production"
Metric:    cpu > 90
```

**Memory pressure on a specific service:**

```
Container: name contains "api"
Metric:    memory > 85
```

**Absolute memory usage (1 GiB):**

```
Container: name == "postgres"
Metric:    memoryUsage > 1073741824
```

## Event Alerts

Event alerts fire on Docker container lifecycle events — useful for catching crashes, OOM kills, and health status changes without parsing logs.

### Event Expression

Available properties:

| Property     | Type   | Description                                         |
| ------------ | ------ | --------------------------------------------------- |
| `name`       | string | Event name (see below)                              |
| `actorId`    | string | Docker actor ID (usually the container ID)          |
| `attributes` | map    | Event attributes from Docker (varies by event type) |
| `timestamp`  | time   | When the event occurred                             |

Common Docker event names include `start`, `stop`, `die`, `kill`, `oom`, `restart`, `destroy`, and `health_status`.

### Event Examples

**Alert when any production container dies:**

```
Container: labels["env"] == "production"
Event:     name == "die"
```

**Alert on OOM kills:**

```
Container: true
Event:     name == "oom"
```

**Alert when a container becomes unhealthy:**

```
Container: true
Event:     name == "health_status" && attributes["health_status"] == "unhealthy"
```

**Alert on unexpected exits (non-zero exit code):**

```
Container: name contains "worker"
Event:     name == "die" && attributes["exitCode"] != "0"
```

## Managing Alerts

From the Notifications page, you can:

- **Enable/disable** alerts without deleting them
- **Edit** alert expressions and destinations
- **View statistics** including trigger count, matched containers, and last triggered time
- **Delete** alerts that are no longer needed
