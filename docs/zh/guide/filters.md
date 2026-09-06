---
title: 过滤器
sourceHash: e380a612fd7f
---

# 过滤容器

<Badge type="tip" text="Docker" />
<Badge type="tip" text="K8s" />

Dozzle 通过 `DOZZLE_FILTER` 或 `--filter` 支持条件过滤，用法类似 Docker 的 [--filter](https://docs.docker.com/reference/cli/docker/container/ls/#filter)。过滤条件会直接传给 Docker，用来限制 Dozzle 能看到的容器。例如按标签过滤可以写成 `--filter "label=color"`，效果等同于 `docker ps --filter "label=color"`。

::: code-group

```sh
docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --filter label=color
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
      DOZZLE_FILTER: label=color
```

:::

常用的过滤条件是 `name` 或 `label`，用来限制 Dozzle 可以访问的容器。

## 界面、Agent 与用户过滤器

Dozzle 支持多种过滤器来限制可见的容器。过滤器可以在界面级、agent 级或用户级设置。

1. **界面过滤器**：作用于 Dozzle 界面实例，会传给 Docker 以限制可见容器。它对所有 agent 以及没有单独设置过滤器的用户生效。
2. **Agent 过滤器**：在 agent 级设置，会传给 Docker 以限制该 agent 暴露的容器。Agent 过滤器和界面过滤器共同起作用。
3. **用户过滤器**：在用户级设置，决定该用户能看到哪些容器。如果没有定义用户过滤器，Dozzle 默认使用界面过滤器。

关于为特定用户设置过滤器，详见[用户过滤器](/zh/guide/authentication#setting-specific-filters-for-users)。关于为 agent 设置过滤器，详见 [agent 过滤器](/zh/guide/agent#setting-up-filters)。

> [!WARNING]
> 需要注意的是，多个过滤器会叠加生效。例如在界面级设置 `--filter label=color`，在 agent 级设置 `--filter label=type`，那么 Dozzle 只会显示同时带有 `color` 和 `type` 两个标签的容器。
