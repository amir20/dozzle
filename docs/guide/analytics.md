---
title: Anonymous Analytics
---

# Data Collection of Analytics

Dozzle collects anonymous usage data via a lightweight beacon to help prioritize features and fixes. It is an open-source project with no funding, so this data is the primary signal for where to invest effort.

## What is Collected

Dozzle sends a beacon when it starts, and another each time someone opens the UI. Together they include:

- the Dozzle version, the deployment mode (server, swarm, k8s, agent) and the Docker Engine version
- which auth provider is enabled, and whether actions and shell are turned on
- small counts: hosts, agents, running containers and filters
- the browser's user agent string, on the UI beacon only
- the Docker Engine's ID, so the same install is not counted twice

No log contents, container names, image names, hostnames or user identifiers are ever transmitted. The exact set of fields evolves over time. The authoritative source is [`types/beacon.go`](https://github.com/amir20/dozzle/blob/master/types/beacon.go), and the sender is [`internal/analytics/http_beacon.go`](https://github.com/amir20/dozzle/blob/master/internal/analytics/http_beacon.go).

## Where is Data Stored

Beacons are posted to `https://b.dozzle.dev/event` and received by [drain](https://github.com/amir20/drain), an open-source Go service that writes them to a database and to Parquet files for analysis. drain does not keep the IP address a beacon came from, and the data is not shared with any third party.

## Opting Out

Pass `--no-analytics` or set `DOZZLE_NO_ANALYTICS=true`. No beacon requests will be made.

```yaml
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      DOZZLE_NO_ANALYTICS: "true"
```
