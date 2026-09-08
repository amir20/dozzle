---
# https://vitepress.dev/reference/default-theme-home-page
layout: home

titleTemplate: Real-time Docker Log Viewer
description: Dozzle is a lightweight, open-source log viewer for Docker, Swarm, and Kubernetes. Stream logs, watch live stats, and debug containers from your browser.

hero:
  name: "Dozzle"
  text: "See what your containers are doing"
  tagline: Real-time Docker logs, stats, and debugging — in your browser.
  actions:
    - theme: brand
      text: Get Started
      link: /guide/getting-started
    - theme: alt
      text: View on GitHub
      link: https://github.com/amir20/dozzle

features:
  - title: Real-time Logs
    details: Stream container logs as they happen. Search, filter, and follow across containers without touching the host.
    icon:
      src: /icons/document.svg
      width: 36
      height: 36
    link: /guide/what-is-dozzle#advanced-log-handling
    linkText: Learn More
  - title: Live Stats & Metrics
    details: Watch CPU, memory, and network usage update in real time, with rolling history charts on every container.
    icon:
      src: /icons/chart-line-data.svg
      width: 36
      height: 36
    link: /guide/what-is-dozzle#real-time-monitoring
    linkText: Learn More
  - title: Multi-host & Swarm
    details: Connect to multiple Docker hosts and Swarm clusters from a single UI, secured with TLS agents.
    icon:
      src: /icons/network-3.svg
      width: 36
      height: 36
    link: /guide/remote-hosts
    linkText: Learn More
  - title: Alerts & Webhooks
    details: Match log patterns, metrics, and lifecycle events with expressions, then notify Slack, Discord, ntfy, or any webhook.
    icon:
      src: /icons/notification-new.svg
      width: 36
      height: 36
    link: /guide/alerts-and-webhooks
    linkText: Learn More
  - title: Dozzle Cloud
    details: An optional managed layer that groups repeated failures, summarizes what broke, and reaches you on email, Telegram, or Discord.
    icon:
      src: /icons/cloud.svg
      width: 36
      height: 36
    link: /guide/dozzle-cloud
    linkText: Learn More
  - title: Shell & Exec Access
    details: Attach to running containers or exec commands directly from the browser when you need to dig deeper.
    icon:
      src: /icons/terminal.svg
      width: 36
      height: 36
    link: /guide/shell
    linkText: Learn More
  - title: Self-hosted & Private
    details: Runs in your own infrastructure with simple or forward-proxy auth. Your logs never leave your network.
    icon:
      src: /icons/locked.svg
      width: 36
      height: 36
    link: /guide/authentication
    linkText: Learn More
  - title: MCP for AI Assistants
    details: Expose containers, logs, and stats over the Model Context Protocol so your coding agent can debug alongside you.
    icon:
      src: /icons/ai.svg
      width: 36
      height: 36
    link: /guide/mcp
    linkText: Learn More
---
