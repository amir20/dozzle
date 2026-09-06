---
title: 容器终端访问
sourceHash: 267b2a6665a0
---

# 附加到容器并执行命令

<Badge type="tip" text="Docker" />
<Badge type="tip" text="K8s" />

Dozzle 支持附加到容器或在容器内执行命令。它提供了一个基于网页的界面来与 Docker 容器交互，用户可以直接在浏览器中附加到运行中的容器并执行命令。这个功能在调试和排查容器化应用问题时特别有用。由于可能带来安全风险，该功能默认是**关闭**的。要启用它，请把 `DOZZLE_ENABLE_SHELL` 环境变量设为 `true`。

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
> 终端访问应当适用于所有类型的容器，包括 Docker、Kubernetes 以及其他编排平台。

## <Icon icon="mdi:shield-lock-outline" inline /> 安全性

任何能访问 Dozzle 界面的人都能在你的容器里打开一个终端，效果等同于 `docker exec`。在可公开访问的 Dozzle 上启用 `--enable-shell` 之前，请先为它加上[认证](/zh/guide/authentication)。基于角色的权限可以把终端访问限制给特定用户。

## <Icon icon="mdi:kubernetes" inline /> Kubernetes

在 k8s 模式下，终端访问走 Kubernetes API，而不是 `docker exec`。目标 pod 中必须有可执行的 shell（`/bin/sh`、`/bin/bash` 等），基于 `FROM scratch` 构建的极简镜像或不带 shell 的 distroless 镜像无法附加。
