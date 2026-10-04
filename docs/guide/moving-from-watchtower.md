---
title: Moving from Watchtower
---

# Moving from Watchtower

Dozzle can do what Watchtower does: check your containers for newer images on a schedule and update them. Each update is watched, and rolled back if the new container fails to stay up. This page maps Watchtower's settings to Dozzle's.

## Turn it on

Dozzle needs actions on and `/data` on a volume:

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - dozzle-data:/data
    ports:
      - 8080:8080
    environment:
      DOZZLE_ENABLE_ACTIONS: true
      DOZZLE_AUTO_UPDATE: daily
      DOZZLE_AUTO_UPDATE_TIME: "04:00"
volumes:
  dozzle-data:
```

You can leave out the two `DOZZLE_AUTO_UPDATE` lines and set the schedule under **Settings → Updates** instead. Then stop Watchtower, so the two do not update the same containers.

## Settings

| Watchtower                                                         | Dozzle                                                                                                                                                                                   |
| ------------------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `--schedule` or `--interval`                                       | `DOZZLE_AUTO_UPDATE` (`daily`, or `weekly` on Sunday) and `DOZZLE_AUTO_UPDATE_TIME`. At most once a day                                                                                  |
| Every container (the default)                                      | **Which containers: Everything**                                                                                                                                                         |
| `--label-enable` with `com.centurylinklabs.watchtower.enable=true` | **Which containers: Dozzle and containers I pick** (the default) with `dev.dozzle.update: auto`                                                                                          |
| `com.centurylinklabs.watchtower.enable=false`                      | `dev.dozzle.update: off`, or **Manual** to keep the update check                                                                                                                         |
| `--cleanup`                                                        | Always on. The old untagged image is removed after an update, and one previous image is kept for rolling back                                                                            |
| `--monitor-only`                                                   | **Manual**: the container is checked and shown as an update, and never updated on its own                                                                                                |
| `--rolling-restart`                                                | Always: each host updates one container at a time                                                                                                                                        |
| `--notification-url`                                               | [Alerts and webhooks](/guide/alerts-and-webhooks) on container events, or [Dozzle Cloud](/guide/dozzle-cloud), which watches each update and tells you when a new version starts failing |
| `--run-once`                                                       | **Update** in the Updates drawer                                                                                                                                                         |
| Private registry credentials                                       | Not supported. Containers from a private registry are skipped                                                                                                                            |

## Labels

Watchtower's labels are not read. Replace `com.centurylinklabs.watchtower.enable` with `dev.dozzle.update`:

| `dev.dozzle.update` | What happens                                            |
| ------------------- | ------------------------------------------------------- |
| `auto`              | Updated on the schedule                                 |
| _(no label)_        | Manual: checked and shown as an update, which you apply |
| `off`               | Never checked, never updated                            |

A label wins over a choice made in the UI. Older Dozzle labels are still accepted: `dev.dozzle.auto-update=true` reads as `auto`, and `dev.dozzle.update-check=false` as `off`.

## What is different

- The old container is kept until the new one has stayed up, and healthy if it has a healthcheck. If not, the old one is put back.
- Unhealthy containers are skipped.
- A container someone [rolled back](/guide/actions#rolling-back) is not updated to the same image again until a newer one is published.
- Images pinned to a digest and images built locally are skipped, since there is nothing newer to compare against.
