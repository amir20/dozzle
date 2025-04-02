---
title: Container Shell Access
---

# Attaching and Executing Commands <Badge type="info" text="new" />

Dozzle supports attaching or executing commands within containers. It provides a web-based interface to interact with Docker containers, allowing users to attach to running containers and execute commands directly from the browser. This feature is particularly useful for debugging and troubleshooting containerized applications. This feature is **disabled** by default as it may pose security risks. To enable it, set the `DOZZLE_ENABLE_SHELL` environment variable to `true`.

::: code-group

```sh
docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --enable-shell
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
      DOZZLE_ENABLE_SHELL: true
```

:::

> [!NOTE]
> Shell access should work across all container types, including Docker, Kubernetes, and other orchestration platforms.
