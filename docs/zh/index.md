---
# https://vitepress.dev/reference/default-theme-home-page
layout: home

titleTemplate: 实时 Docker 日志查看器
description: Dozzle 是一个轻量的开源日志查看器，支持 Docker、Swarm 和 Kubernetes。在浏览器中流式查看日志、实时监控指标，并安全地更新容器。

hero:
  name: "Dozzle"
  text: "看清容器正在做什么"
  tagline: 实时的 Docker 日志、指标和更新，全在浏览器里。
  actions:
    - theme: brand
      text: 快速开始
      link: /zh/guide/getting-started
    - theme: alt
      text: 在 GitHub 上查看
      link: https://github.com/amir20/dozzle

features:
  - title: 实时日志
    details: 容器日志一产生就流式呈现。无需登录主机，即可按级别或时间范围过滤，并用 SQL 查询 JSON 字段。
    icon:
      src: /icons/document.svg
      width: 36
      height: 36
    link: /zh/guide/what-is-dozzle#advanced-log-handling
    linkText: 了解更多
  - title: 容器与主机指标
    details: 查看每个容器的 CPU、内存、网络和磁盘，以及每台主机的负载、磁盘和运行时间。
    icon:
      src: /icons/chart-line-data.svg
      width: 36
      height: 36
    link: /zh/guide/what-is-dozzle#real-time-monitoring
    linkText: 了解更多
  - title: 安全更新
    details: 发现过时的镜像，逐个、批量或按计划更新容器。新容器无法稳定运行时，旧容器会被恢复。
    icon:
      src: /icons/update-now.svg
      width: 36
      height: 36
    link: /zh/guide/updates
    linkText: 了解更多
  - title: Docker、Swarm 与 Kubernetes
    details: 在一个界面里查看多台 Docker 主机、Swarm 集群或 Kubernetes 集群，并直接在浏览器中添加 TLS agent。
    icon:
      src: /icons/network-3.svg
      width: 36
      height: 36
    link: /zh/guide/remote-hosts
    linkText: 了解更多
  - title: 警报与 Webhook
    details: 用表达式匹配日志模式、指标和生命周期事件，通知 Slack、Discord、ntfy 或任意 webhook。
    icon:
      src: /icons/notification-new.svg
      width: 36
      height: 36
    link: /zh/guide/alerts-and-webhooks
    linkText: 了解更多
  - title: Dozzle Cloud
    details: 可选的托管层，会记住警报、标记容器中新出现的错误，并通过邮件、Telegram 或 Discord 告诉你哪里坏了。
    icon:
      src: /icons/cloud.svg
      width: 36
      height: 36
    link: /zh/guide/dozzle-cloud
    linkText: 了解更多
  - title: 终端与操作
    details: 启动、停止和重启容器，触发 Kubernetes 滚动重启，或在需要深入排查时打开终端。
    icon:
      src: /icons/terminal.svg
      width: 36
      height: 36
    link: /zh/guide/actions
    linkText: 了解更多
  - title: 面向 AI 助手的 MCP
    details: 通过 Model Context Protocol 暴露容器、日志和指标，支持 OAuth 登录，让你的编程助手和你一起调试。
    icon:
      src: /icons/ai.svg
      width: 36
      height: 36
    link: /zh/guide/mcp
    linkText: 了解更多
sourceHash: 135ab46c4cca
---
