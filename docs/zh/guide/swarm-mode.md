---
title: Swarm 模式
sourceHash: f73672e30c85
---

# Swarm 模式

<Badge type="warning" text="Docker Only" />

Dozzle 支持 Docker Swarm 模式。在 Swarm 模式下，Dozzle 会自动发现服务和自定义分组。Dozzle 内部并不使用 Swarm API，因为它的能力[有限](https://github.com/moby/moby/issues/33183)。取而代之的是，Dozzle 用 swarm 标签实现了自己的分组逻辑。此外，Dozzle 会合并同一分组中各容器的统计数据。也就是说，你可以在一个视图中看到分组内所有容器的日志和统计。不过这也意味着每台主机都需要部署 Dozzle。

## <Icon icon="mdi:cogs" inline /> 它是如何工作的？

在 Swarm 模式下部署时，Dozzle 会在 swarm 的所有节点之间建立一个安全的网状网络，用于各个 Dozzle 实例之间的通信。这个网状网络使用带私有 TLS 证书的 [mTLS](https://www.cloudflare.com/learning/access-management/what-is-mutual-tls) 建立。因此各个 Dozzle 实例之间的所有通信都是加密的，可以放心地部署在任何地方。

Dozzle 支持用 Docker [stack](https://docs.docker.com/reference/cli/docker/stack/deploy/)、[service](https://docs.docker.com/engine/swarm/how-swarm-mode-works/services/) 和自定义分组把日志合并在一起。分组容器时使用的是 `com.docker.stack.namespace` 和 `com.docker.compose.project` 标签。对于服务，Dozzle 使用服务名作为分组名，也就是 `com.docker.swarm.service.name`。

## <Icon icon="mdi:rocket-launch-outline" inline /> 如何启用 Swarm 模式？

要部署到 swarm 中的每个节点，可以使用 `mode: global`，它会把 Dozzle 部署到 swarm 的每个节点上。下面是一个使用 Docker Stack 的示例：

```yml
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      - DOZZLE_MODE=swarm
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /opt/dozzle/data:/data
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

注意 `DOZZLE_MODE` 环境变量被设成了 `swarm`。这会让 Dozzle 自动发现 swarm 中的其他 Dozzle 实例。`overlay` 网络用于在各个 Dozzle 实例之间建立网状网络。

挂载 `/data` 卷是为了持久化 Dozzle 的配置（通知、云端设置、自定义 stack）。由于 Dozzle 以 global 方式部署在每个节点上，请在每个节点上挂载一个主机路径，让各个实例在重启后都能保留自己的本地状态。

> [!WARNING]
> 在 Docker Swarm 模式下无法使用 socket-proxy。这个限制来自 Docker 本身，而不是 Dozzle。在 Swarm 模式下，服务只能与其他服务通信，而 Dozzle 需要直接连接到单个代理实例，这是不被支持的。如果你有在 Swarm 模式下使用 socket-proxy 的方案，欢迎告诉我们！

## <Icon icon="mdi:shield-lock-outline" inline /> 在 Swarm 模式下配置简单身份验证

要配置简单身份验证，可以用 Docker secret 来存放 `users.yml` 文件。下面是一个使用 Docker Stack 的示例：

```yml
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      - DOZZLE_LEVEL=debug
      - DOZZLE_MODE=swarm
      - DOZZLE_AUTH_PROVIDER=simple
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /opt/dozzle/data:/data
    secrets:
      - source: users
        target: /data/users.yml

    ports:
      - "8080:8080"
    networks:
      - dozzle
    deploy:
      mode: global

networks:
  dozzle:
    driver: overlay
secrets:
  users:
    file: users.yml
```

在这个例子中，`users.yml` 文件存放在 Docker secret 里。其内容与[简单身份验证](/zh/guide/authentication#generating-users-yml)示例中的相同。

## <Icon icon="mdi:server-plus-outline" inline /> 在 Swarm 模式下添加独立代理

在 Swarm 模式下运行时，Dozzle 支持添加独立的[代理](/zh/guide/agent)。

只需按照平常的方式，把[远程代理添加](/zh/guide/agent#how-to-connect-to-an-agent)到你的 Swarm compose 文件中即可。

> [!NOTE]
> 虽然支持远程代理，但套接字代理之类的远程连接方式不受支持。

```yml
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      - DOZZLE_MODE=swarm
      - DOZZLE_REMOTE_AGENT=agent:7007
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /opt/dozzle/data:/data
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

现在这些远程代理会和其他节点一起显示在 Dozzle 中。
