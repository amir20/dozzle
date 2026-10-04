---
title: 从 Watchtower 迁移
sourceHash: 3c48dbce256f
---

# 从 Watchtower 迁移

Dozzle 可以做 Watchtower 做的事：按计划检查容器是否有更新的镜像并更新它们。每次更新都会被观察，如果新容器无法稳定运行，就会换回旧容器。本页把 Watchtower 的设置对应到 Dozzle 的设置。

## 开启

Dozzle 需要开启操作，并把 `/data` 放在卷上：

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - dozzle-data:/data
    ports:
      - 8080:8080
    environment:
      DOZZLE_ENABLE_ACTIONS: true
      DOZZLE_AUTO_UPDATE: daily
      DOZZLE_AUTO_UPDATE_TIME: "04:00"
volumes:
  dozzle-data:
```

也可以去掉两行 `DOZZLE_AUTO_UPDATE`，改在 **设置 → 更新** 中设置计划，并在那里选择 **哪些容器**。然后停止 Watchtower，以免两者更新同一批容器。

## 设置

| Watchtower                                                         | Dozzle                                                                                                                                                  |
| ------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `--schedule` 或 `--interval`                                       | `DOZZLE_AUTO_UPDATE`（`daily`，或周日的 `weekly`）和 `DOZZLE_AUTO_UPDATE_TIME`。每天最多一次                                                            |
| 所有容器（默认）                                                   | **哪些容器：全部**                                                                                                                                      |
| `--label-enable` 配合 `com.centurylinklabs.watchtower.enable=true` | **哪些容器：有标签的容器**（默认）配合 `dev.dozzle.update: auto`                                                                                        |
| `com.centurylinklabs.watchtower.enable=false`                      | `dev.dozzle.update: off`                                                                                                                                |
| `--cleanup`                                                        | 始终开启。更新后删除没有标签的旧镜像，并保留一个上一版本镜像用于回滚                                                                                    |
| `--monitor-only`                                                   | 在 **有标签的容器** 下不给容器加标签：容器会被检查并显示为可用更新，但从不自行更新                                                                      |
| `--rolling-restart`                                                | 始终如此：每台主机一次只更新一个容器                                                                                                                    |
| `--notification-url`                                               | 针对容器事件的 [告警和 Webhook](/zh/guide/alerts-and-webhooks)，或 [Dozzle Cloud](/zh/guide/dozzle-cloud)，它会观察每次更新，并在新版本开始出错时告诉你 |
| `--run-once`                                                       | 更新抽屉中的 **更新**                                                                                                                                   |
| 私有仓库凭据                                                       | 不支持。来自私有仓库的容器会被跳过                                                                                                                      |

## 标签

Dozzle 不读取 Watchtower 的标签。把 `com.centurylinklabs.watchtower.enable` 换成 `dev.dozzle.update`：

| `dev.dozzle.update` | 结果                                                                 |
| ------------------- | -------------------------------------------------------------------- |
| `auto`              | 按计划更新，除非 **哪些容器** 选的是 **仅 Dozzle**                   |
| _（无标签）_        | 选 **全部** 时按计划更新。否则会被检查并显示为可用更新，由你手动应用 |
| `off`               | 从不检查，从不更新                                                   |

旧的 Dozzle 标签仍然有效：`dev.dozzle.auto-update=true` 视为 `auto`，`dev.dozzle.update-check=false` 视为 `off`。

## 不同之处

- 旧容器会一直保留，直到新容器稳定运行（如果有健康检查，还要处于健康状态）。否则会恢复旧容器。
- 不健康的容器会被跳过。
- 已停止的容器永远不会被更新，即使它带有 `auto` 标签。
- 使用 [Dozzle Cloud](/zh/guide/dozzle-cloud) 时，开始出错的计划更新可以被[回滚](/zh/guide/actions#rolling-back)。之后在发布更新的镜像之前，计划会跳过该容器。
- 固定到摘要的镜像和本地构建的镜像会被跳过，因为没有更新的版本可比较。
