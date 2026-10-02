---
title: 容器操作
sourceHash: 347c47e3b567
---

# 容器操作

<Badge type="warning" text="Docker Only" />

Dozzle 支持容器操作，你可以通过容器统计信息右侧的下拉菜单对容器执行 `start`、`stop`、`restart`、`remove` 和 `update`。该功能默认**禁用**，把环境变量 `DOZZLE_ENABLE_ACTIONS` 设为 `true` 即可启用。

`update` 操作会拉取容器的最新镜像，并用相同的配置重新创建它，适合在不改动 compose 文件的情况下就地升级容器。只有当镜像使用会移动的标签（比如 `latest`、`stable`）时，`update` 才有实际效果；固定的标签只会重新拉取同一个镜像。

> [!WARNING]
> `remove` 和 `update` 会重新创建容器。写入**匿名卷**或容器可写层的数据会丢失。具名卷和绑定挂载则会保留。

::: code-group

```sh
docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --enable-actions
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
      DOZZLE_ENABLE_ACTIONS: true
```

:::

## 更新检查

Dozzle 会检查容器正在运行的镜像是否仍然是其仓库提供的那一个。如果两者不同，容器菜单上会出现一个圆点，菜单中会提示有可用更新。

检查的方式是向仓库查询容器创建时所用标签的摘要，并与容器实际运行的摘要作比较。这是通过对镜像 manifest 发起一个 `HEAD` 请求完成的，因此不会下载任何层，也不会计入 Docker Hub 的拉取速率限制。结果会缓存六小时，而且无论有多少容器或主机在运行同一个镜像，都只会查询一次。

由于比较的对象是容器正在_运行_的镜像，所以即使主机上已经拉取了更新的镜像，容器在被重新创建之前仍然算作过期。

检查和操作是相互独立的。无论 Dozzle 是否被允许做出改动，知道容器已过期本身就是有用的，所以即使 `DOZZLE_ENABLE_ACTIONS` 关闭，这个提示也会出现。只有 `Update` 按钮需要启用容器操作。

### 关闭它

`DOZZLE_IMAGE_CHECK_MODE` 控制 Dozzle 是否会去访问镜像仓库。

| 值          | 行为                                             |
| ----------- | ------------------------------------------------ |
| `automatic` | 在查看容器时于后台检查。                         |
| `manual`    | 从不自动检查。菜单中会提供“检查更新”操作。       |
| `off`       | 功能完全关闭。不注册任何接口，也不发出任何请求。 |

它的默认值取决于 `DOZZLE_RELEASE_CHECK_MODE` 的设置，所以如果你已经让 Dozzle 不自动获取版本发布信息，它也不会自动检查镜像。

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      DOZZLE_IMAGE_CHECK_MODE: off
```

要让某一个容器不再提示（比如一个刻意固定了版本的容器），给它加上标签：

```yaml [docker-compose.yml]
services:
  database:
    image: postgres:18-alpine
    labels:
      dev.dozzle.update-check: false
```

发现更新时也可以显示通知。该功能默认关闭，位于设置中。

### 哪些情况无法检查

有些容器没有可比较的对象，Dozzle 会保持沉默，而不是去猜：

- 本地构建的镜像，它们没有仓库摘要
- 固定到某个摘要的引用，它们不会变化
- 私有仓库，因为 Dozzle 自己没有凭据。在 Kubernetes 中，这也包括通过 `imagePullSecrets` 拉取的镜像。

### Kubernetes

在 Kubernetes 模式下，检查会把 Pod 状态中记录的摘要与仓库进行比较，因此在 k3s、EKS、GKE 等基于 containerd 的集群上无需额外权限即可使用。该提示仅供参考，不会附带 `Update` 按钮，因为 Pod 的镜像由其工作负载决定。对于 `:latest` 这类会变动的标签，并且设置了 `imagePullPolicy: Always`，执行一次[滚动重启](/zh/guide/k8s#rollout-restart)即可获取新镜像。其他情况则需要修改工作负载的 spec。在 Kubernetes 模式下，仪表盘不会显示更新按钮和更新抽屉。

### 更新 Dozzle 自身

Dozzle 自身容器上的 `Update` 操作会就地更新 Dozzle。它拉取新镜像，并把替换工作交给一个短暂存在的辅助容器，所以 Dozzle 会离开几秒钟，然后以新版本回来，配置和卷保持不变。它也可以按计划运行。关于哪些内容会保留以及哪些部署方式不受支持，请参阅 [自更新的工作原理](/zh/guide/setup-wizard#self-update)。以 Swarm 服务方式运行的 Dozzle 通过编排器更新。其他主机上的 Dozzle 代理是普通容器，和其他容器一样更新。

## 一次更新多个容器

开启操作后，仪表板会一次性检查所有容器。过期的容器名称旁会出现一个小圆环，容器列表上方会出现一个 **N 个更新** 按钮。两者都会打开"更新"抽屉，其中列出所有有更新镜像的容器，并且默认全部选中。取消勾选你不想动的容器，然后点击 **更新**。

各主机并行更新，而每台主机一次只更新一个容器，这样任何守护进程都不必同时拉取十几个镜像。抽屉会显示每个容器依次经历拉取、重建和已更新这几个阶段，某个容器失败也不会影响其余容器。更新在服务器端运行，所以关闭标签页不会中断它。再次打开抽屉时，会接着显示当前的进度。

如果 Dozzle 自身的容器也在列表中，它总是最后更新，因为更新它会重启 Dozzle。

当设置了 `DOZZLE_IMAGE_CHECK_MODE=manual` 时，按钮会显示 **检查更新**，直到你点击它。关闭操作时，仪表板和现在完全一样，每个容器自己的菜单仍会提示是否有可用更新。

## 自动更新容器 {#auto-updating-containers}

Dozzle 可以按计划更新容器。用标签让某个容器加入：

```yaml [docker-compose.yml]
services:
  whoami:
    image: traefik/whoami:latest
    labels:
      dev.dozzle.auto-update: true
```

带有该标签的容器遵循与 [Dozzle 自身的自动更新](/zh/guide/setup-wizard#_4-自动更新) 相同的计划，你可以在设置向导中设置，也可以通过 `DOZZLE_AUTO_UPDATE` 和 `DOZZLE_AUTO_UPDATE_TIME` 设置。到了这个时间，Dozzle 会把每个带标签的容器与其镜像仓库进行比对，只更新有新镜像的容器。这些容器先更新，Dozzle 最后更新。

自动更新是刻意设计为需要主动开启的。使用 `postgres:latest` 这类浮动标签的数据库，可能会升级到一个无法读取现有数据文件的新主版本，所以只给那些你愿意在无人看管时被替换的容器加标签。Dozzle [无法检查](#哪些情况无法检查) 的容器（例如来自私有仓库的容器）永远不会被自动更新。

自动更新在服务器模式下运行，也包括 [远程代理](/zh/guide/agent) 上的容器。它需要开启操作。
