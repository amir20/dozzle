---
title: 容器链接
sourceHash: d39d7f2119c5
---

# 容器链接

大多数值得盯着的容器本身也提供网页界面。加上 `dev.dozzle.url` 标签后，Dozzle 会在容器名旁边显示一个链接，让你从日志直接跳到应用本身。

::: code-group

```sh
docker run --label dev.dozzle.url=https://grafana.example.com grafana/grafana
```

```yaml [docker-compose.yml]
services:
  grafana:
    image: grafana/grafana
    labels:
      - dev.dozzle.url=https://grafana.example.com
```

:::

链接会出现在三个位置：侧边栏、容器表格，以及容器页面的标题栏。它总是在新标签页中打开，点击时不会离开当前日志页面。

## 标签接受什么值

值必须是绝对的 `http` 或 `https` URL。其他任何形式，比如相对路径、纯主机名或别的协议，都会被忽略，不会渲染出链接。

Dozzle 不会检查该 URL 是否可解析，也不会按主机重写它。你写什么，链接就打开什么，所以请填写在你查看 Dozzle 的那个浏览器中可用的地址。

## 为什么没有自动识别

Dozzle 知道容器发布了哪些端口，但发布的端口不等于可访问的 URL。反向代理、自定义路径、TLS 和网络隔离都会让猜测频繁出错，足以让人厌烦。用标签的方式更明确：Dozzle 只显示你写下的链接。

对于还没有标签的容器，Dozzle 会在名称旁显示一个浅色的链接图标，仪表盘和容器页面上都有。点击后会展开一段可以复制到 compose 文件里的片段，并预填一个猜测值。关掉它之后，提示在所有位置都不再出现。

猜测值来自两个地方。首先读取 Traefik 的路由标签，因为路由规则指明的地址确实能从浏览器访问到该容器：

```yaml
labels:
  - traefik.http.routers.grafana.rule=Host(`grafana.example.com`)
  - traefik.http.routers.grafana.tls=true
```

路径前缀会被追加上去，协议根据路由的 TLS 设置和 entrypoints 决定，`traefik.enable=false` 则会整体关闭这一识别。如果读不到这些，Dozzle 会退回到用发布的主机端口，配合你当前访问 Dozzle 所用的主机名。这两者都只会预填到代码片段里。在你自己把它写进 `dev.dozzle.url` 之前，它们都不会变成链接。

位于反向代理后面的容器根本不发布主机端口，所以 Traefik 标签通常是唯一的线索。如果你用的是其他代理，提示不会出现，需要手动添加标签。

## Swarm

在 Swarm 中，`deploy.labels` 给服务设置标签，顶层的 `labels` 键给容器设置标签。Traefik 的 swarm provider 读取服务标签，所以大家都把标签写在那里：

```yaml
services:
  ui:
    image: my/ui
    deploy:
      labels:
        - traefik.http.routers.ui.rule=Host(`app.example.com`)
        - dev.dozzle.url=https://app.example.com
```

Dozzle 会把服务标签回读到每个任务容器上，所以 `dev.dozzle.url` 和 Traefik 提示都能通过 `deploy.labels` 生效。`dev.dozzle.name`、`dev.dozzle.group` 和 `dev.dozzle.icon` 同理。直接设置在容器上的标签优先于服务上的标签。

列出服务是仅限管理节点的 API。在多节点 swarm 中，工作节点上的 agent 读不到服务标签，因此调度到那里的容器只能看到自己身上的标签。

## 相关标签

- [`dev.dozzle.name`](/zh/guide/container-names) 设置自定义显示名称
- [`dev.dozzle.group`](/zh/guide/container-groups) 把容器分到一组
- [`dev.dozzle.icon`](/zh/guide/app-icons) 指定应用图标
