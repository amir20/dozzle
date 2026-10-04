---
title: 容器操作
sourceHash: 121d4b250806
---

# 容器操作

<Badge type="warning" text="Docker Only" />

Dozzle 支持容器操作，你可以通过容器统计信息右侧的下拉菜单对容器执行 `start`、`stop`、`restart`、`remove` 和 `update`。该功能默认**禁用**，把环境变量 `DOZZLE_ENABLE_ACTIONS` 设为 `true` 即可启用。

`update` 操作会拉取容器的最新镜像，并用相同的配置重新创建它，适合在不改动 compose 文件的情况下就地升级容器。只有当镜像使用会移动的标签（比如 `latest`、`stable`）时，`update` 才有实际效果；固定的标签只会重新拉取同一个镜像。

旧容器会被重命名并保留，直到新容器连续运行 10 秒且没有重启，并且在镜像带有健康检查时报告为健康。如果新容器无法启动、退出、重启或变为不健康，Dozzle 会删除它并恢复旧容器，更新会显示**已回滚**及原因。新容器会带上标签 `dev.dozzle.previous-image`（被替换的镜像 ID）和 `dev.dozzle.previous-ref`（该镜像的 `repo@sha256:…` 摘要，本地构建的镜像没有此标签）。已停止的容器永远不会被更新，因为它可能是有意停止的。有新镜像时它仍会显示出来，但不会提供 **更新**，计划会跳过它，而通过其他方式（例如 Dozzle Cloud）请求的更新会被拒绝，并提示“请先启动容器”。启动它之后即可更新。

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

