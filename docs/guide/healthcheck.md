---
title: Healthcheck
---

# Adding healthcheck

Dozzle has internal support for healthcheck using `dozzle healthcheck` command. It is not enabled by default as it adds extra CPU usage. To use `healthcheck` you need to configure it. Below is an example that checks the health of Dozzle every 3 seconds. 

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
