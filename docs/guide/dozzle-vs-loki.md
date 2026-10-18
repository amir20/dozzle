---
title: Dozzle vs Grafana Loki
description: Whether you need Grafana and Loki to read Docker logs, or whether Dozzle is enough, and how to use both.
---

# Dozzle vs Grafana Loki

Loki is a log store. A collector ships every log line into it, Loki keeps them for as long as you configure, and Grafana is where you query them. Dozzle stores nothing: it reads logs straight from Docker, the same ones `docker logs` shows, and streams them to your browser.

So the real question is whether you need to keep logs somewhere other than Docker.

## At a glance

|                          | Dozzle                                                        | Grafana + Loki                                       |
| ------------------------ | ------------------------------------------------------------- | ---------------------------------------------------- |
| What you run             | One container                                                 | Loki, Grafana, and a collector such as Grafana Alloy |
| Where logs live          | In Docker, as it already keeps them                           | Copied into Loki's own storage                       |
| How far back             | As far as Docker's log rotation keeps                         | As long as you set Loki's retention                  |
| Live tail                | The default view                                              | Available in Grafana Explore                         |
| Querying                 | Search, regular expressions, [SQL](/guide/sql-engine) on JSON | LogQL, including metrics computed from logs          |
| Dashboards               | Container CPU and memory                                      | Any dashboard you build in Grafana                   |
| Alerts                   | Log, metric and event [alerts](/guide/alerts-and-webhooks)    | Grafana alerting on LogQL queries                    |
| Logs from outside Docker | No                                                            | Yes: files, systemd journal, Kubernetes and more     |

## When to use Loki instead

Pick Loki when you need logs after Docker has rotated them away, when a container is removed and its logs must stay, or when you need to search across weeks of history. It is also the better fit if you already run Grafana and want logs next to your metrics, or need logs from things that are not containers.

Plan for the setup. Promtail, the collector most older guides use, reached end of life in March 2026, so new installs use Grafana Alloy or the Loki Docker driver.

## When Dozzle fits better

Pick Dozzle when you want to see what a container is doing now, or what it did in the last few hours or days, without running a log pipeline. There is no storage to size, no retention to tune and no query language to learn. It starts in seconds. See [Getting Started](/guide/getting-started).

How much history Dozzle can show depends on Docker's log rotation. To keep more, raise the limits in `/etc/docker/daemon.json`:

```json
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "50m",
    "max-file": "5"
  }
}
```

The new limits apply to containers created after Docker restarts.

## Use both

Many setups run Loki for history and Dozzle for everyday debugging. They don't get in each other's way: Dozzle reads from Docker, and Alloy reads from Docker. If you switch containers to the Loki Docker driver instead, keep Docker's [dual logging](https://docs.docker.com/engine/logging/dual-logging/) on (the default), so `docker logs`, and therefore Dozzle, can still read them. Docker then keeps that copy in a cache sized by `cache-max-size` and `cache-max-file` (5 files of 20 MB by default), not by the `max-size` above.
