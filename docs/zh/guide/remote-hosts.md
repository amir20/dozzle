---
title: 远程主机配置
sourceHash: 6ae09165c838
---

# 远程主机配置

<Badge type="warning" text="Docker Only" />

Dozzle 支持连接到远程 Docker 主机。当 Dozzle 运行在容器中、而你想监控另一台 Docker 主机时，这会很有用。

不过，使用 Dozzle 代理可以在不暴露 Docker 套接字的情况下连接远程主机。更多信息请参见[代理](/zh/guide/agent)页面。

Dozzle 代理省去了远程暴露 Docker 套接字的需要，但不能在 Dozzle 代理的 stack 内部配合 Docker 套接字代理使用。如果你想单独使用套接字代理而不使用 Dozzle 代理，请参见[通过套接字代理连接](#connecting-with-a-socket-proxy)一节。

> [!WARNING]
> 远程主机已经被代理取代。代理提供了更安全的方式来连接远程主机。虽然远程主机仍然受支持，但推荐使用代理。更多信息和示例请参见[代理](/zh/guide/agent)页面。两者的对比参见[代理与远程连接的比较](/zh/guide/agent#comparing-agents-with-remote-connection)一节。远程主机相关的问题我无法逐一排查，因为太耗时间了。

## 使用 TLS 连接远程主机

远程主机可以通过 `--remote-host` 或 `DOZZLE_REMOTE_HOST` 配置。所有证书都必须挂载到 `/certs` 目录。`/certs` 目录中需要有 `/certs/{ca,cert,key}.pem`，多主机时则是 `/certs/{host}/{ca,cert,key}.pem`。

注意这里说的 `{host}` 是配置中的 IP 或 FQDN，而不是[可选的标签](#adding-labels-to-hosts)。

可以多次使用 `--remote-host` 标志来指定多台主机。但使用 `DOZZLE_REMOTE_HOST` 时，值需要用逗号分隔。

::: code-group

```sh [cli]
$ docker run -v /var/run/docker.sock:/var/run/docker.sock -v /path/to/certs:/certs -p 8080:8080 amir20/dozzle --remote-host tcp://167.99.1.1:2376 --remote-host tcp://167.99.1.2:2376
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /path/to/certs:/certs
    ports:
      - 8080:8080
    environment:
      DOZZLE_REMOTE_HOST: tcp://167.99.1.1:2376,tcp://167.99.1.2:2376
```

:::

## 通过套接字代理连接

如果你在私有网络中，可以使用 [Docker Socket Proxy](https://github.com/Tecnativa/docker-socket-proxy)，它不需要 TLS 就能暴露 `docker.sock` 文件。这样就不需要 Dozzle 代理了，Dozzle 会直接连接到套接字代理。Dozzle 永远不会尝试写入 Docker，但它需要访问列表类 API。下面的命令会以最小权限启动一个代理：

```sh
$ docker container run --privileged -e CONTAINERS=1 -e INFO=1 -v /var/run/docker.sock:/var/run/docker.sock -p 2375:2375 tecnativa/docker-socket-proxy
```

> [!TIP]
> 列出运行中的容器需要 `CONTAINERS=1`。`EVENTS` 也是必需的，但它默认已启用。列出系统信息需要 `INFO=1`。

不带任何证书运行 Dozzle 应该就能工作。示例如下：

::: code-group

```sh [cli]
$ docker run -p 8080:8080 amir20/dozzle --remote-host tcp://123.1.1.1:2375
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    ports:
      - 8080:8080
    environment:
      DOZZLE_REMOTE_HOST: tcp://123.1.1.1:2375
```

:::

使用远程主机时，挂载 `/var/run/docker.sock` 是可选的。但你至少需要有一台可以连接的远程主机。

> [!WARNING]
> Docker Socket Proxy 会把 Docker API 暴露到互联网上。如果没有做好安全防护，这会带来安全风险。

## 为主机添加标签

`--remote-host` 支持在连接字符串后面用 `|` 追加主机标签。例如 `--remote-host tcp://123.1.1.1:2375|foobar.com` 会在界面中使用 foobar.com 作为标签。下面是使用 CLI 或 compose 的完整示例：

::: code-group

```sh
docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --remote-host tcp://123.1.1.1:2375|foobar.com
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /path/to/certs:/certs
    ports:
      - 8080:8080
    environment:
      DOZZLE_REMOTE_HOST: tcp://167.99.1.1:2376|foo.com,tcp://167.99.1.2:2376|bar.com
```

:::

> [!WARNING]
> Dozzle 使用 Docker API 收集主机信息。每个代理都需要一个唯一的主机 ID。它们使用 Docker 的系统 ID 或节点 ID 来标识主机。如果使用 swarm，则使用节点 ID。如果你看不到全部主机，可能是配置了多台主机 ID 相同的重复主机。要解决这个问题，请删除 `/var/lib/docker/engine-id` 文件。更多信息参见 [FAQ](/zh/guide/faq#i-am-seeing-duplicate-hosts-error-in-the-logs-how-do-i-fix-it)。

## 修改 localhost 的标签

`localhost` 是一个特殊连接，使用的配置方式与 `--remote-host` 不同。要修改 localhost 的标签，可以使用 `--hostname` 标志或 `DOZZLE_HOSTNAME` 环境变量。用法示例参见[主机名](/zh/guide/hostname)页面。