要让某一个容器不再被检查（比如一个刻意固定了版本的容器），给它加上标签。这也会让它不参与[自动更新计划](#auto-updating-containers)。

```yaml [docker-compose.yml]
services:
  database:
    image: postgres:18-alpine
    labels:
      dev.dozzle.update: off
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

Dozzle 可以按计划更新容器。在 **设置 → 更新** 或 [设置向导](/zh/guide/setup-wizard#auto-update) 中设置：

- **时间：** 关闭、每天或每周（周日），以及一天中的时间。等同于 `DOZZLE_AUTO_UPDATE` 和 `DOZZLE_AUTO_UPDATE_TIME`。
- **哪些容器：** **仅 Dozzle**、**有标签的容器**（默认）或 **全部**。无论选哪一项，Dozzle 自身都会按计划更新。等同于 `DOZZLE_UPDATE_CONTAINERS`（`off`、`labelled` 或 `all`）。

容器上的一个标签决定其余的部分：

| `dev.dozzle.update` | 结果                                                                 |
| ------------------- | -------------------------------------------------------------------- |
| `auto`              | 按计划更新，除非 **哪些容器** 选的是 **仅 Dozzle**                   |
| _（无标签）_        | 选 **全部** 时按计划更新。否则会被检查并显示为可用更新，由你手动应用 |
| `off`               | 从不检查，从不更新                                                   |

```yaml
services:
  app:
    image: ghcr.io/example/app:latest
    labels:
      dev.dozzle.update: auto
```

旧标签仍然有效：`dev.dozzle.auto-update=true` 视为 `auto`，`dev.dozzle.update-check=false` 视为 `off`。

到了计划时间，Dozzle 会把计划中的每个容器与其镜像仓库进行比对，只更新有新镜像的容器，Dozzle 自身最后更新。每次更新都是上文所述的安全替换，因此新容器如果无法稳定运行，旧容器会被换回来。已停止或不健康的容器、Dozzle [无法检查](#哪些情况无法检查) 的容器，以及从当前提供的镜像[回滚](#rolling-back)过的容器都会被跳过。**设置 → 更新** 会列出下一次运行将要更新的容器。

选 **全部** 时，使用 `postgres:latest` 这类浮动标签的数据库可能会升级到一个无法读取现有数据文件的主版本。选择 **全部** 时会列出把数据保存在命名卷中的容器。给这些容器加上 `dev.dozzle.update: off` 标签即可将其排除。

**哪些容器** 作为 `updateContainers` 保存在 [`dozzle.yml`](/zh/guide/setup-wizard#dozzle-yml) 中，因此在界面中修改它需要把 `/data` 放在卷上。自动更新在服务器模式下运行，也包括 [远程代理](/zh/guide/agent) 上的容器，并且需要开启操作。从 Watchtower 迁移过来？请参阅 [从 Watchtower 迁移](/zh/guide/moving-from-watchtower)。

## 清理旧镜像 {#cleaning-up-old-images}

每次更新都会在主机上留下被替换的镜像，因此 Dozzle 会在更新后删除旧镜像，类似 Watchtower 的 `--cleanup`。清理始终进行，适用于所有更新：定时更新、容器的 `Update` 操作，以及更新面板。无需任何开关。

Dozzle 会保留容器之前运行的镜像，以便还能回退到它，并删除再之前的那个。从 1.4.1 更新到 1.4.2 会删除 1.4.0 并保留 1.4.1，因此每个容器最多保留一个备用镜像。Dozzle 从旧容器的 `dev.dozzle.previous-image` 标签读取要删除的镜像，所以容器的第一次更新不会删除任何镜像。

只有在更新完成且旧容器已删除之后才会清理。已回滚的更新不会删除任何镜像。Dozzle 只删除没有标签且没有容器使用的镜像：仍带有标签的镜像（例如你自己拉取或构建的镜像）会被保留，并且删除不强制，因此只要还有其他容器（无论运行中还是已停止）在使用它，Docker 就会拒绝删除。删除被拒绝不会导致更新失败。

[远程代理](/zh/guide/agent) 上的容器也以同样方式清理，Dozzle 自身的容器也是如此：新的 Dozzle 稳定运行后，[自更新](/zh/guide/setup-wizard#self-update) 的辅助容器会删除再之前的那个镜像。Swarm 服务（包括以 Swarm 服务运行的 Dozzle）不会被清理，因为每个节点保存自己的镜像，并且 Swarm 会自行清理任务历史。

## 回滚 {#rolling-back}

回滚更新是 [Dozzle Cloud](/zh/guide/dozzle-cloud) 的功能。Dozzle Cloud 会关注计划所做的每次更新，当新版本开始出现故障时，会提出回滚。随后 Dozzle 会把容器换回它之前运行的镜像，也就是其 `dev.dozzle.previous-image` 标签所指的镜像，方式与更新相同：当前容器会一直保留，直到之前的镜像稳定运行；如果没有稳定运行，就把当前容器放回去。设置和卷保持不变。回滚不会拉取任何镜像，因此如果之前的镜像已不在主机上，回滚会失败，容器保持原样。

回滚后的容器会带上 `dev.dozzle.rolled-back-from` 标签，记录它离开的那个镜像，自动更新计划会跳过它，直到它的标签指向更新的镜像。回滚稳定运行后，被回滚的镜像会像其他旧镜像一样被[清理](#cleaning-up-old-images)，因此只有在不再有标签指向它时才会被删除。

回滚适用于独立容器，也包括 [远程代理](/zh/guide/agent) 上的容器。Swarm 服务、Kubernetes 和 Dozzle 自身的容器不支持回滚。

## 日志视图中的更新

当 Dozzle 更新一个容器时（按计划、从 Dozzle 界面或从 Dozzle Cloud），新容器的日志会以一个标记开头，写明它从哪个镜像换到哪个镜像，以及是谁发起的更新。回滚和被撤销的更新也会被标记。连接了 Dozzle Cloud 时，标记还会显示 Dozzle Cloud 对这次更新的判断，并附有前往 Dozzle Cloud 的链接。Dozzle 只在内存中保存最近的更新，因此 Dozzle 重启后标记会消失。
