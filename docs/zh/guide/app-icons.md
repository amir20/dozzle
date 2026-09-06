---
title: 应用图标
sourceHash: b11b42409e46
---

# 应用图标

Dozzle 会把常见的容器镜像匹配到对应项目的 logo，并显示在侧边栏、容器表格和命令面板中的容器名旁边。如果你跑的是 \*arr 全家桶、Plex 或 Home Assistant，列表会好认得多。

图标随 Dozzle 一起打包，不会从 CDN 获取，所以容器的任何信息都不会离开你的网络，在离线环境中也能正常工作。

## 关闭该功能

开关位于**设置 → 选项 → 显示应用图标**。这是按配置文件保存的设置，只对你自己的浏览器生效。

## 匹配规则

Dozzle 只看镜像名，忽略仓库地址、标签和摘要。以最后一段路径为准，所以下面这些都会匹配到 Sonarr：

- `sonarr`
- `linuxserver/sonarr:latest`
- `lscr.io/linuxserver/sonarr`
- `ghcr.io/hotio/sonarr@sha256:...`

当镜像名过于通用时，Dozzle 会退回到命名空间来判断。`ghcr.io/goauthentik/server` 就是这样匹配到 Authentik 的。

## 覆盖图标

有些镜像匹配不上，而 fork 出来的镜像也可能匹配到错误的 logo。设置 `dev.dozzle.icon` 标签可以自己指定图标，设为 `none` 则可以隐藏该容器的图标。

::: code-group

```sh
docker run --label dev.dozzle.icon=plex my-custom-media-server
```

```yaml [docker-compose.yml]
services:
  media:
    image: my-custom-media-server
    labels:
      - dev.dozzle.icon=plex

  scratch:
    image: alpine
    labels:
      - dev.dozzle.icon=none
```

:::

标签的值是 [dashboard-icons](https://github.com/homarr-labs/dashboard-icons) 中的图标名称。只有 Dozzle 打包进来的图标可用，未知名称会退回到不显示图标。

## 缺少某个图标？

为了控制镜像体积，Dozzle 只收录了精选的一部分图标，而不是全部 3000 多个。如果某个常见图标缺失，请带上镜像名[提交 issue](https://github.com/amir20/dozzle/issues)，可以再补进来。
