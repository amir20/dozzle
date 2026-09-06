---
title: 健康检查
sourceHash: 4ab4dd21a26a
---

# 启用健康检查

Dozzle 自带 `dozzle healthcheck` 子命令。由于会带来少量 CPU 开销，镜像里默认没有接入它。可以在 compose 文件中启用：

```yaml
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    ports:
      - 8080:8080
    healthcheck:
      test: ["CMD", "/dozzle", "healthcheck"]
      interval: 3s
      timeout: 30s
      retries: 5
      start_period: 30s
```

## 它检查什么

以服务方式运行时，`dozzle healthcheck` 会向自身的 `/healthcheck` 端点发送一个 HTTP `GET` 请求。该端点会 ping 每一个**本地** Docker 客户端（每个客户端最多 3 秒），并返回：

- `200 OK`：至少有一个本地 Docker 客户端响应，**或者**没有配置本地客户端但已知至少一个远程 agent 主机。
- `500 Internal Server Error`：所有本地客户端都 ping 失败，且没有已知的 agent 主机。

远程 agent 被有意排除在服务端健康检查之外，因为某个 agent 不可达不应该让 Dozzle 主进程被判定为不健康。每个 agent 可以暴露自己的健康检查，参见 [Agent 健康检查](/zh/guide/agent#setting-up-healthcheck)。

## 退出码

- `0`：健康（HTTP 200）
- 非零：不健康、网络错误或非 200 响应。失败的 URL 和状态码会输出到 stdout。

该命令会遵循 `--addr` 和 `--base`，因此自定义端口和基础路径时无需额外配置即可工作。

> [!WARNING]
> 由于 Docker 的一个 bug，`healthcheck` 命令无法配合 `--health-cmd` 参数使用。请按上面的示例在 `docker-compose.yml` 中使用 `healthcheck` 配置块。详见 [docker/cli#3719](https://github.com/docker/cli/issues/3719)。
