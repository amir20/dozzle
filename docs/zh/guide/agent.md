---
title: 代理模式
sourceHash: a73c7ed62d8a
---

# 代理模式

<Badge type="warning" text="仅限 Docker" />

Dozzle 可以运行在代理模式下，把 Docker 主机暴露给其他 Dozzle 实例。所有通信都通过 TLS 加密连接完成。也就是说，你可以把 Dozzle 部署在远程主机上，然后从本地机器连接过去。

> [!WARNING] 代理的安全性取决于它所监听的网络
> Dozzle 自带的证书在每一份镜像副本里都是同一个，它只能加密连接，并不能证明对端是谁。任何能访问 `7007` 端口的人都可以用自己的 Dozzle 连上你的代理，读取该主机上的全部日志，并在其容器里执行命令。代理并不检查 `DOZZLE_ENABLE_SHELL` 或 `DOZZLE_ENABLE_ACTIONS`，这两个开关只决定界面上显示什么。
>
> 请把 `7007` 端口留在私有网络里；如果代理能从别处访问到，就[生成你自己的证书](#自定义证书)，让代理只接受你自己的主实例。

> [!NOTE] 在用 Docker Swarm？
> 如果你使用 Docker Swarm 模式，就不需要代理。Dozzle 会自动发现自身并利用 swarm 模式组建集群。详见 [Swarm 模式](/zh/guide/swarm-mode)。

## <Icon icon="mdi:plus-box-outline" inline /> 如何创建代理

要创建 Dozzle 代理，需要用 `agent` 子命令运行 Dozzle。示例如下：

::: code-group

```sh
docker run -v /var/run/docker.sock:/var/run/docker.sock -p 7007:7007 amir20/dozzle:latest agent
```

```yaml [docker-compose.yml]
services:
  dozzle-agent:
    image: amir20/dozzle:latest
    command: agent
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    ports:
      - 7007:7007
```

:::

> [!NOTE] socket 代理（Docker Socket Proxy）用户请注意
> 如果你使用远程代理，就**不能**在代理之上再加一层 socket proxy。Dozzle 代理是用来**替代** socket proxy 的，更多信息以及如何用 socket proxy 代替代理，请见[远程主机](/zh/guide/remote-hosts)。

代理会启动并监听 `7007` 端口。在 Dozzle 界面中填入代理的 IP 地址和端口即可连接。代理只会显示它所在主机上的容器。

> [!TIP]
> 如果使用 Docker 网络，则不必暴露 7007 端口。同一网络中的其他容器可以直接访问该代理。这是运行代理最安全的方式，因为该网络之外的任何东西都访问不到它。

## <Icon icon="mdi:connection" inline /> 如何连接到代理

连接代理时需要提供代理的 IP 地址和端口。示例如下：

::: code-group

```sh
docker run -p 8080:8080 amir20/dozzle:latest --remote-agent agent:7007
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      - DOZZLE_REMOTE_AGENT=agent:7007
    ports:
      - 8080:8080 # Dozzle UI port
```

:::

注意，连接代理时不必挂载本地的 Docker socket，此时界面上只会显示各个代理上的容器。

> [!TIP]
> 如果你也想在界面中看到本机的容器，请按[快速开始](/zh/guide/getting-started)中的示例挂载 `docker.sock`。

> [!TIP]
> 你可以提供多个 `DOZZLE_REMOTE_AGENT` 环境变量来连接多个代理。例如 `DOZZLE_REMOTE_AGENT=agent1:7007,agent2:7007`。

## <Icon icon="mdi:group" inline /> 主机分组

当你要管理分布在不同环境中的大量代理时，可以为每个代理指定一个分组名称。分组在侧边栏中显示为可折叠的区块，每个分组都有一个“合并全部”按钮，用于查看该分组内所有主机的合并日志。

连接字符串的格式为 `endpoint|name|group`，三个部分都是可选的：

| 格式                            | 结果               |
| ------------------------------- | ------------------ |
| `agent:7007`                    | 不覆盖名称，无分组 |
| `agent:7007\|web-1`             | 覆盖名称，无分组   |
| `agent:7007\|web-1\|Production` | 覆盖名称 + 分组    |
| `agent:7007\|\|Production`      | 默认主机名 + 分组  |

::: code-group

```sh
docker run -p 8080:8080 amir20/dozzle:latest \
  --remote-agent agent1:7007|web-1|Production \
  --remote-agent agent2:7007|web-2|Production \
  --remote-agent agent3:7007|dev-1|Development
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      - DOZZLE_REMOTE_AGENT=agent1:7007|web-1|Production,agent2:7007|web-2|Production,agent3:7007|dev-1|Development
    ports:
      - 8080:8080
```

:::

侧边栏会显示为：

```
▾ Production
    web-1
    web-2
▾ Development
    dev-1
  ungrouped-host   ← agents without a group appear below
```

点击分组名称旁边的合并图标，会打开一个从该分组所有主机汇总的日志视图。合并视图也可以直接通过 `/host-group/<group-name>` 访问。

没有分组的代理行为和以前完全一样，显示在分组区块的下方。

## <Icon icon="mdi:alert-circle-outline" inline /> 常见问题

### 代理没有出现

如果你看到 `An agent with an existing ID was found. Removing the duplicate host.`，说明有两台主机使用了相同的 Server ID。

Dozzle 通过 Docker API 收集主机信息。每个代理都需要一个在重启后保持不变的唯一主机 ID，以便正确识别。目前代理使用 Docker 的 system ID 或 node ID 来标识主机。

如果运行在 Swarm 环境中，则使用 node ID。但如果你发现并非所有主机都可见，可能是因为存在使用相同主机 ID 的重复主机。

要解决这个问题，请删除系统上的 `/var/lib/docker/engine-id` 并重启。这样可以消除重复主机 ID 引起的冲突。更多信息和排查建议请参阅 [FAQ](/zh/guide/faq#i-am-seeing-duplicate-hosts-error-in-the-logs-how-do-i-fix-it)。

## <Icon icon="mdi:cog-outline" inline /> 高级选项

### 配置健康检查

你可以为代理配置健康检查，和主 Dozzle 实例的健康检查类似。在代理模式下，健康检查会检测代理与 Docker 的连接。如果 Docker 不可达，代理会被标记为不健康，并且不会显示在界面上。

使用 `healthcheck` 子命令来配置健康检查。示例如下：

```yml
services:
  dozzle-agent:
    image: amir20/dozzle:latest
    command: agent
    healthcheck:
      test: ["CMD", "/dozzle", "healthcheck"]
      interval: 5s
      retries: 5
      start_period: 5s
      start_interval: 5s
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    ports:
      - 7007:7007
```

### 修改代理名称

和 Dozzle 实例一样，你可以通过 `DOZZLE_HOSTNAME` 环境变量修改代理的名称。示例如下：

::: code-group

```sh
docker run -v /var/run/docker.sock:/var/run/docker.sock -p 7007:7007 amir20/dozzle:latest agent --hostname my-special-name
```

```yaml [docker-compose.yml]
services:
  dozzle-agent:
    image: amir20/dozzle:latest
    command: agent
    environment:
      - DOZZLE_HOSTNAME=my-special-name
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    ports:
      - 7007:7007
```

:::

这会把代理的名称改为 `my-special-name`，连接该代理时界面上会显示这个名称。

### 配置过滤器

你可以为代理配置过滤器，限制它能访问的容器。这些过滤器会直接传给 Docker，从而限制 Dozzle 能看到的内容。

```yaml
services:
  dozzle-agent:
    image: amir20/dozzle:latest
    command: agent
    environment:
      - DOZZLE_FILTER=label=color
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
```

这会让该代理只显示带有 `color` 标签的容器。注意，这些过滤器会和界面上的过滤器叠加，进一步缩小容器范围。想了解各类过滤器，请阅读[过滤器文档](/zh/guide/filters#ui-agents-and-user-filters)。

### 自定义证书

Dozzle 自带一份自签名证书，两端互相出示的是同一份证书，而且每一端只信任这一份，别的都不认。它被编译进了二进制文件，所以每个 Dozzle 安装里都完全相同，任何人都能从公开镜像里把它提取出来。

也就是说，它提供的是加密，而不是身份认证。代理分不清你的主实例和别人的主实例。对于用默认配置启动的代理，唯一把陌生人挡在外面的东西，就是他们访问不到 `7007` 端口。

> [!WARNING] 什么时候需要自己的证书
> 只要 `7007` 端口能被你无法控制的东西访问到，包括把它发布在有公网 IP 的主机上，就请生成你自己的证书对。这样每个部署都会拥有一份别人没有的凭据。

运行 `generate-certs` 生成一份独有的证书对：

```sh
docker run --rm -v "$PWD":/out amir20/dozzle:latest generate-certs --cert-out /out/dozzle_cert.pem --key-out /out/dozzle_key.pem
```

把两个文件都复制到主实例和每个代理上。Dozzle 默认在 `/dozzle_cert.pem` 和 `/dozzle_key.pem` 查找，所以挂载到这两个路径就够了。你也可以用 `--cert` 和 `--key` 参数或者 `DOZZLE_CERT` 和 `DOZZLE_KEY` 环境变量放到别处。

请像对待密码一样对待私钥。拿到它的人都能连上你的代理；而代理会拒绝出示其他证书的主实例，所以主实例和它的代理必须拿到同一份证书对，并且一起重启。

下面是使用默认路径的示例：

```yml
services:
  agent:
    image: amir20/dozzle:latest
    command: agent
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    secrets:
      - source: cert
        target: /dozzle_cert.pem
      - source: key
        target: /dozzle_key.pem
    ports:
      - 7007:7007
secrets:
  cert:
    file: ./cert.pem
  key:
    file: ./key.pem
```

或者用环境变量指定自定义路径：

```yml
services:
  agent:
    image: amir20/dozzle:latest
    command: agent
    environment:
      - DOZZLE_CERT=/certs/my-cert.pem
      - DOZZLE_KEY=/certs/my-key.pem
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./certs:/certs
    ports:
      - 7007:7007
```

或者使用命令行参数：

::: code-group

```sh
docker run -v /var/run/docker.sock:/var/run/docker.sock -v ./certs:/certs -p 7007:7007 amir20/dozzle:latest agent --cert /certs/my-cert.pem --key /certs/my-key.pem
```

```yaml [docker-compose.yml]
services:
  agent:
    image: amir20/dozzle:latest
    command: agent --cert /certs/my-cert.pem --key /certs/my-key.pem
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./certs:/certs
    ports:
      - 7007:7007
```

:::

> [!TIP]
> 推荐使用 Docker secrets 来提供证书。可以用 `docker secret create` 命令创建，也可以像上面的例子那样在 `docker-compose.yml` 中定义。连接该代理的 Dozzle 实例必须使用同一套证书。

这样会把证书和密钥文件挂载到代理中，代理会用这些证书进行通信。连接该代理的 Dozzle 实例必须使用同一套证书。

如果你更想用 openssl 而不是 `generate-certs` 来生成：

```sh
$ openssl genpkey -algorithm Ed25519 -out key.pem
$ openssl req -new -key key.pem -out request.csr -subj "/C=US/ST=California/L=San Francisco/O=My Company"
$ openssl x509 -req -in request.csr -signkey key.pem -out cert.pem -days 365
```

## <Icon icon="mdi:compare-horizontal" inline /> 代理与远程连接的对比

代理和远程连接很相似，但代理有一些优势。出于性能和安全方面的考虑，通常更推荐使用代理。对比如下：

| 特性     | 代理                 | 远程连接                   |
| -------- | -------------------- | -------------------------- |
| 性能     | 负载分散，表现更好   | 界面端表现更差             |
| 安全性   | 加密传输，可自带证书 | 不加密或使用 Docker TLS    |
| 易用性   | 开箱即用             | 需要暴露 Docker socket     |
| 权限     | 对 Docker 的完全访问 | 可以通过 socket proxy 控制 |
| 重连     | 自动重连             | 需要重启界面               |
| 健康检查 | 内置健康检查         | 没有健康检查               |
| 过滤器   | 支持过滤器           | 不支持过滤器               |

如果你确实打算使用远程连接，请务必用 Docker TLS 或反向代理来保护连接。
