---
title: 什么是 Dozzle？
sourceHash: a37de5cd15f3
---

# 什么是 Dozzle？

Dozzle 是一个由 Docker OSS 赞助的开源项目。它是一个轻量的网页版日志查看器，用于简化 Docker、Docker Swarm 和 Kubernetes 环境中容器化应用的监控与调试。

第一次接触？直接看[快速开始](/zh/guide/getting-started)，一分钟内就能跑起来。

## 主要功能

### <Icon icon="mdi:pulse" inline /> 实时监控

实时流式查看运行中容器的日志，更新即时可见。CPU、内存和网络指标实时呈现，并带有历史图表。

### <Icon icon="mdi:cube-outline" inline /> 灵活部署

可以作为[独立服务](/zh/guide/getting-started)运行，也可以部署为 [Swarm](/zh/guide/swarm-mode) 服务、安装到 [Kubernetes](/zh/guide/k8s)，或者通过[远程 agent](/zh/guide/agent) 覆盖多台主机。

### <Icon icon="mdi:text-search" inline /> 强大的日志处理

自动识别 JSON 并着色、多行堆栈跟踪归组、[过滤器](/zh/guide/filters)，还有内置的 [SQL 引擎](/zh/guide/sql-engine)用于临时查询。

### <Icon icon="mdi:server-network" inline /> 多主机支持

在一个界面里监控多台 Docker 主机上的容器。参见 [agent](/zh/guide/agent)。

### <Icon icon="mdi:console" inline /> 交互式终端

在浏览器里附加到运行中的容器或执行命令。参见[终端访问](/zh/guide/shell)。

### <Icon icon="mdi:gesture-tap-button" inline /> 容器操作

直接在界面上启动、停止、重启和更新容器。参见[操作](/zh/guide/actions)。

### <Icon icon="mdi:bell-ring-outline" inline /> 警报与 Webhook

定义日志匹配规则，触发通知到 Slack、Discord、电子邮件等。参见[警报与 Webhook](/zh/guide/alerts-and-webhooks)。

### <Icon icon="mdi:shield-lock-outline" inline /> 认证

可以完全开放运行，也可以加上[简单认证或前置代理认证](/zh/guide/authentication)，并配合基于角色的访问控制。

### <Icon icon="mdi:feather" inline /> 轻量快速

Go 后端、Vue 3 前端，通过 SSE 和 WebSocket 传输数据，资源占用极低。

## 下一步

- [快速开始](/zh/guide/getting-started)
- [支持的环境变量](/zh/guide/supported-env-vars)
- [常见问题](/zh/guide/faq)

Dozzle 采用 MIT 许可证，并在持续维护中。
