---
title: Alerts & Webhooks
---

# Alerts & Webhooks

<Badge type="tip" text="New in v10" />

Dozzle v10 introduces a powerful alerting system that lets you monitor container logs and receive notifications when specific conditions are met. Alerts use customizable expressions to filter containers and log messages, and can send notifications to webhooks, Slack, Discord, ntfy, or [Dozzle Cloud](/guide/dozzle-cloud).

## How It Works

Alerts are configured with two expressions:

1. **Container filter** — selects which containers to monitor
2. **Log filter** — defines which log messages trigger the alert

When a log entry matches both filters, Dozzle sends a notification to the configured destination.

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

| Variable                  | Description                 |
| ------------------------- | --------------------------- |
| `{{.Container.Name}}`     | Container name              |
| `{{.Container.Image}}`    | Container image             |
| `{{.Container.HostName}}` | Docker host name            |
| `{{.Container.State}}`    | Container state             |
| `{{.Log.Message}}`        | Log message content         |
| `{{.Log.Level}}`          | Log level                   |
| `{{.Log.Timestamp}}`      | Log timestamp               |
| `{{.Log.Stream}}`         | Stream type (stdout/stderr) |
| `{{.Subscription.Name}}`  | Alert rule name             |

</div>

> [!TIP]
> Use the **Test** button to verify your webhook is working before saving.

### Dozzle Cloud

You can also send alerts to [Dozzle Cloud](/guide/dozzle-cloud) for centralized monitoring across multiple Dozzle instances. See the [Dozzle Cloud guide](/guide/dozzle-cloud) for more details.

## Creating an Alert

Navigate to the **Notifications** page and click **Add Alert**. You'll need to configure:

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

### Examples

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

> [!NOTE]
> The alert editor includes autocomplete and real-time validation. You can preview matched containers and logs before saving.

## Managing Alerts

From the Notifications page, you can:

- **Enable/disable** alerts without deleting them
- **Edit** alert expressions and destinations
- **View statistics** including trigger count, matched containers, and last triggered time
- **Delete** alerts that are no longer needed
