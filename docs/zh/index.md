---
# https://vitepress.dev/reference/default-theme-home-page
layout: home

titleTemplate: 实时 Docker 日志查看器
description: Dozzle 是一个轻量的开源日志查看器，支持 Docker、Swarm 和 Kubernetes。在浏览器中流式查看日志、实时监控指标并调试容器。

hero:
  name: "Dozzle"
  text: "看清容器正在做什么"
  tagline: 实时的 Docker 日志、指标和调试，全在浏览器里。
  actions:
    - theme: brand
      text: 快速开始
      link: /zh/guide/getting-started
    - theme: alt
      text: 在 GitHub 上查看
      link: https://github.com/amir20/dozzle

features:
  - title: 实时日志
    details: 容器日志一产生就流式呈现。无需登录主机，即可跨容器搜索、过滤和跟踪。
    icon:
      src: /icons/document.svg
      width: 36
      height: 36
    link: /zh/guide/what-is-dozzle#advanced-log-handling
    linkText: 了解更多
  - title: 实时指标与监控
    details: 实时查看 CPU、内存和网络使用情况，每个容器都带有滚动的历史图表。
    icon:
      src: /icons/chart-line-data.svg
      width: 36
      height: 36
    link: /zh/guide/what-is-dozzle#real-time-monitoring
    linkText: 了解更多
  - title: SQL 日志分析
    details: 用 DuckDB 和 WebAssembly 查询日志，完整的 SQL，全部在浏览器中运行。
    icon:
      src: /icons/sql.svg
      width: 36
      height: 36
    link: /zh/guide/sql-engine
    linkText: 了解更多
  - title: 警报与 Webhook
    details: 用强大的表达式匹配日志模式，通知 Slack、Discord、ntfy 或任意 webhook。
    icon:
      src: /icons/notification-new.svg
      width: 36
      height: 36
    link: /zh/guide/alerts-and-webhooks
    linkText: 了解更多
  - title: 多主机与 Swarm
    details: 在一个界面里连接多台 Docker 主机和 Swarm 集群，通过 TLS agent 保障安全。
    icon:
      src: /icons/network-3.svg
      width: 36
      height: 36
    link: /zh/guide/remote-hosts
    linkText: 了解更多
  - title: 终端与命令执行
    details: 需要深入排查时，直接在浏览器里附加到运行中的容器或执行命令。
    icon:
      src: /icons/terminal.svg
      width: 36
      height: 36
    link: /zh/guide/shell
    linkText: 了解更多
  - title: 终端里的 dtop
    details: 一个命令行搭档，实时展示你的容器，并能直接跳转到 Dozzle。
    icon:
      src: /icons/terminal-command.svg
      width: 36
      height: 36
    link: /zh/guide/dtop
    linkText: 了解更多
  - title: 面向 AI 助手的 MCP
    details: 通过 Model Context Protocol 暴露容器、日志和指标，让你的编程助手和你一起调试。
    icon:
      src: /icons/ai.svg
      width: 36
      height: 36
    link: /zh/guide/mcp
    linkText: 了解更多
  - title: 自托管与私密
    details: 运行在你自己的基础设施上，支持简单认证或前置代理认证。日志永远不会离开你的网络。
    icon:
      src: /icons/locked.svg
      width: 36
      height: 36
    link: /zh/guide/authentication
    linkText: 了解更多
sourceHash: a11fae50734d
---
