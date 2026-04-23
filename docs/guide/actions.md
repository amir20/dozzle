---
title: Container Actions
---

# Container Actions

<Badge type="warning" text="Docker Only" />

Dozzle supports container actions, which allows you to `start`, `stop`, `restart`, `remove`, and `update` containers from the dropdown menu on the right next to the container stats. This feature is **disabled** by default and can be enabled by setting the environment variable `DOZZLE_ENABLE_ACTIONS` to `true`.

The `update` action pulls the latest image for the container and recreates it with the same configuration — useful for upgrading a container in place without editing its compose file. `update` only has a meaningful effect when the image uses a moving tag (e.g. `latest`, `stable`); a pinned tag will simply re-pull the same image.

> [!WARNING]
> `remove` and `update` recreate the container. Data written to **anonymous volumes** or the container's writable layer will be lost. Named volumes and bind mounts are preserved.

> [!NOTE]
> Enabling actions also unlocks Compose [Deployments](/guide/deployments) when using [Dozzle Cloud](/guide/dozzle-cloud).

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
