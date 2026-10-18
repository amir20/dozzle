---
title: Dozzle 与 Portainer 对比
description: 在查看 Docker 日志方面，Dozzle 和 Portainer 有何不同，以及如何同时运行两者。
sourceHash: 676bc9bd86c4
---

# Dozzle 与 Portainer 对比

Portainer 是为管理容器而生的：stack、卷、网络、镜像和用户。Dozzle 是为查看它们的日志而生的。Portainer 有日志查看器，Dozzle 也能启动、停止和更新容器，但两者的重心各不相同。很多人两个都在用。

## 一览

|            | Dozzle                                                                               | Portainer                              |
| ---------- | ------------------------------------------------------------------------------------ | -------------------------------------- |
| 主要用途   | 查看日志                                                                             | 管理容器、stack、卷、网络和镜像        |
| 实时日志   | 日志行一到就流式显示                                                                 | 自动刷新                               |
| 多个容器   | 一个 stack、一个服务或一个自定义分组，在同一个合并视图中查看                         | 一次一个容器                           |
| JSON 日志  | 自动识别、格式化并着色。可对 JSON 字段使用 [SQL](/zh/guide/sql-engine)               | 以文本显示                             |
| 搜索       | 搜索和过滤，支持正则表达式                                                           | 按文本搜索和过滤                       |
| CPU 和内存 | 带近期历史的实时图表                                                                 | 实时统计                               |
| 警报       | 针对日志行、CPU 或内存，以及容器事件。参见[警报](/zh/guide/alerts-and-webhooks)      | 日志查看器中没有                       |
| 镜像更新   | 检查新镜像并按计划更新。参见[更新](/zh/guide/updates)                                | 手动拉取并重新部署                     |
| 多台主机   | [Agent](/zh/guide/agent)、[Swarm](/zh/guide/swarm-mode)、[Kubernetes](/zh/guide/k8s) | 环境（Environments）                   |
| 部署       | 一个容器，无需任何配置                                                               | 一个容器，首次启动时还需创建管理员账户 |

## 什么时候该用 Portainer

如果你想在浏览器里创建和编辑 stack，管理卷、网络和镜像，或者让团队只能访问部分环境，就选 Portainer。这些 Dozzle 都不打算做。

## 什么时候 Dozzle 更合适

当问题是“这个容器现在在做什么？”时，就选 Dozzle。它打开就是日志，能同时跟踪多个容器，无需任何设置就能让 JSON 变得易读。它默认也是只读的：[操作](/zh/guide/actions)和[终端访问](/zh/guide/shell)在你开启之前都是关闭的。

## 同时运行两者

两者互不冲突。它们都读取 Docker 套接字，并运行在不同的端口上：

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    ports:
      - 8080:8080
  portainer:
    image: portainer/portainer-ce:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - portainer-data:/data
    ports:
      - 9443:9443
volumes:
  portainer-data:
```

要从 Portainer 跳转到某个容器在 Dozzle 中的日志，请参见[按名称链接到容器](/zh/guide/faq#我们有工具会在新容器创建时用到-dozzle。怎么按容器名称拿到直达链接)。
