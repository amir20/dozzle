---
title: What is Dozzle?
---

# What is Dozzle?

Dozzle is an open-source project sponsored by Docker OSS. It is a lightweight, web-based log viewer designed to simplify monitoring and debugging containerized applications across Docker, Docker Swarm, and Kubernetes environments.

New here? Jump to [Getting Started](/guide/getting-started) to run it in under a minute.

## Key Features

### Real-time Monitoring

Stream logs from running containers with instant updates. Live CPU, memory, and network metrics with historical visualizations.

### Flexible Deployment

Run as a [standalone server](/guide/getting-started), a [Swarm](/guide/swarm-mode) deployment, a [Kubernetes](/guide/k8s) install, or with [remote agents](/guide/agent) across multiple hosts.

### Advanced Log Handling

Automatic JSON detection and color coding, multi-line stack-trace grouping, [filters](/guide/filters), and an embedded [SQL engine](/guide/sql-engine) for ad-hoc queries.

### Multi-Host Support

Monitor containers across multiple Docker hosts from one UI. See [agents](/guide/agent).

### Interactive Terminal

Attach or exec into running containers from the browser. See [Shell Access](/guide/shell).

### Container Actions

Start, stop, restart, and update containers directly from the UI. See [Actions](/guide/actions).

### Alerts & Webhooks

Define log patterns that trigger notifications to Slack, Discord, email, and more. See [Alerts and Webhooks](/guide/alerts-and-webhooks).

### Authentication

Run open, or layer in [simple or forward-proxy auth](/guide/authentication) with role-based access control.

### Lightweight & Fast

Go backend, Vue 3 frontend, streaming over SSE and WebSocket — minimal resource footprint.

## Next Steps

- [Getting Started](/guide/getting-started)
- [Supported Environment Variables](/guide/supported-env-vars)
- [FAQ](/guide/faq)

Dozzle is MIT-licensed and actively maintained.
