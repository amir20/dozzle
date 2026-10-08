---
# https://vitepress.dev/reference/default-theme-home-page
layout: home

titleTemplate: Real-time Docker Log Viewer
description: Dozzle is a lightweight, open-source log viewer for Docker, Swarm, and Kubernetes. Stream logs, watch live stats, and update containers safely from your browser.

hero:
  name: "Dozzle"
  text: "See what your containers are doing"
  tagline: Real-time Docker logs, stats, and updates — in your browser.
  actions:
    - theme: brand
      text: Get Started
      link: /guide/getting-started
    - theme: alt
      text: View on GitHub
      link: https://github.com/amir20/dozzle

features:
  - title: Real-time Logs
    details: Stream container logs as they happen. Search, filter by level or time range, and query JSON fields with SQL without touching the host.
    icon:
      src: /icons/document.svg
      width: 36
      height: 36
    link: /guide/what-is-dozzle#advanced-log-handling
    linkText: Learn More
  - title: Container & Host Metrics
    details: Watch CPU, memory, network, and disk for every container, with load, disk, and uptime for each host.
    icon:
      src: /icons/chart-line-data.svg
      width: 36
      height: 36
    link: /guide/what-is-dozzle#real-time-monitoring
    linkText: Learn More
  - title: Safe Updates
    details: Spot outdated images and update containers one by one, in bulk, or on a schedule. If a new container fails to stay up, the old one comes back.
    icon:
      src: /icons/update-now.svg
      width: 36
      height: 36
    link: /guide/updates
    linkText: Learn More
  - title: Docker, Swarm & Kubernetes
    details: Watch many Docker hosts, Swarm clusters, or a Kubernetes cluster from one UI. Add TLS agents straight from the browser.
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
    details: An optional managed layer that remembers alerts, flags errors that are new to a container, and tells you what broke on email, Telegram, or Discord.
    icon:
      src: /icons/cloud.svg
      width: 36
      height: 36
    link: /guide/dozzle-cloud
    linkText: Learn More
  - title: Shell & Actions
    details: Start, stop, and restart containers, roll out a Kubernetes restart, or open a shell when you need to dig deeper.
    icon:
      src: /icons/terminal.svg
      width: 36
      height: 36
    link: /guide/actions
    linkText: Learn More
  - title: MCP for AI Assistants
    details: Expose containers, logs, and stats over the Model Context Protocol, with OAuth sign-in, so your coding agent can debug alongside you.
    icon:
      src: /icons/ai.svg
      width: 36
      height: 36
    link: /guide/mcp
    linkText: Learn More
---
