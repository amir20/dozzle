---
title: Dozzle vs Portainer
description: How Dozzle and Portainer differ for reading Docker logs, and how to run them side by side.
---

# Dozzle vs Portainer

Portainer is built to manage containers: stacks, volumes, networks, images and users. Dozzle is built to read their logs. Portainer has a log viewer, and Dozzle can start, stop and update containers, but each one is centered on a different job. Many people run both.

## At a glance

|                    | Dozzle                                                                                      | Portainer                                                 |
| ------------------ | ------------------------------------------------------------------------------------------- | --------------------------------------------------------- |
| Main job           | Reading logs                                                                                | Managing containers, stacks, volumes, networks and images |
| Live logs          | Streams as lines arrive                                                                     | Auto refresh                                              |
| Several containers | A stack, a service or a custom group in one merged view                                     | One container at a time                                   |
| JSON logs          | Detected, formatted and color coded. [SQL](/guide/sql-engine) over JSON fields              | Shown as text                                             |
| Search             | Search and filter, including regular expressions                                            | Search and filter by text                                 |
| CPU and memory     | Live charts with recent history                                                             | Live stats                                                |
| Alerts             | On log lines, CPU or memory, and container events. See [Alerts](/guide/alerts-and-webhooks) | Not in the log viewer                                     |
| Image updates      | Checks for new images and updates on a schedule. See [Updates](/guide/updates)              | Pull and redeploy by hand                                 |
| Several hosts      | [Agents](/guide/agent), [Swarm](/guide/swarm-mode), [Kubernetes](/guide/k8s)                | Environments                                              |
| Setup              | One container, plus an optional setup wizard on first start                                 | One container, plus an admin account on first start       |

## When to use Portainer instead

Pick Portainer if you want to create and edit stacks from a browser, manage volumes, networks and images, or give a team access to some environments and not others. Dozzle does not try to do any of that.

## When Dozzle fits better

Pick Dozzle when the question is "what is this container doing right now?" It opens straight to logs, follows several containers at once, and makes JSON readable without any setup. It is also read-only by default: [actions](/guide/actions) and [shell access](/guide/shell) are off until you turn them on.

## Run both

The two do not conflict. Both read the Docker socket and run on different ports:

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - dozzle_data:/data
    ports:
      - 8080:8080
  portainer:
    image: portainer/portainer-ce:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - portainer-data:/data
    ports:
      - 9443:9443
volumes:
  dozzle_data:
  portainer-data:
```

To jump from Portainer to a container's logs in Dozzle, see [linking to a container by name](/guide/faq#we-have-tools-that-use-dozzle-when-a-new-container-is-created-how-can-i-get-a-direct-link-to-a-container-by-name).
