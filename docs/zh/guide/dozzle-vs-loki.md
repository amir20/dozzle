---
title: Dozzle 与 Grafana Loki 对比
description: 查看 Docker 日志时是否需要 Grafana 和 Loki，还是 Dozzle 就够了，以及如何同时使用两者。
sourceHash: 95a7cee5f63a
---

# Dozzle 与 Grafana Loki 对比

Loki 是一个日志存储。采集器把每一行日志发送进去，Loki 按你配置的时长保留它们，而 Grafana 是你查询它们的地方。Dozzle 什么都不存储：它直接从 Docker 读取日志，也就是 `docker logs` 显示的那些，并把它们流式传到你的浏览器。

所以真正的问题是：你是否需要把日志保存在 Docker 以外的地方。

## 一览

|                        | Dozzle                                                     | Grafana + Loki                                    |
| ---------------------- | ---------------------------------------------------------- | ------------------------------------------------- |
| 需要运行什么           | 一个容器                                                   | Loki、Grafana，以及一个采集器，例如 Grafana Alloy |
| 日志存放在哪里         | 在 Docker 中，就按它原本保存的方式                         | 复制到 Loki 自己的存储中                          |
| 能回溯多久             | Docker 日志轮转保留多久就多久                              | Loki 的保留期设多长就多长                         |
| 实时跟踪               | 默认视图                                                   | 可在 Grafana Explore 中使用                       |
| 查询                   | 搜索、正则表达式、对 JSON 使用 [SQL](/zh/guide/sql-engine) | LogQL，包括从日志计算出的指标                     |
| 仪表盘                 | 容器 CPU 和内存                                            | 你在 Grafana 中搭建的任何仪表盘                   |
| 警报                   | 日志、指标和事件[警报](/zh/guide/alerts-and-webhooks)      | 基于 LogQL 查询的 Grafana 告警                    |
| 来自 Docker 之外的日志 | 不支持                                                     | 支持：文件、systemd journal、Kubernetes 等        |

## 什么时候该用 Loki

当你需要在 Docker 轮转掉日志之后仍能查看它们，当容器被删除后其日志必须保留，或者当你需要搜索数周的历史时，就选 Loki。如果你已经在运行 Grafana，想把日志和指标放在一起，或者需要来自非容器的日志，它也更合适。

要为部署做好规划。Promtail 是大多数旧教程使用的采集器，已于 2026 年 3 月停止维护，因此新安装要使用 Grafana Alloy 或 Loki Docker 驱动。

## 什么时候 Dozzle 更合适

当你想看一个容器现在在做什么，或者它在过去几个小时或几天里做了什么，又不想运行一套日志管道时，就选 Dozzle。不需要估算存储，不需要调整保留期，也不需要学习查询语言。它几秒钟就能启动，参见[快速开始](/zh/guide/getting-started)。

Dozzle 能显示多少历史取决于 Docker 的日志轮转。要保留更多，请在 `/etc/docker/daemon.json` 中调高限制：

```json
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "50m",
    "max-file": "5"
  }
}
```

新的限制只对 Docker 重启后创建的容器生效。

## 同时使用两者

很多部署用 Loki 保存历史，用 Dozzle 做日常调试。它们互不干扰：Dozzle 从 Docker 读取，Alloy 也从 Docker 读取。如果你改为让容器使用 Loki Docker 驱动，请保持 Docker 的[双重日志](https://docs.docker.com/engine/logging/dual-logging/)开启（默认即开启），这样 `docker logs`，以及 Dozzle，仍然可以读取它们。此时 Docker 把这份副本保存在一个缓存中，其大小由 `cache-max-size` 和 `cache-max-file` 决定（默认 5 个 20 MB 的文件），而不是上面的 `max-size`。
