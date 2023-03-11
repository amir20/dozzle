---
title: Healthcheck
---

# Adding healthcheck

Dozzle doesn't enable healthcheck by default as it adds extra CPU usage. `healthcheck` can be enabled manually.

```yaml
version: "3"
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
