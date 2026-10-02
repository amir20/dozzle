---
title: 容器名称
sourceHash: cafa1526d194
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

## Kubernetes

在 Kubernetes 模式下，容器默认命名为 `<pod>/<container>`。在 Pod 模板上设置 `dev.dozzle.name` 即可覆盖它。更推荐使用注解（annotation），因为标签值不能包含空格，且长度上限为 63 个字符。如果两者都设置了，以注解为准。其他所有 `dev.dozzle.*` 设置也同样如此，例如 `dev.dozzle.url` 和 `dev.dozzle.icon`。

```yaml [deployment.yaml]
spec:
  template:
    metadata:
      annotations:
        dev.dozzle.name: Public API
```

该名称作用于整个 Pod。init 容器始终会以自己的名称作为后缀，例如 `Public API/migrate`；在包含多个应用容器的 Pod 中，每个容器也是如此，例如 `Public API/proxy`。

同一 Deployment 的所有副本会得到相同的名称，而 Dozzle 会把同名的容器视为同一个容器。在侧边栏中固定其中一个副本会固定所有副本，容器标题菜单也会把其他副本列为同一容器的历史运行记录。

## Coolify 集成

如果你使用 [Coolify](https://coolify.io/)，Dozzle 会自动识别 Coolify 的标签作为备用值：

- `coolify.serviceName` → 未设置 `dev.dozzle.name` 时用作容器名称
- `coolify.projectName` → 未设置 `dev.dozzle.group` 时用于分组

Coolify 部署无需额外配置。
