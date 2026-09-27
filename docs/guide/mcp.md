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

| Tool                    | Description                                                                        |
| ----------------------- | ---------------------------------------------------------------------------------- |
| `list_containers`       | List all containers across all hosts. Supports optional `state` filter.            |
| `get_container_logs`    | Fetch structured logs with detected levels, JSON parsing, and multi-line grouping. |
| `search_container_logs` | Search container logs for a keyword or phrase. Returns only matching entries.      |
| `list_hosts`            | List all connected Docker hosts.                                                   |
| `get_container_stats`   | Get CPU and memory usage history for a container.                                  |

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

### Simple Auth and OIDC

With `--auth-provider simple` or `--auth-provider oidc`, Dozzle is an OAuth authorization server for MCP clients. Clients that support MCP authorization (VS Code, Claude Code, Claude Desktop and others) sign in on their own. Add the server with no headers, and the first time the client connects it opens a browser tab:

1. Sign in to Dozzle the way you normally do (password, GitHub or your OIDC provider).
2. Dozzle shows a consent page with the client's name and the address it will send you back to. Choose **Allow**.
3. The browser returns to the client, which stores the token and refreshes it on its own.

The token only works on `/api/mcp` and carries the same roles and container filters as your browser session. Access tokens last an hour. Refresh tokens stop working 30 days after you approved the client, and the client then asks you to approve it again. Anything that signs everyone out of Dozzle, such as editing `users.yml` or changing the OIDC issuer, also revokes MCP tokens.

> [!NOTE]
> Dozzle builds its OAuth URLs from the request, the same way it builds the OIDC callback. Behind a reverse proxy, forward `Host` (or `X-Forwarded-Host`) and `X-Forwarded-Proto`. With a custom base path, some clients look for metadata under `/.well-known/` at the root of the domain, so route `/.well-known/` to Dozzle if you can.

#### Clients without OAuth support

With simple auth, a client that can only send static headers can use a session token instead:

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

This is not available with oidc, because there is no password to exchange for a token.

### Forward Proxy Auth

With `--auth-provider forward-proxy`, the reverse proxy in front of Dozzle handles authentication and injects the appropriate headers. MCP clients should connect through the same proxy, and authentication will be handled transparently.

### No Auth

With no authentication provider configured (default), the MCP endpoint is publicly accessible. No additional configuration is needed.
