---
title: 容器名称
sourceHash: 31d0ec398f6d
---

# 容器名称

默认情况下，Dozzle 直接从 Docker 获取容器名称。这通常已经够用，因为可以通过 `docker run` 的 `--name` 参数或 Docker Compose 服务中的 `container_name` 字段来自定义这些名称。

## 自定义名称

如果无法修改容器名称本身，可以给容器添加 `dev.dozzle.name` 标签来覆盖它。

下面是使用 Docker Compose 或 Docker CLI 的示例：

::: code-group

```sh
docker run --label dev.dozzle.name=hello hello-world
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: hello-world
    labels:
      - dev.dozzle.name=hello
```

:::

## Coolify 集成

如果你使用 [Coolify](https://coolify.io/)，Dozzle 会自动识别 Coolify 的标签作为备用值：

- `coolify.serviceName` → 未设置 `dev.dozzle.name` 时用作容器名称
- `coolify.projectName` → 未设置 `dev.dozzle.group` 时用于分组

Coolify 部署无需额外配置。
