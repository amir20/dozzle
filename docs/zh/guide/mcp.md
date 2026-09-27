---
title: MCP 集成
sourceHash: fe1cee485b1f
---

# MCP 集成

<Badge type="tip" text="Docker" />
<Badge type="tip" text="Swarm" />

Dozzle 支持 [Model Context Protocol (MCP)](https://modelcontextprotocol.io/)，让 AI 编程助手可以与你的 Docker 容器交互。启用后，Dozzle 会在 `/api/mcp` 上使用 Streamable HTTP 传输方式提供一个 MCP 端点，由同一个容器提供服务，不需要额外的进程或 sidecar。

该功能默认**禁用**。要启用它，请把 `--enable-mcp` 标志或 `DOZZLE_ENABLE_MCP` 环境变量设为 `true`。

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

## 可用工具

所有工具都是**只读**的，不会修改容器。

| 工具                    | 说明                                                    |
| ----------------------- | ------------------------------------------------------- |
| `list_containers`       | 列出所有主机上的全部容器。支持可选的 `state` 过滤条件。 |
| `get_container_logs`    | 获取结构化日志，包含识别出的级别、JSON 解析和多行分组。 |
| `search_container_logs` | 在容器日志中搜索关键词或短语，只返回匹配的条目。        |
| `list_hosts`            | 列出所有已连接的 Docker 主机。                          |
| `get_container_stats`   | 获取某个容器的 CPU 和内存使用历史。                     |

## 配置 MCP 客户端

### VS Code（GitHub Copilot / Copilot Chat）

把下面的内容加入你的 `.vscode/mcp.json` 或用户级 MCP 设置：

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

把下面的内容加入你的 Claude Desktop MCP 配置：

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
> 请把 `localhost:8080` 替换为你的 Dozzle 实例地址。如果 Dozzle 配置了自定义基础路径（例如 `--base /dozzle`），MCP 端点就位于 `/dozzle/api/mcp`。

## 身份验证

MCP 端点属于需要身份验证的 API 组。启用身份验证后，MCP 客户端必须提供有效的凭据。

### 简单身份验证和 OIDC

使用 `--auth-provider simple` 或 `--auth-provider oidc` 时，Dozzle 会作为 MCP 客户端的 OAuth 授权服务器。支持 MCP 授权的客户端（VS Code、Claude Code、Claude Desktop 等）会自行完成登录。添加服务器时无需任何请求头，客户端首次连接时会打开一个浏览器标签页：

1. 按平常的方式登录 Dozzle（密码、GitHub 或你的 OIDC 提供商）。
2. Dozzle 会显示一个授权页面，列出客户端名称以及登录后将返回的地址。选择 **允许**。
3. 浏览器返回客户端，客户端保存令牌并自动刷新。

该令牌只能用于 `/api/mcp`，并带有与你的浏览器会话相同的角色和容器过滤器。访问令牌有效期为一小时。刷新令牌在你批准客户端 30 天后失效，届时客户端会请你重新批准。任何会让所有人退出 Dozzle 的操作（例如修改 `users.yml` 或更换 OIDC issuer）也会吊销 MCP 令牌。

> [!NOTE]
> Dozzle 根据请求构造 OAuth URL，与构造 OIDC 回调的方式相同。在反向代理之后，请转发 `Host`（或 `X-Forwarded-Host`）和 `X-Forwarded-Proto`。如果使用了自定义基础路径，有些客户端会在域名根部的 `/.well-known/` 下查找元数据，因此请尽量把 `/.well-known/` 也转发给 Dozzle。

#### 不支持 OAuth 的客户端

使用简单身份验证时，只能发送固定请求头的客户端可以改用会话令牌：

1. 向 `/api/token` 发送包含用户名和密码的 `POST` 请求。
2. 配置 MCP 客户端，将令牌作为 Bearer 请求头发送。

例如在 VS Code 的 MCP 设置中：

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

oidc 模式下无法使用此方式，因为没有可以换取令牌的密码。

### 前置代理身份验证

使用 `--auth-provider forward-proxy` 时，由 Dozzle 前面的反向代理处理身份验证并注入相应的请求头。MCP 客户端应通过同一个代理连接，身份验证会被透明地处理。

### 无身份验证

在没有配置身份验证提供方（默认情况）时，MCP 端点是公开可访问的，不需要额外配置。
