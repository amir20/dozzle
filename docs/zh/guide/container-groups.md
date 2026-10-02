---
title: 容器分组
sourceHash: 83c35e4fb25c
---

# 容器分组

Dozzle 会根据 stack 名称或服务名称自动对容器进行分组。你也可以通过标签创建自定义分组。

## 默认分组

在主机模式下，容器默认按 stack 名称分组。如果存在 `com.docker.swarm.service.name` 标签，Dozzle 会自动启用“Swarm 模式”，将所有服务名称相同的容器合并在一起。

## 自定义分组

此外，你可以给容器添加标签来创建自定义分组。标签名为 `dev.dozzle.group`，值为分组名称。所有分组名称相同的容器会在界面中合并显示。例如，若分组名为 `myapp`，则所有带有 `dev.dozzle.group=myapp` 标签的容器都会被合并在一起。

下面是使用 Docker Compose 或 Docker CLI 的示例：

::: code-group

```sh
docker run --label dev.dozzle.group=myapp hello-world
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: hello-world
    labels:
      - dev.dozzle.group=myapp
```

:::

## Kubernetes

在 Kubernetes 模式下，侧边栏按命名空间和所有者对 Pod 进行分组。在 Pod 模板上添加 `dev.dozzle.group` 标签或注解，还会把该 Pod 中的所有容器加入一个自定义分组。标签和注解的效果相同，因此如果分组名称包含空格，请使用注解。两者都设置时，以注解为准。

```yaml [deployment.yaml]
spec:
  template:
    metadata:
      labels:
        dev.dozzle.group: myapp
```
