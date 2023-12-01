---
title: Container Actions
---

# Using Container Actions

Dozzle now supports **Container Actions**, which allows you to `start`, `stop` and `restart` container from within the UI in the dropdown menu.

<img title="Container Actions" alt="Container Acions Menu UI" width="250" src="/.vitepress/theme/media/dozzle-ui-actions.png">

This feature is **disabled** by default, which can be enabled by setting environment variable`DOZZLE_ENABLE_ACTIONS` to `true`

::: code-group

```sh
docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --enable-actions
```

```yaml [docker-compose.yml]
version: "3"
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    ports:
      - 8080:8080
    environment:
      DOZZLE_ENABLE_ACTIONS: true
```

:::
