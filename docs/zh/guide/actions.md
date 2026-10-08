---
title: 容器操作
sourceHash: 0b048a644851
---

# 容器操作

<Badge type="warning" text="Docker Only" />

Dozzle 支持容器操作，你可以通过容器统计信息右侧的下拉菜单对容器执行 `start`、`stop`、`restart`、`remove` 和 `update`。该功能默认**禁用**，把环境变量 `DOZZLE_ENABLE_ACTIONS` 设为 `true` 即可启用。

`update` 操作会拉取容器的最新镜像，并用相同的配置重新创建它，适合在不改动 compose 文件的情况下就地升级容器。只有当镜像使用会移动的标签（比如 `latest`、`stable`）时，`update` 才有实际效果；固定的标签只会重新拉取同一个镜像。

> [!WARNING]
> `remove` 会删除容器：其可写层中的数据会丢失，其匿名卷会被留下，不再挂载到任何容器。`update` 会重新创建容器，并保留所有卷（包括匿名卷）和所有绑定挂载。只有写入容器可写层的数据会丢失。

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

## 更新 {#auto-updating-containers}

每次更新都会保留旧容器，直到新容器稳定运行，否则会恢复旧容器。检查新镜像、批量更新、自动更新计划、清理旧镜像和回滚，请参阅[容器更新](/zh/guide/updates)。
