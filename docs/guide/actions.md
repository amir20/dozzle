---
title: Container Actions
---

# Using Container Actions

Dozzle supports container actions, which allows you to `start`, `stop` and `restart` containers from the dropdown menu on the right next to the container stats. This feature is **disabled** by default and can be enabled by setting the environment variable `DOZZLE_ENABLE_ACTIONS` to `true`.

::: code-group

```sh
docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --enable-actions
```

```yaml [docker-compose.yml]
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
