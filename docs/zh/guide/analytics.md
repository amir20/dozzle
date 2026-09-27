---
title: 匿名统计
sourceHash: 8421075e674a
---

# 统计数据的采集

Dozzle 通过一个轻量的信标采集匿名使用数据，用于确定功能和修复的优先级。这是一个没有资金支持的开源项目，因此这些数据是决定精力投向的主要依据。

## 收集哪些数据

Dozzle 在启动时发送一次信标，每次有人打开界面时再发送一次。它们合起来包含：

- Dozzle 版本、部署模式（server、swarm、k8s、agent）以及 Docker Engine 版本
- 启用了哪个认证提供方，以及是否开启了操作和 shell
- 少量计数：主机、代理、运行中的容器和过滤器的数量
- 浏览器的 user agent 字符串，仅包含在界面信标中
- Docker Engine 的 ID，用来避免重复统计同一个安装

绝不会传输日志内容、容器名称、镜像名称、主机名或用户标识。具体字段会随时间变化，权威来源是 [`types/beacon.go`](https://github.com/amir20/dozzle/blob/master/types/beacon.go)，发送代码位于 [`internal/analytics/http_beacon.go`](https://github.com/amir20/dozzle/blob/master/internal/analytics/http_beacon.go)。

## 数据存储在哪里

信标发送到 `https://b.dozzle.dev/event`，由开源的 Go 服务 [drain](https://github.com/amir20/drain) 接收，并写入数据库和 Parquet 文件以供分析。drain 不会保存信标来源的 IP 地址，数据也不会与任何第三方共享。

## 如何关闭

传入 `--no-analytics` 或设置 `DOZZLE_NO_ANALYTICS=true`，就不会发出任何信标请求。

```yaml
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      DOZZLE_NO_ANALYTICS: "true"
```
