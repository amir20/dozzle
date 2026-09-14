---
title: 设置向导
sourceHash: 4a73b46d7ba8
---

# 设置向导

<Badge type="warning" text="Docker Only" />

全新安装的 Dozzle 会先打开一个简短的设置向导。它会带你完成大多数人在安装后马上要改的几件事：开启登录、允许容器操作和终端访问，以及连接 Dozzle Cloud。它保存的所有内容也都可以通过命令行参数或环境变量设置，所以向导是可选的。

向导只会出现在以服务器模式运行的全新安装中。Swarm 和 Kubernetes 部署永远不会显示它。之后你可以在设置中再次打开它。

## <Icon icon="mdi:format-list-numbered" inline /> 步骤

### 1. 登录

登录是第一步，这样在任何人都能访问的实例上，其他设置都无法被修改。

向导首先检查 `/data` 是否挂载在卷上。设置和用户都保存在这里，没有卷的话，下次重新创建容器时它们就会丢失。如果 `/data` 没有持久化，向导会说明如何挂载，并等待你点击 **重新检查**。

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - dozzle_data:/data
    ports:
      - 8080:8080
volumes:
  dozzle_data:
```

`/data` 持久化之后，从三个选项中选择一个：

- **Dozzle 账户** 创建一个用户，包含用户名、可选的邮箱和密码。Dozzle 会写入 `/data/users.yml` 并设置 `authProvider: simple`。之后要添加更多用户或角色，请参阅 [简单认证](/zh/guide/authentication/simple)。
- **我的代理** 适用于 Authelia、Authentik、Cloudflare Access 等。Dozzle 信任 `Remote-User` 请求头，所以只发布代理的端口，永远不要发布 Dozzle 自己的端口。这会设置 `authProvider: forward-proxy`。请参阅 [前置代理](/zh/guide/authentication/forward-proxy)。
- **OIDC** 显示 [OpenID Connect](/zh/guide/authentication/oidc) 指南的链接和需要添加的环境变量。OIDC 需要客户端密钥，所以这里不会写入任何内容，由你自行配置。

如果 Dozzle 只能在你自己的网络中访问，**不设置登录，继续** 会跳过此步骤。

保存账户或代理后，Dozzle 会立即重启，确保在修改其他任何设置之前登录已经生效。你会进入登录页面，登录后向导会从下一步继续。

### 2. 操作与终端

两个开关控制 Dozzle 可以对你的容器做什么：

- **启动、停止和重启** 开启 [容器操作](/zh/guide/actions)（`enableActions`）。
- **终端** 开启在容器内 [附加并执行命令](/zh/guide/shell)（`enableShell`）。默认关闭。容器的终端访问往往相当于主机的访问权限，所以只在需要时开启。

如果某个设置已经由命令行参数或环境变量固定，它的开关会是只读的，并给出说明。和登录一样，这些开关需要 `/data` 挂载在卷上，在此之前会保持只读。

### 3. Dozzle Cloud

[Dozzle Cloud](/zh/guide/dozzle-cloud) 会在出现故障的第一时间发送告警，每天早上发送一份待修复问题的摘要，并保留重启后依然存在的历史记录。**连接 Dozzle Cloud** 会关联此实例，**暂不** 则继续下一步。如果实例已经关联，或者你没有权限关联，此步骤会被跳过。

### 4. 自动更新

Dozzle 可以让自己保持最新。选择 **关闭**、**每天** 或 **每周**（每周在周日运行），再选择一天中的时间。时间使用服务器的本地时间，默认是 `03:00`。到了这个时间，Dozzle 会检查镜像仓库中是否有更新的镜像，只有在有新镜像时才会 [更新自身](#self-update)。

此设置立即生效，不需要重启。

更新自身属于操作功能，所以在操作关闭时，这一步仍会留在列表中，但显示为灰色并标注 **需要操作功能**。在第 2 步开启操作后，它会立即变为可用。如果此实例因为其他原因无法更新自身（例如运行的是固定版本标签），这一步会改为说明原因。

### 5. 重启

最后一步列出已保存但尚未生效的更改。**重启 Dozzle** 会重启容器，等待它恢复后重新加载页面。如果没有待处理的更改，这一步只会提示你已完成。

如果 Dozzle 无法自行重启（例如找不到自己的容器），向导会改为显示可以添加到 compose 文件中的环境变量。

## <Icon icon="mdi:file-cog-outline" inline /> 设置保存在哪里

向导会把你的选择保存到 `/data/dozzle.yml`。Dozzle 只在启动时读取一次这个文件，所以更改需要重启才能生效。Dozzle 会从向导中自行重启，你不需要手动操作。自动更新相关的键是例外：Dozzle 每分钟都会重新读取它们，所以无需重启即可生效。

```yaml [/data/dozzle.yml]
authProvider: simple
enableActions: true
enableShell: false
autoUpdate: weekly
autoUpdateTime: "03:00"
```

| 键               | 取值                              | 等同于                    |
| ---------------- | --------------------------------- | ------------------------- |
| `authProvider`   | `none`, `simple`, `forward-proxy` | `DOZZLE_AUTH_PROVIDER`    |
| `enableActions`  | `true`, `false`                   | `DOZZLE_ENABLE_ACTIONS`   |
| `enableShell`    | `true`, `false`                   | `DOZZLE_ENABLE_SHELL`     |
| `autoUpdate`     | `off`, `daily`, `weekly`          | `DOZZLE_AUTO_UPDATE`      |
| `autoUpdateTime` | `HH:MM`，服务器本地时间           | `DOZZLE_AUTO_UPDATE_TIME` |

命令行参数和环境变量始终优先于该文件。如果设置了 `DOZZLE_ENABLE_ACTIONS`，`dozzle.yml` 中的值会被忽略，向导会将该开关显示为锁定。若想重新通过向导管理某个设置，请从 compose 文件中删除对应的变量。

## <Icon icon="mdi:update" inline /> 自更新的工作原理 {#self-update}

Dozzle 可以通过自身容器上的 `Update` 操作更新自己，也可以按自动更新计划进行。两者做的事情相同：

1. Dozzle 拉取它正在运行的镜像标签。如果该标签仍然指向正在运行的镜像，就到此为止，并报告已是最新版本。
2. Dozzle 用新镜像启动一个短暂存在的辅助容器，它可以访问同一个 Docker socket。几秒钟后 Dozzle 会退出。
3. 辅助容器重命名旧容器，并以原来的名称创建一个替代容器，使用相同的配置、网络和卷。之后才停止旧容器并启动替代容器。匿名卷也会保留，所以即使没有命名卷，`/data` 中的数据也不会丢失。
4. 辅助容器等待替代容器保持运行（如果设置了健康检查，还要保持健康）。如果成功，旧容器会被删除，但它的卷保持不动。如果失败，替代容器会被删除，旧容器会改回原来的名称并重新启动。

用 `--rm` 启动的容器也以同样的方式更新。旧容器停止时会自行删除，但此时替代容器已经持有它的卷，所以卷会保留下来。如果更新需要回滚，辅助容器会根据保存的配置重新创建旧容器。

辅助容器的日志是更新过程的唯一记录。它完成后会自行删除，所以要跟踪一次更新，请在它运行期间查看 `dozzle-self-update-*` 容器。

有些部署方式无法通过这种方式更新：

- **必须开启操作。** 自更新需要 `DOZZLE_ENABLE_ACTIONS`，开启登录时，`Update` 操作还需要 actions 角色。
- **服务器模式，包括以 Swarm 服务运行。** 当 Dozzle 作为 Swarm 服务的任务运行时，不会使用辅助容器：Dozzle 会请求 Swarm manager 把服务滚动更新到新镜像，并沿用 Swarm 自己的更新和回滚设置。这要求 Dozzle 运行在 manager 节点上。有多个副本时，只有第一个副本会执行计划任务。Kubernetes 和 Dozzle 代理不会自行更新。
- **固定版本标签永远不会更新。** 拉取 `amir20/dozzle:v8.12.0` 总是得到同一个镜像，所以自动更新不可用，手动更新会报告已是最新版本。请使用 `latest`，或者自己修改标签。

## <Icon icon="mdi:shield-lock-outline" inline /> 安全

- **登录是第一步。** 保存账户或代理后的重启会先开启登录，之后才能修改其他设置。
- **只有已登录的用户才能修改操作、终端和自动更新设置或重启 Dozzle。** 该用户需要拥有全部角色。
- **没有登录时，只有全新安装才有 15 分钟的时间窗口。** 当 `authProvider` 为 `none` 时，这些设置只能在全新安装（`/data` 为空）首次启动后的 15 分钟内修改。已经有之前运行留下数据的安装永远不会获得这个窗口，因此重启主机或更新镜像都无法打开它。窗口之外，请使用环境变量或开启登录。
- **路由仍然在启动时决定。** 向导只会写入 `dozzle.yml`。操作和终端的接口在 Dozzle 启动时注册，与使用环境变量时完全相同，所以在 Dozzle 重启之前不会启用任何功能。
