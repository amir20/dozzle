---
title: Container Shell Access
---

# Attaching and Executing Shell Commands

<Badge type="tip" text="Docker" />
<Badge type="tip" text="K8s" />

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

## Security

Anyone who can reach the Dozzle UI will be able to open a shell inside your containers — equivalent to `docker exec`. Before enabling `--enable-shell` on a publicly reachable Dozzle, put it behind [authentication](/guide/authentication). Role-based permissions can restrict shell access to specific users.

## Kubernetes

In k8s mode, shell access uses the Kubernetes API rather than `docker exec`. The target pod must contain an executable shell (`/bin/sh`, `/bin/bash`, etc.) — minimal images built `FROM scratch` or distroless images without a shell will not be attachable.
