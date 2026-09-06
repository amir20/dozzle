---
title: 匿名统计
sourceHash: 0fedd8c524b8
---

# 统计数据的采集

Dozzle 通过一个轻量的信标采集匿名使用数据，用于确定功能和修复的优先级。这是一个没有资金支持的开源项目，因此这些数据是决定精力投向的主要依据。

## 采集了哪些内容

概括地说，信标包含 Dozzle 版本、部署模式（server、swarm、k8s、agent）、启用的认证方式、少量功能开关、Docker Engine 版本，以及一些数量统计（主机数、容器数、过滤器数）。其中还有一个随机生成的安装 ID，用于去重。

日志内容、容器名称、镜像名称、IP 地址和用户标识都不会被发送。具体字段会随时间变化，以 [`types/beacon.go`](https://github.com/amir20/dozzle/blob/master/types/beacon.go) 为准，发送逻辑位于 [`internal/analytics/http_beacon.go`](https://github.com/amir20/dozzle/blob/master/internal/analytics/http_beacon.go)。

## 数据存储在哪里

事件会发送到 `https://b.dozzle.dev/event`，这是一个小型 Go 服务，它把事件写入 DigitalOcean 上的平面文件，供后续处理。

## 如何关闭

传入 `--no-analytics` 或设置 `DOZZLE_NO_ANALYTICS=true`，就不会发出任何信标请求。

```yaml
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      DOZZLE_NO_ANALYTICS: "true"
```
