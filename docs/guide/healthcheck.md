---
title: Healthcheck
---

# Enabling Healthcheck

Dozzle ships a built-in `dozzle healthcheck` subcommand. It is not wired into the image by default because it adds a small amount of CPU overhead. Enable it from your compose file:

```yaml
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    ports:
      - 8080:8080
    healthcheck:
      test: ["CMD", "/dozzle", "healthcheck"]
      interval: 3s
      timeout: 30s
      retries: 5
      start_period: 30s
```

## What It Checks

When running as a server, `dozzle healthcheck` sends an HTTP `GET` to its own `/healthcheck` endpoint. The endpoint pings every **local** Docker client (up to 3s per client) and returns:

- `200 OK` — at least one local Docker client responded, **or** no local clients are configured but at least one remote agent host is known.
- `500 Internal Server Error` — all local clients failed to ping and no agent hosts are known.

Remote agents are intentionally **not** part of the server's healthcheck — an unreachable agent should not mark the main Dozzle process unhealthy. Each agent can expose its own healthcheck; see [Agent healthcheck](/guide/agent#setting-up-healthcheck).

## Exit Codes

- `0` — healthy (HTTP 200)
- non-zero — unhealthy, network error, or non-200 response. The failing URL and status are logged to stdout.

The command honors `--addr` and `--base`, so it works with custom ports and base paths without extra configuration.

> [!WARNING]
> The `healthcheck` command does not work with the `--health-cmd` flag due to a bug in Docker. Use the `healthcheck` block in `docker-compose.yml` as shown above. See [docker/cli#3719](https://github.com/docker/cli/issues/3719) for details.
