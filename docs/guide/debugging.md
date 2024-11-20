---
title: Debugging
---

# Debugging with Logs

By default, Dozzle does not output a lot of logs. However, this can be changed with the `--level` flag. The default value is `info` which only prints limited logs. You can use `debug` or `trace` which will show details about memory, configuration and other stats. `DOZZLE_LEVEL` can be used in compose configurations. Below is an example of using `docker-compose.yml` file to enable `debug` level.

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
