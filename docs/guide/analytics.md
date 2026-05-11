---
title: Anonymous Analytics
---

# Data Collection of Analytics

Dozzle collects anonymous usage data via a lightweight beacon to help prioritize features and fixes. It is an open-source project with no funding, so this data is the primary signal for where to invest effort.

## What is Collected

At a high level, the beacon includes things like the Dozzle version, deployment mode (server, swarm, k8s, agent), which auth provider is enabled, a few feature flags, the Docker Engine version, and small counts (number of hosts, containers, filters). A random per-install ID is included for deduplication.

No log contents, container names, image names, IP addresses, or user identifiers are ever transmitted. The exact set of fields evolves over time — the authoritative source is [`types/beacon.go`](https://github.com/amir20/dozzle/blob/master/types/beacon.go), and the sender is [`internal/analytics/http_beacon.go`](https://github.com/amir20/dozzle/blob/master/internal/analytics/http_beacon.go).

## Where is Data Stored

Events are posted to `https://b.dozzle.dev/event`, a small Go service that writes events to a flat file on DigitalOcean for later processing.

## Opting Out

Pass `--no-analytics` or set `DOZZLE_NO_ANALYTICS=true`. No beacon requests will be made.

```yaml
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      DOZZLE_NO_ANALYTICS: "true"
```
