---
title: 容器操作
sourceHash: 7eab1f511f5f
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
- 私有仓库，因为 Dozzle 自己没有凭据
- Kubernetes，镜像的更新发布由集群负责

### 更新 Dozzle 自身

Dozzle 无法停止自己来就地更新，所以独立运行的 Dozzle 容器只会显示更新提示和一个指向发布说明的链接，而不是 `Update` 按钮。以 Swarm 服务方式运行的 Dozzle 则可以正常更新，因为更新交给了编排器。其他主机上的 Dozzle 代理是普通容器，和其他容器一样更新。
