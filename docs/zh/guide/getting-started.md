---
title: 快速开始
sourceHash: 6fa151a446d0
---

# 快速开始

Dozzle 以单个容器的方式运行。请在下面选择 Docker CLI、Docker Compose、Swarm 或 Kubernetes。

## <Icon icon="mdi:docker" inline /> 独立 Docker

挂载 `docker.sock`，让 Dozzle 能读取容器；在 `/data` 挂载一个卷，让设置在重启后依然保留；并映射 8080 端口。

::: code-group

```sh [docker run]
docker run -d -v /var/run/docker.sock:/var/run/docker.sock -v dozzle_data:/data -p 8080:8080 amir20/dozzle:latest
```

```yaml [docker-compose.yml]
# 使用 docker compose up -d 运行
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - dozzle_data:/data
    ports:
      - 8080:8080
    environment:
      # 取消注释以启用容器操作（停止、启动、重启）。参见 https://dozzle.dev/guide/actions
      # - DOZZLE_ENABLE_ACTIONS=true
      #
      # 取消注释以允许访问容器终端。参见 https://dozzle.dev/guide/shell
      # - DOZZLE_ENABLE_SHELL=true
      #
      # 取消注释以启用身份验证。参见 https://dozzle.dev/guide/authentication
      # - DOZZLE_AUTH_PROVIDER=simple
      #
      # 为这个 Dozzle 实例命名（显示在标题栏和多主机菜单中）。参见 https://dozzle.dev/guide/hostname
      # - DOZZLE_HOSTNAME=my-server
      #
      # 连接一个或多个远程代理以监控其他 Docker 主机。参见 https://dozzle.dev/guide/agent
      # - DOZZLE_REMOTE_AGENT=192.168.1.10:7007,192.168.1.11:7007
      #
      # 只显示匹配过滤条件的容器。参见 https://dozzle.dev/guide/filters
      # - DOZZLE_FILTER=label=com.example.app
volumes:
  dozzle_data:
```

:::

打开 `http://localhost:8080` 就可以了。其余功能，包括容器操作、终端访问、身份验证和远程代理，都是可选的，默认关闭。Compose 文件里被注释掉的环境变量都链接到了对应的指南。

> [!WARNING]
> 挂载 `docker.sock` 相当于把主机的 root 权限交给了 Dozzle。如果你打算把 Dozzle 暴露到私有网络之外，请先阅读[安全注意事项](/zh/guide/authentication#security-considerations)。

Dozzle 需要 Docker Engine 19.03 或更高版本（API 版本 1.40+）。如果你的网络无法访问 Docker Hub，可以改为从 [GitHub Container Registry](https://ghcr.io/amir20/dozzle:latest) 拉取 `ghcr.io/amir20/dozzle:latest`。

## <Icon icon="mdi:hexagon-multiple-outline" inline /> Docker Swarm

Dozzle 支持在 Swarm 模式下运行，方式是把它部署到每个节点上。要以 Swarm 模式运行 Dozzle，可以使用下面的配置：

```yaml [dozzle-stack.yml]
# 使用 docker stack deploy -c dozzle-stack.yml <name> 运行
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      - DOZZLE_MODE=swarm
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    ports:
      - 8080:8080
    networks:
      - dozzle
    deploy:
      mode: global
networks:
  dozzle:
    driver: overlay
```

然后用下面的命令部署这个 stack：

```bash
docker stack deploy -c dozzle-stack.yml <name>
```

更多信息请参见 [Swarm 模式](/zh/guide/swarm-mode)。

## <Icon icon="mdi:kubernetes" inline /> K8s

Dozzle 支持在 Kubernetes 中运行，只需要部署在集群中的一个节点上。你需要设置 `DOZZLE_MODE=k8s`，并为读取 Pod 日志配置 RBAC。

完整的配置方式，包括 RBAC、Deployment 和 Service 清单，请参见 [Kubernetes 模式](/zh/guide/k8s)。
