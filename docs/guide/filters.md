---
title: Filter
---

# Filtering Containers

Dozzle supports conditional filtering similar to Docker's [--filter](https://docs.docker.com/reference/cli/docker/container/ls/#filter) with `DOZZLE_FILTER` or `--filter`. Filters are passed directly to Docker to limit what Dozzle can see. For example, filtering by label is supported with `--filter "label=color"`, which is similar to `docker ps` command with `docker ps --filter "label=color"`.

::: code-group

```sh
docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --filter label=color
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
      DOZZLE_FILTER: label=color
```

:::

Common filters are `name` or `label` to limit Dozzle's access to containers.
