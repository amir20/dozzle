---
title: 常见问题
sourceHash: c964c824dcab
---

# 常见问题

## Dozzle 启动失败，提示 `client version 1.x is too new`，这是什么意思？

Dozzle 需要 Docker Engine 19.03 或更高版本（API 版本 1.40 及以上）。更老的 daemon，例如 Docker 18.06（API 1.38），底层的 Docker SDK 并不支持，启动时会报出类似 `failed to create docker client: ... client version 1.54 is too new. Maximum supported API version is 1.38` 的错误。

请把 Docker Engine 升级到受支持的版本。作为临时办法，可以把 Dozzle 固定在 `v10.5.2` 或更早的版本，那些版本使用的 Docker SDK 还能向下协商到更老的 API 版本。

## 怎么升级 Dozzle？

Dozzle 遵循标准的 Docker 镜像使用方式。升级时拉取新镜像并重建容器：

```sh
docker pull amir20/dozzle:latest
docker compose up -d dozzle
```

用户设置、通知规则和其他状态都存放在 `/data`（见下文），因此升级过程中要保持这个卷的挂载。生产环境建议固定具体的 tag（例如 `amir20/dozzle:v10.9.2`）而不是 `latest`，这样升级才是可控的。发布说明发布在 [GitHub releases 页面](https://github.com/amir20/dozzle/releases)。回滚也很简单，重新部署旧 tag 即可。

## 我的平台包装了容器 entrypoint，Dozzle 报 `no such file or directory`

默认镜像基于 `FROM scratch` 构建，里面只有 Dozzle 二进制文件，别无他物。没有 shell，也没有解释器。

有些平台通过 bind mount 把一个 `#!/bin/sh` 包装脚本盖在容器 entrypoint 上，再由它重新执行原来的 entrypoint，以此提供可选功能。Unraid 的按容器 Tailscale 开关就是这么做的，一些 sidecar 和 init 注入器也是如此。镜像里没有 `/bin/sh`，包装脚本就无法执行，容器会退出并报出包装脚本的名字，而不是提示缺少 shell：

```
exec /opt/unraid/tailscale: no such file or directory
```

遇到这类情况请使用 `alpine` 变体，它是同一个二进制文件，只是基于 Alpine：

```sh
docker run \
  --volume=/var/run/docker.sock:/var/run/docker.sock \
  -p 8080:8080 \
  amir20/dozzle:alpine
```

带版本号的 tag 也遵循同样的规则（`amir20/dozzle:v10.9.2-alpine`）。其余场景仍推荐基于 scratch 的 `latest`，因为它体积小得多，也没有需要打补丁的发行版软件包。

## `/data` 里存了什么？怎么备份？

`/data` 目录用于持久化所有需要在容器重启后保留的内容：

- `users.yml` / `users.yaml`，简单验证模式的用户文件（如果你创建过）
- 通知规则、通知目标和投递状态
- 每个用户的界面设置（仅多用户模式；单用户模式的设置存在浏览器 localStorage 中）
- 少量内部文件，比如已忽略公告的状态

这个目录很小（一般远小于 10 MB），对挂载的卷做一次简单的 `tar` 或 `rsync` 即可备份。升级或迁移到新主机时，把 `/data` 卷带过去，所有设置也就跟着走了。

## 装好 Dozzle 后日志很慢，或者根本加载不出来，怎么办？

Dozzle 使用 Server Sent Events（SSE），它通过一条不关闭的 HTTP 流连接服务器。如果中间有代理试图缓冲这条连接，Dozzle 就永远收不到数据，只能一直等反向代理刷新缓冲区。从 `1.23.0` 版本起，Dozzle 会发送 `X-Accel-Buffering: no` 头，用来阻止反向代理缓冲。不过有些代理会忽略这个头，这时你需要显式关闭缓冲。

### 在 nginx 中关闭缓冲

下面是 nginx 中用 `proxy_pass` 关闭缓冲的示例：

```
server {
    ...

    location / {
        proxy_pass                  http://<dozzle.container.ip.address>:8080;
    }

    location /api {
        proxy_pass                  http://<dozzle.container.ip.address>:8080;

        proxy_buffering             off;
        proxy_cache                 off;
    }
}
```

### 在 traefik 中关闭压缩

Traefik 反向代理可以通过中间件启用压缩。启用后，常见的配置是这样的：

```
http:
  middlewares:
    middlewares-compress:
      compress: {}
```

在这种配置下，你可能会发现通过 traefik 打开 dozzle（例如 dozzle.mydomain.com）时，某些容器不再显示日志。
而直接访问同一个 dozzle 实例（例如 localhost:8080）时日志却是正常的。

已观察到该现象的容器包括（不完全列举）：dozzle、homepage、glances、filebrowser。

要让日志恢复正常，请把 `text/event-stream` 从压缩中间件里排除：

```
http:
  middlewares:
    middlewares-compress:
      compress:
        excludedContentTypes:
          - text/event-stream
```

## 我们有工具会在新容器创建时用到 Dozzle。怎么按容器名称拿到直达链接？

Dozzle 有一个专门的[路由](https://github.com/amir20/dozzle/blob/master/assets/pages/show.vue)，可以按名称搜索容器并跳转过去。例如，如果有个容器名为 `"foo.bar"`、id 为 `abc123`，你可以把用户导向 `/show?name=foo.bar`，它会转到 `/container/abc123`。

## 装好 Dozzle 但内存占用不显示！

_这是 ARM 设备上特有的问题。_

Dozzle 通过 Docker API 获取容器的内存使用信息。如果内存占用没有显示，很可能是 Docker API 没有返回这个数据。

运行 docker info 就能确认，你应该会看到：

```
WARNING: No memory limit support
WARNING: No swap limit support
```

这种情况下，需要在 `/boot/cmdline.txt` 文件中加上下面这行，然后重启设备：

```
cgroup_enable=cpuset cgroup_enable=memory cgroup_memory=1
```

## 日志里出现主机重复的错误，怎么解决？

如果你在日志里看到下面这样的错误，说明可能有多台主机配置了相同的主机 ID：

```
time="2024-07-10T13:35:53Z" level=warning msg="duplicate host ID: *********, Endpoint: 1.1.1.1:7007 found, skipping"
```

Dozzle 通过 Docker API 收集主机信息。每台主机必须有唯一的 ID，界面上就是靠这个 ID 来区分主机的。在 swarm 模式下，Dozzle 使用 `docker system info` 里的 node ID 标识主机；不使用 swarm 模式时，则用 `docker system info` 里的 system ID 作为主机 ID。

有时虚拟机从备份恢复后会带着相同的主机 ID，这会让 Dozzle 以为该主机已经存在，从而跳过它。要修复这个问题，需要删除 `/var/lib/docker/engine-id` 文件。这个文件保存着主机 ID，在 Docker daemon 启动时创建。

## 日志里出现找不到主机的错误，怎么解决？

这基本上只会出现在 Podman 上：Podman 不像 Docker 那样生成 engine-id。
如果你用的是 Docker，请检查 `/var/lib/docker` 下的 `engine-id` 文件是否存在、权限是否正确、里面是否有 UUID。

按以下步骤解决这个错误：

1. 创建目录：`mkdir -p /var/lib/docker`
2. 如有需要，先安装 uuidgen
3. 用 uuidgen 生成一个 UUID：`uuidgen > engine-id`

现在 engine-id 文件里应该已经有 UUID 了。

Ansible 的配置示例可以在 [Podman](/zh/guide/podman) 中找到。

你可能还需要清理 Podman 下已有的 Dozzle 部署：停掉容器，删除相关数据（容器/卷）。之后重新部署 Dozzle 容器，日志应该就能正常显示了。

## 为什么我只看到运行中的容器？怎么才能看到已停止的容器？

默认情况下 Dozzle 只显示运行中的容器。要看到已停止的容器，需要在设置里打开 `Show Stopped Containers` 选项。这个选项默认关闭，是为了减少界面上显示的容器数量。

## 有办法在多个 Dozzle 实例之间同步我的设置吗？

在单用户模式下，Dozzle 把设置存在浏览器的 local storage 里，也就是说设置只在设置它的那个浏览器上有效。要让 Dozzle 在多个实例之间同步设置，它必须知道当前用户是谁。在多用户模式下，Dozzle 会用用户名把设置存到磁盘上，并在多个实例间同步。这些信息保存在 `/data` 目录中。如果你想在多个实例之间同步设置，需要[启用](/zh/guide/authentication)多用户模式并提供用户名。

## 为什么 Dozzle 不直接支持 Slack、Discord、Telegram、邮件之类的通知？

在设计上，Dozzle 对警报发往何处不做预设。Dozzle 不打包特定通知平台的集成，而是提供带自定义载荷模板的 **webhook**。也就是说，你可以把警报发给_任何_接受 HTTP 请求的服务，Slack、Discord、Telegram、ntfy、PagerDuty、Opsgenie，或是你自己内部的工具，都不必等 Dozzle 专门去做适配。

这么做有几个原因：

- **通用性。** webhook 几乎适用于所有通知平台。做特定服务商的集成只能覆盖用户需求的一小部分，而 webhook 全都能覆盖。
- **维护成本。** 每个服务商的集成都有自己的 API 怪癖、认证流程、限流规则和破坏性变更。支持它们意味着 Dozzle 的维护者要去排查第三方服务的问题，这超出了一个日志查看器该管的范围。
- **简单。** Dozzle 是一个轻量、专注的 Docker 日志查看工具。保持通知层的通用性，代码库才能小巧，项目才能长期维持下去。

如果你需要更成体系、集成更丰富的体验（比如 Web 推送通知、ntfy 操作按钮），[Dozzle Cloud](/zh/guide/dozzle-cloud) 正是为此而生。

关于如何为你偏好的服务配置 webhook，请看[警报与 Webhook](/zh/guide/alerts-and-webhooks)指南，里面内置了 Slack、Discord 和 ntfy 的载荷模板，可以直接用，也可以自行修改。

## 为什么没有浏览器连接时，Dozzle 还让 dockerd 和 containerd 的 CPU 略微偏高？

最后一个浏览器断开后，Dozzle 会继续推送容器统计数据最多 6 小时（Kubernetes 上是 2 小时），然后自行关闭统计采集器。这是有意为之。统计数据持续采集，是为了让你重新打开界面时能看到历史的 CPU 和内存曲线，而不是一张空图。如果关掉标签页的一瞬间就停止采集，就没有历史数据可看了。

代价是 dockerd 和 containerd 上会有少量持续的 CPU 开销，因为 Docker 的统计 API 是轮询式的。重启 Dozzle 容器会立即重置计时器，这也是为什么一重启主机就回到空闲状态。这一点在设计上不可配置。超时太短会破坏其他依赖统计数据持续推送的功能，调低它也就失去了统计历史的意义。

## Swarm 模式下我的 Dozzle 实例超时，或者在负载均衡器后面看不到全部 Swarm 节点，怎么解决？

在 Swarm 模式下，Dozzle 实例可能需要自己的 overlay 网络。如果连接不同 Dozzle 节点时行为不一致，可以像下面这样，新建一个只包含 Dozzle 实例的独立 overlay 网络：

```
services:
  logs:
    ...
    networks: [ traefik, dozzle ]
    ...

networks:
  dozzle:
    driver: overlay
  traefik:
    external: true
```

外部网络 `traefik` 是负载均衡器做服务发现用的 overlay 网络，而我们新建的 `dozzle` overlay 网络供各个 Dozzle 节点相互通信。
