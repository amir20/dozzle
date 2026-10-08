---
title: Container Actions
---

# Container Actions

<Badge type="warning" text="Docker Only" />

Dozzle supports container actions, which allows you to `start`, `stop`, `restart`, `remove`, and `update` containers from the dropdown menu on the right next to the container stats. This feature is **disabled** by default and can be enabled by setting the environment variable `DOZZLE_ENABLE_ACTIONS` to `true`.

The `update` action pulls the latest image for the container and recreates it with the same configuration — useful for upgrading a container in place without editing its compose file. `update` only has a meaningful effect when the image uses a moving tag (e.g. `latest`, `stable`); a pinned tag will simply re-pull the same image.

> [!WARNING]
> `remove` deletes the container: data in its writable layer is lost, and its anonymous volumes are left behind, detached. `update` recreates the container and keeps every volume, anonymous ones included, and every bind mount. Only data written to the container's writable layer is lost.

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

## Updates {#auto-updating-containers}

Every update keeps the old container until the new one stays up, and puts it back if it does not. Checking for newer images, updating several containers at once, the auto-update schedule, cleanup and rolling back are covered in [Container Updates](/guide/updates).
