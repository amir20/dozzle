---
title: 设置向导
sourceHash: 33f53b1244a8
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

### 4. 重启

最后一步列出已保存但尚未生效的更改。**重启 Dozzle** 会重启容器，等待它恢复后重新加载页面。如果没有待处理的更改，这一步只会提示你已完成。

如果 Dozzle 无法自行重启（例如找不到自己的容器），向导会改为显示可以添加到 compose 文件中的环境变量。

## <Icon icon="mdi:file-cog-outline" inline /> 设置保存在哪里

向导会把你的选择保存到 `/data/dozzle.yml`。Dozzle 只在启动时读取一次这个文件，所以更改需要重启才能生效。Dozzle 会从向导中自行重启，你不需要手动操作。

```yaml [/data/dozzle.yml]
authProvider: simple
enableActions: true
enableShell: false
```

| 键              | 取值                              | 等同于                  |
| --------------- | --------------------------------- | ----------------------- |
| `authProvider`  | `none`, `simple`, `forward-proxy` | `DOZZLE_AUTH_PROVIDER`  |
| `enableActions` | `true`, `false`                   | `DOZZLE_ENABLE_ACTIONS` |
| `enableShell`   | `true`, `false`                   | `DOZZLE_ENABLE_SHELL`   |

命令行参数和环境变量始终优先于该文件。如果设置了 `DOZZLE_ENABLE_ACTIONS`，`dozzle.yml` 中的值会被忽略，向导会将该开关显示为锁定。若想重新通过向导管理某个设置，请从 compose 文件中删除对应的变量。

## <Icon icon="mdi:shield-lock-outline" inline /> 安全

- **登录是第一步。** 保存账户或代理后的重启会先开启登录，之后才能修改其他设置。
- **只有已登录的用户才能修改操作和终端设置或重启 Dozzle。** 该用户需要拥有全部角色。
- **没有登录时，有 15 分钟的时间窗口。** 当 `authProvider` 为 `none` 时，这些设置只能在 Dozzle 启动后的 15 分钟内修改。之后向导变为只读，直到你开启登录或重启 Dozzle。
- **路由仍然在启动时决定。** 向导只会写入 `dozzle.yml`。操作和终端的接口在 Dozzle 启动时注册，与使用环境变量时完全相同，所以在 Dozzle 重启之前不会启用任何功能。
