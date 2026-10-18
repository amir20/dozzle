---
title: Dozzle vs docker logs
description: What Dozzle adds over docker logs and terminal tools, and when the terminal is all you need.
---

# Dozzle vs `docker logs`

Dozzle reads the same logs as `docker logs`. Nothing is copied or stored, so it never shows anything the command could not. What it changes is how you read them: in a browser, across containers and hosts, without a shell on the server.

## At a glance

|                     | Dozzle                                                                | `docker logs`                                         |
| ------------------- | --------------------------------------------------------------------- | ----------------------------------------------------- |
| Where you read them | A browser, from any device                                            | A shell on the Docker host                            |
| Several containers  | Any set of containers in one merged view, or a whole stack or service | One container. `docker compose logs` covers a project |
| Several hosts       | One UI for every host, through [agents](/guide/agent)                 | One host per shell, or a `DOCKER_HOST` per command    |
| Searching           | Search as you type, with regular expressions                          | Pipe to `grep`                                        |
| JSON logs           | Formatted and color coded, [SQL](/guide/sql-engine) over fields       | Raw lines, pipe to `jq`                               |
| Stack traces        | Multi-line entries grouped together                                   | One line at a time                                    |
| CPU and memory      | Charts next to the logs                                               | `docker stats` in another shell                       |
| Sharing             | A link to a container, behind [authentication](/guide/authentication) | Copy and paste                                        |
| Install             | One container                                                         | Already there                                         |

## When the terminal is enough

If you are already on the host and want the last hundred lines of one container, `docker logs --tail 100 -f <name>` is the quickest way. For a live view of every container in a terminal, [dtop](/guide/dtop) shows them in a `docker ps`-style table and opens Dozzle when you need the full logs.

## When Dozzle fits better

Use Dozzle when more than one person reads the logs, when you would rather not hand out SSH access to read them, or when you are following one request through several containers at once. It is also the easier way to read logs from a phone.

## Try it

```sh
docker run -d -v /var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle:latest
```

Then open `http://localhost:8080`. More options are in [Getting Started](/guide/getting-started).
