---
title: 反向代理与基础路径
sourceHash: f344ea2a42cc
---

# 反向代理与基础路径

Dozzle 经常被放在反向代理之后，用于 TLS 终止、身份验证，或者与其他服务共用一个主机名。本页既讲如何把 Dozzle 挂载到子路径下，也讲让流式传输正常工作所需的代理设置。

## 修改基础路径

Dozzle 默认挂载在 `/`。这可以通过 `--base` 标志或 `DOZZLE_BASE` 环境变量修改。例如，挂载到 `/foobar`：

::: code-group

```sh
docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --base /foobar
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
      DOZZLE_BASE: /foobar
```

:::

Dozzle 将在 `http://localhost:8080/foobar/` 上可用。这个选项会把所有静态资源重写为 `/foobar/{file.path}`，并自动把 `/foobar` 重定向到 `/foobar/`。

## 对代理的要求

Dozzle 通过 **Server-Sent Events（SSE）** 流式传输日志，并使用 **WebSocket** 提供容器终端。反向代理必须：

1. **关闭响应缓冲** —— SSE 在事件发生时立即推送。任何缓冲都会导致日志成批到达，甚至根本收不到。Dozzle 会发送 `X-Accel-Buffering: no`，但有些代理会忽略它。
2. **转发 WebSocket 升级头** —— 终端和 attach 功能需要它。
3. **不要压缩 `text/event-stream`** —— 压缩中间件经常会破坏 SSE。

## Nginx

```nginx
location ^~ /foobar/ {
    proxy_pass http://dozzle:8080;

    chunked_transfer_encoding off;
    proxy_buffering off;
    proxy_cache off;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
}
```

如果 Dozzle 挂载在根路径，去掉 `^~ /foobar/` 前缀即可。另请参见 FAQ 中关于[关闭缓冲](/zh/guide/faq#disabling-buffering-in-nginx)的条目。

## Traefik

Traefik 会自动处理 WebSocket 升级，但默认的 `compress` 中间件会破坏 SSE。把 `text/event-stream` 排除掉：

```yaml
http:
  middlewares:
    middlewares-compress:
      compress:
        excludedContentTypes:
          - text/event-stream
```

Dozzle 服务上典型的 labels 配置如下：

```yaml
services:
  dozzle:
    image: amir20/dozzle:latest
    labels:
      - traefik.enable=true
      - traefik.http.routers.dozzle.rule=Host(`dozzle.example.com`)
      - traefik.http.routers.dozzle.entrypoints=websecure
      - traefik.http.routers.dozzle.tls.certresolver=letsencrypt
      - traefik.http.services.dozzle.loadbalancer.server.port=8080
```

## Caddy

```caddyfile
dozzle.example.com {
    reverse_proxy dozzle:8080 {
        flush_interval -1
    }
}
```

`flush_interval -1` 会为流式接口关闭响应缓冲。

## 常见问题

- **使用 `--base` 后页面空白或静态资源 404** —— 代理在转发之前把路径前缀去掉了。请配置它把完整路径原样传给 Dozzle。
- **日志几秒后就停止** —— 代理上的连接超时太短。把读取/发送超时提高到至少几分钟（例如 Nginx 的 `proxy_read_timeout 3600s`）。
- **终端立即断开** —— WebSocket 升级头没有被转发。请检查 `Upgrade` 和 `Connection` 头。
