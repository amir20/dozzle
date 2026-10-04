---
title: 容器操作
sourceHash: 1bc77bb2b87e
---

# 容器操作

<Badge type="warning" text="Docker Only" />

Dozzle 支持容器操作，你可以通过容器统计信息右侧的下拉菜单对容器执行 `start`、`stop`、`restart`、`remove` 和 `update`。该功能默认**禁用**，把环境变量 `DOZZLE_ENABLE_ACTIONS` 设为 `true` 即可启用。

`update` 操作会拉取容器的最新镜像，并用相同的配置重新创建它，适合在不改动 compose 文件的情况下就地升级容器。只有当镜像使用会移动的标签（比如 `latest`、`stable`）时，`update` 才有实际效果；固定的标签只会重新拉取同一个镜像。

旧容器会被重命名并保留，直到新容器连续运行 10 秒且没有重启，并且在镜像带有健康检查时报告为健康。如果新容器无法启动、退出、重启或变为不健康，Dozzle 会删除它并恢复旧容器，更新会显示**已回滚**及原因。新容器会带上标签 `dev.dozzle.previous-image`（被替换的镜像 ID）和 `dev.dozzle.previous-ref`（该镜像的 `repo@sha256:…` 摘要，本地构建的镜像没有此标签）。未在运行的容器（例如已经退出的一次性任务）会基于新镜像重新创建并保持停止状态，因此不会被重新运行，也不会被检查。

> [!WARNING]
> `remove` 会删除容器：其可写层中的数据会丢失，其匿名卷会被留下，不再挂载到任何容器。`update` 会重新创建容器，并保留所有卷（包括匿名卷）和所有绑定挂载。只有写入容器可写层的数据会丢失。

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

带有该标签的容器遵循与 [Dozzle 自身的自动更新](/zh/guide/setup-wizard#auto-update) 相同的计划，你可以在设置向导中设置，也可以通过 `DOZZLE_AUTO_UPDATE` 和 `DOZZLE_AUTO_UPDATE_TIME` 设置。到了这个时间，Dozzle 会把每个带标签的容器与其镜像仓库进行比对，只更新有新镜像的容器。这些容器先更新，Dozzle 最后更新。

自动更新是刻意设计为需要主动开启的。使用 `postgres:latest` 这类浮动标签的数据库，可能会升级到一个无法读取现有数据文件的新主版本，所以只给那些你愿意在无人看管时被替换的容器加标签。Dozzle [无法检查](#哪些情况无法检查) 的容器（例如来自私有仓库的容器）永远不会被自动更新。

自动更新在服务器模式下运行，也包括 [远程代理](/zh/guide/agent) 上的容器。它需要开启操作。

## 清理旧镜像 {#cleaning-up-old-images}

每次更新都会在主机上留下被替换的镜像，因此 Dozzle 会在更新后删除旧镜像，类似 Watchtower 的 `--cleanup`。清理始终进行，适用于所有更新：定时更新、容器的 `Update` 操作，以及更新面板。无需任何开关。

Dozzle 会保留容器之前运行的镜像，以便还能回退到它，并删除再之前的那个。从 1.4.1 更新到 1.4.2 会删除 1.4.0 并保留 1.4.1，因此每个容器最多保留一个备用镜像。Dozzle 从旧容器的 `dev.dozzle.previous-image` 标签读取要删除的镜像，所以容器的第一次更新不会删除任何镜像。

只有在更新完成且旧容器已删除之后才会清理。已回滚的更新不会删除任何镜像。Dozzle 只删除没有标签且没有容器使用的镜像：仍带有标签的镜像（例如你自己拉取或构建的镜像）会被保留，并且删除不强制，因此只要还有其他容器（无论运行中还是已停止）在使用它，Docker 就会拒绝删除。删除被拒绝不会导致更新失败。

[远程代理](/zh/guide/agent) 上的容器也以同样方式清理，Dozzle 自身的容器也是如此：新的 Dozzle 稳定运行后，[自更新](/zh/guide/setup-wizard#self-update) 的辅助容器会删除再之前的那个镜像。Swarm 服务（包括以 Swarm 服务运行的 Dozzle）不会被清理，因为每个节点保存自己的镜像，并且 Swarm 会自行清理任务历史。

## 回滚 {#rolling-back}

容器更新后，容器菜单会提供 **回滚到** 更新前运行的镜像，更新抽屉中每个已更新的容器也会出现 **回滚** 链接。Dozzle 通过自身的更新得知这个镜像，即更新在容器上留下的 `dev.dozzle.previous-image` 标签，或者通过它启动以来在主机上看到的更新得知，这也涵盖了由 Watchtower 或 `docker compose` 完成的更新。只有当 Dozzle 知道之前的镜像时，这一项才会出现。

回滚与更新一样替换容器：当前容器会保留到之前的镜像稳定运行为止，否则会被恢复。设置和卷保持不变。如果之前的镜像已从主机上删除，Dozzle 会按其摘要重新拉取。它从不拉取标签，因为标签现在指向较新的镜像，所以已删除的本地构建镜像无法恢复。

Dozzle 在回滚前会请求确认，并提醒两件事。较新的版本可能已将容器卷中的数据迁移为旧版本无法读取的格式。对于 compose 项目，下一次 `docker compose pull` 会重新带回较新的镜像，除非 compose 文件固定了旧镜像。

之后，自动更新计划会对该容器跳过较新的镜像，直到其标签再次指向更新的镜像。回滚稳定运行后，回滚前的镜像会像其他旧镜像一样被 [清理](#cleaning-up-old-images)，因此只有在没有标签再指向它时才会被删除。

回滚适用于独立容器，包括 [远程代理](/zh/guide/agent) 上的容器。Swarm 服务、Kubernetes 和 Dozzle 自身的容器不支持回滚。
