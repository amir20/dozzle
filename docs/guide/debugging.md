---
title: Debugging
---

# Debugging with Logs

By default Dozzle logs at `info` level, which is intentionally quiet. When something isn't working, turn up the verbosity using the `--level` flag or the `DOZZLE_LEVEL` environment variable.

| Level   | When to use                                                                  |
| ------- | ---------------------------------------------------------------------------- |
| `info`  | Default. Startup details, errors, and warnings.                              |
| `debug` | Request-level diagnostics, auth decisions, agent connections, config dump.   |
| `trace` | Everything. Individual log events, beacon payloads, gRPC frames. Very noisy. |

Dozzle writes all logs to `stdout`, so `docker logs dozzle` is the right place to read them.

```yaml
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    ports:
      - 8080:8080
    environment:
      DOZZLE_LEVEL: debug
```

## Reporting a Bug

If you think you've hit a bug, please open an issue at [github.com/amir20/dozzle/issues](https://github.com/amir20/dozzle/issues). Include:

- Dozzle version (visible in the UI footer or `dozzle --version`)
- Deployment mode: server, swarm, k8s, or agent
- Docker or Kubernetes version
- Relevant `debug`- or `trace`-level log output
- Steps to reproduce, ideally with a minimal `docker-compose.yml`

The more context in the initial report, the faster it can be triaged.
