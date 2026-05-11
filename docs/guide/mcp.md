---
title: MCP Integration
---

# MCP Integration

<Badge type="tip" text="Docker" />
<Badge type="tip" text="Swarm" />

Dozzle supports the [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) to allow AI coding assistants to interact with your Docker containers. When enabled, Dozzle exposes an MCP endpoint at `/api/mcp` using the Streamable HTTP transport, served from the same container — no extra processes or sidecars needed.

This feature is **disabled** by default. To enable it, set the `--enable-mcp` flag or `DOZZLE_ENABLE_MCP` environment variable to `true`.

::: code-group

```sh [cli]
docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --enable-mcp
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    ports:
      - 8080:8080
    environment:
      DOZZLE_ENABLE_MCP: true
```

:::

## Available Tools

All tools are **read-only** and do not modify containers.

| Tool                   | Description                                                                          |
| ---------------------- | ------------------------------------------------------------------------------------ |
| `list_containers`      | List all containers across all hosts. Supports optional `state` filter.              |
| `get_container_logs`   | Fetch structured logs with detected levels, JSON parsing, and multi-line grouping.   |
| `list_hosts`           | List all connected Docker hosts.                                                     |
| `get_container_stats`  | Get CPU and memory usage history for a container.                                    |

## Configuring MCP Clients

### VS Code (GitHub Copilot / Copilot Chat)

Add the following to your `.vscode/mcp.json` or user MCP settings:

```json
{
  "servers": {
    "dozzle": {
      "type": "http",
      "url": "http://localhost:8080/api/mcp"
    }
  }
}
```

### Claude Desktop

Add the following to your Claude Desktop MCP configuration:

```json
{
  "mcpServers": {
    "dozzle": {
      "type": "streamable-http",
      "url": "http://localhost:8080/api/mcp"
    }
  }
}
```

> [!NOTE]
> Replace `localhost:8080` with your Dozzle instance address. If Dozzle is configured with a custom base path (e.g., `--base /dozzle`), the MCP endpoint will be at `/dozzle/api/mcp`.

## Authentication

The MCP endpoint is part of the authenticated API group. When authentication is enabled, MCP clients must provide valid credentials.

### Simple Auth

With `--auth-provider simple`, MCP clients need to include a valid JWT token in the `Authorization` header. To obtain a token:

1. Send a `POST` request to `/api/token` with your username and password.
2. Configure your MCP client to send the token as a Bearer header.

For example, in VS Code MCP settings:

```json
{
  "servers": {
    "dozzle": {
      "type": "http",
      "url": "http://localhost:8080/api/mcp",
      "headers": {
        "Authorization": "Bearer <your-jwt-token>"
      }
    }
  }
}
```

### Forward Proxy Auth

With `--auth-provider forward-proxy`, the reverse proxy in front of Dozzle handles authentication and injects the appropriate headers. MCP clients should connect through the same proxy, and authentication will be handled transparently.

### No Auth

With no authentication provider configured (default), the MCP endpoint is publicly accessible. No additional configuration is needed.
