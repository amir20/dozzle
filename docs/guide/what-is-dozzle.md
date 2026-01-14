---
title: What is Dozzle?
---

# What is Dozzle?

Dozzle is an open-source project sponsored by Docker OSS. It is a lightweight, web-based log viewer designed to simplify monitoring and debugging containerized applications across Docker, Docker Swarm, and Kubernetes environments.

## Key Features

**Real-time Monitoring**: Stream logs from running containers with instant updates through an intuitive web interface. Monitor CPU, memory, and network usage with live metrics and historical visualizations.

**Flexible Deployment**: Deploy as a standalone server for single or multi-host Docker monitoring, enable automatic discovery in Docker Swarm clusters, or monitor pod logs in Kubernetes environments.

**Advanced Log Handling**: Automatically detects and formats JSON logs with intelligent color coding. Supports simple text logs, structured JSON logs, and multi-line grouped entries with powerful filtering and search capabilities.

**Multi-Host Support**: Monitor containers across multiple Docker hosts simultaneously through a distributed agent architecture using gRPC.

**Interactive Terminal**: Attach to running containers or execute commands directly through the web interface.

**Lightweight & Fast**: Built with Go backend and Vue 3 frontend, Dozzle uses efficient streaming protocols (SSE/WebSocket) and requires minimal resources.

Dozzle is easy to install and configure, making it an ideal solution for developers and system administrators seeking an efficient log viewer for their containerized environments. The tool is available under the MIT license and is actively maintained by its developer, Amir Raminfar.
