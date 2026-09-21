---
title: 环境变量与子命令
sourceHash: 94da032e7db0
---

# 环境变量

每个选项都可以通过标志或环境变量设置。标志和环境变量的优先级始终高于[设置向导](/zh/guide/setup-wizard)保存在 `dozzle.yml` 中的设置。

接受列表的选项（`DOZZLE_FILTER`、`DOZZLE_REMOTE_HOST`、`DOZZLE_REMOTE_AGENT`、`DOZZLE_NAMESPACE`）在环境变量中使用逗号分隔的值，或者每一项重复一次标志：

```sh
--remote-agent 167.99.1.1:7007 --remote-agent 167.99.1.2:7007
DOZZLE_REMOTE_AGENT=167.99.1.1:7007,167.99.1.2:7007
```

## 服务器

| 变量                                      | 说明                                                                      | 取值                                                              | 默认值   |
| ----------------------------------------- | ------------------------------------------------------------------------- | ----------------------------------------------------------------- | -------- |
| `DOZZLE_ADDR`<br>`--addr`                 | Web 服务器监听的地址和端口。在容器内很少需要设置。                        | `host:port`，例如 `:9090`                                         | `:8080`  |
| `DOZZLE_BASE`<br>`--base`                 | 提供 Dozzle 服务的路径前缀。参见[修改基础路径](/zh/guide/changing-base)。 | 路径，例如 `/logs`                                                | `/`      |
| `DOZZLE_HOSTNAME`<br>`--hostname`         | 此实例在界面中显示的名称。参见[主机名](/zh/guide/hostname)。              | 任意字符串                                                        | 无       |
| `DOZZLE_HOST_ID`<br>`--host-id`           | 覆盖 Dozzle 为此主机推导出的 id。只有在与其他主机冲突时才需要设置。       | 字母、数字、`_`、`.`、`-`                                         | 自动推导 |
| `DOZZLE_LEVEL`<br>`--level`               | Dozzle 自身的日志级别。参见[调试](/zh/guide/debugging)。                  | `trace`、`debug`、`info`、`warn`、`error`                         | `info`   |
| `DOZZLE_MODE`<br>`--mode`                 | 部署模式。                                                                | `server`、[`swarm`](/zh/guide/swarm-mode)、[`k8s`](/zh/guide/k8s) | `server` |
| `DOZZLE_TIMEOUT`<br>`--timeout`           | 调用 Docker 或 Kubernetes API 的超时时间。                                | 时长，例如 `30s`                                                  | `10s`    |
| `DOZZLE_NO_ANALYTICS`<br>`--no-analytics` | 关闭匿名[数据分析](/zh/guide/analytics)。                                 | `true`、`false`                                                   | `false`  |

## 容器与主机

| 变量                                      | 说明                                                                        | 取值                              | 默认值            |
| ----------------------------------------- | --------------------------------------------------------------------------- | --------------------------------- | ----------------- |
| `DOZZLE_FILTER`<br>`--filter`             | 只显示匹配 Docker 过滤条件的容器。参见[过滤器](/zh/guide/filters)。         | `key=value`，例如 `label=app=web` | 无                |
| `DOZZLE_REMOTE_AGENT`<br>`--remote-agent` | 要连接的[代理](/zh/guide/agent)，可选附带显示名称和分组。                   | `host:port[\|name[\|group]]`      | 无                |
| `DOZZLE_REMOTE_HOST`<br>`--remote-host`   | 通过 TCP 连接的 Docker 主机。参见[远程主机](/zh/guide/remote-hosts)。       | `tcp://host:port[\|label]`        | 无                |
| `DOZZLE_NAMESPACE`<br>`--namespace`       | 要监听的 Kubernetes 命名空间。仅在 `DOZZLE_MODE=k8s` 时使用。               | 命名空间名称                      | 全部              |
| `DOZZLE_CERT`<br>`--cert`                 | 与代理通信时使用的 TLS 证书。参见[自定义证书](/zh/guide/agent#自定义证书)。 | 文件路径                          | `dozzle_cert.pem` |
| `DOZZLE_KEY`<br>`--key`                   | 与代理通信时使用的 TLS 私钥。                                               | 文件路径                          | `dozzle_key.pem`  |

## 功能

| 变量                                                  | 说明                                                                                  | 取值                         | 默认值                              |
| ----------------------------------------------------- | ------------------------------------------------------------------------------------- | ---------------------------- | ----------------------------------- |
| `DOZZLE_ENABLE_ACTIONS`<br>`--enable-actions`         | 允许在界面中启动、停止、重启、删除和更新容器。参见[容器操作](/zh/guide/actions)。     | `true`、`false`              | `false`                             |
| `DOZZLE_ENABLE_SHELL`<br>`--enable-shell`             | 允许在界面中附加到容器或在容器内运行 shell。参见 [shell](/zh/guide/shell)。           | `true`、`false`              | `false`                             |
| `DOZZLE_ENABLE_MCP`<br>`--enable-mcp`                 | 为 LLM 客户端开放 [MCP](/zh/guide/mcp) 端点。                                         | `true`、`false`              | `false`                             |
| `DOZZLE_DISABLE_AVATARS`<br>`--disable-avatars`       | 启用身份验证时隐藏用户头像。                                                          | `true`、`false`              | `false`                             |
| `DOZZLE_RELEASE_CHECK_MODE`<br>`--release-check-mode` | Dozzle 是否检查自身的新版本。`manual` 只在你主动要求时检查。                          | `automatic`、`manual`        | `automatic`                         |
| `DOZZLE_IMAGE_CHECK_MODE`<br>`--image-check-mode`     | Dozzle 是否到镜像仓库检查更新的容器镜像。参见[更新检查](/zh/guide/actions#更新检查)。 | `automatic`、`manual`、`off` | 与 `DOZZLE_RELEASE_CHECK_MODE` 相同 |
| `DOZZLE_AUTO_UPDATE`<br>`--auto-update`               | 按计划更新 Dozzle 自身的容器。`weekly` 在周日运行。需要开启 `DOZZLE_ENABLE_ACTIONS`。 | `off`、`daily`、`weekly`     | `off`                               |
| `DOZZLE_AUTO_UPDATE_TIME`<br>`--auto-update-time`     | 每天运行自动更新的时间，按服务器本地时间计算。                                        | `HH:MM`，例如 `04:30`        | `03:00`                             |

## 身份验证

各个提供方的工作方式请参见[身份验证](/zh/guide/authentication)。

| 变量                                            | 说明                                                                                            | 取值                                               | 默认值    |
| ----------------------------------------------- | ----------------------------------------------------------------------------------------------- | -------------------------------------------------- | --------- |
| `DOZZLE_AUTH_PROVIDER`<br>`--auth-provider`     | 使用哪个身份验证提供方。`github` 和 `google` 是 `simple` 的别名。                               | `none`、`simple`、`oidc`、`forward-proxy`          | `none`    |
| `DOZZLE_AUTH_TTL`<br>`--auth-ttl`               | 登录的有效时长。`session` 表示关闭浏览器时登出。                                                | `session` 或时长，例如 `48h`（单位 `s`、`m`、`h`） | `session` |
| `DOZZLE_AUTH_LOGOUT_URL`<br>`--auth-logout-url` | 使用 `forward-proxy` 时，登出后将用户跳转到的地址。使用 `oidc` 时，它会覆盖 issuer 的登出端点。 | URL                                                | 无        |

### GitHub OAuth

为 `simple` 身份验证加上“使用 GitHub 登录”。参见 [OAuth](/zh/guide/authentication/oauth)。

| 变量                                                                | 说明                                | 取值   | 默认值 |
| ------------------------------------------------------------------- | ----------------------------------- | ------ | ------ |
| `DOZZLE_AUTH_GITHUB_CLIENT_ID`<br>`--auth-github-client-id`         | GitHub OAuth 应用的 client id。     | 字符串 | 无     |
| `DOZZLE_AUTH_GITHUB_CLIENT_SECRET`<br>`--auth-github-client-secret` | GitHub OAuth 应用的 client secret。 | 字符串 | 无     |

### OpenID Connect

与 `DOZZLE_AUTH_PROVIDER=oidc` 一起使用。参见 [OIDC](/zh/guide/authentication/oidc)。

| 变量                                                            | 说明                                                                                                                                                                 | 取值                  | 默认值 |
| --------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------- | ------ |
| `DOZZLE_AUTH_OIDC_ISSUER`<br>`--auth-oidc-issuer`               | 身份提供方的 issuer URL。                                                                                                                                            | URL                   | 无     |
| `DOZZLE_AUTH_OIDC_CLIENT_ID`<br>`--auth-oidc-client-id`         | 在提供方处注册的 client id。                                                                                                                                         | 字符串                | 无     |
| `DOZZLE_AUTH_OIDC_CLIENT_SECRET`<br>`--auth-oidc-client-secret` | 在提供方处注册的 client secret。                                                                                                                                     | 字符串                | 无     |
| `DOZZLE_AUTH_OIDC_NAME`<br>`--auth-oidc-name`                   | 登录按钮上的文字。                                                                                                                                                   | 任意字符串            | `SSO`  |
| `DOZZLE_AUTH_OIDC_ROLES_CLAIM`<br>`--auth-oidc-roles-claim`     | 读取角色的 claim。只有当默认搜索（`dozzle_roles`、`resource_access.<client-id>.roles`、`roles`）找不到时才需要设置。                                                 | 以点分隔的 claim 路径 | 无     |
| `DOZZLE_AUTH_OIDC_FILTERS_CLAIM`<br>`--auth-oidc-filters-claim` | 读取容器过滤条件的 claim。只有当默认搜索（`dozzle_filters`、`resource_access.<client-id>.filters`、`filters`）找不到时才需要设置。                                   | 以点分隔的 claim 路径 | 无     |
| `DOZZLE_AUTH_OIDC_SCOPES`<br>`--auth-oidc-scopes`               | 在 `openid`、`profile` 和 `email` 之外额外请求的 scope，用于那些只有在[请求了对应 scope](/zh/guide/authentication/oidc#请求额外的-scope) 时才会给出 claim 的提供方。 | 以逗号分隔的 scope    | 无     |

> [!TIP]
> `DOZZLE_AUTH_GITHUB_CLIENT_SECRET` 和 `DOZZLE_AUTH_OIDC_CLIENT_SECRET` 还接受一个 `_FILE` 形式的对应变量，用来指明从哪个文件读取这个值，方便配合 [Docker secrets](/zh/guide/authentication/oauth#用-docker-secrets-保存-client-secret) 使用。

### Forward Proxy

与 `DOZZLE_AUTH_PROVIDER=forward-proxy` 一起使用。每个变量指定代理设置的 HTTP 头名称。参见 [Forward Proxy](/zh/guide/authentication/forward-proxy)。

| 变量                                                  | 说明                           | 取值       | 默认值          |
| ----------------------------------------------------- | ------------------------------ | ---------- | --------------- |
| `DOZZLE_AUTH_HEADER_USER`<br>`--auth-header-user`     | 携带用户名的请求头。           | 请求头名称 | `Remote-User`   |
| `DOZZLE_AUTH_HEADER_EMAIL`<br>`--auth-header-email`   | 携带邮箱的请求头。             | 请求头名称 | `Remote-Email`  |
| `DOZZLE_AUTH_HEADER_NAME`<br>`--auth-header-name`     | 携带显示名称的请求头。         | 请求头名称 | `Remote-Name`   |
| `DOZZLE_AUTH_HEADER_FILTER`<br>`--auth-header-filter` | 携带用户容器过滤条件的请求头。 | 请求头名称 | `Remote-Filter` |
| `DOZZLE_AUTH_HEADER_ROLES`<br>`--auth-header-roles`   | 携带用户角色的请求头。         | 请求头名称 | `Remote-Roles`  |

## 子命令

### generate

为 [simple 身份验证](/zh/guide/authentication/simple)生成一条 `users.yml` 条目。第一个参数是用户名。

```sh
docker run -it --rm amir20/dozzle generate admin --password password --email test@email.net --name "John Doe" --user-filter name=foo --user-roles shell > users.yml
```

| 标志               | 说明                                                                   | 取值                                                                    |
| ------------------ | ---------------------------------------------------------------------- | ----------------------------------------------------------------------- |
| `--password`, `-p` | 用户的密码。                                                           | 字符串                                                                  |
| `--email`, `-e`    | 用户的邮箱，用于显示头像。                                             | 邮箱地址                                                                |
| `--name`, `-n`     | 用户的显示名称。                                                       | 字符串                                                                  |
| `--user-filter`    | 用户可以看到的容器。                                                   | 以逗号分隔的 `key=value` 过滤条件                                       |
| `--user-roles`     | 用户可以执行的操作。在角色前加 `^` 表示移除该角色，例如 `all,^shell`。 | `all`、`none`、`shell`、`actions`、`download`、`notifications`、`cloud` |

### agent

将 Dozzle 作为[代理](/zh/guide/agent)运行，供另一个 Dozzle 实例连接。

```sh
docker run -v /var/run/docker.sock:/var/run/docker.sock -p 7007:7007 amir20/dozzle agent
```

| 变量                                  | 说明                   | 取值        | 默认值  |
| ------------------------------------- | ---------------------- | ----------- | ------- |
| `DOZZLE_AGENT_ADDR`<br>`--agent-addr` | 代理监听的地址和端口。 | `host:port` | `:7007` |

### generate-certs

为代理连接生成唯一的证书和私钥，替代 Dozzle 自带的共享证书。参见[自定义证书](/zh/guide/agent#自定义证书)。

| 标志         | 说明             | 默认值            |
| ------------ | ---------------- | ----------------- |
| `--cert-out` | 证书的写入位置。 | `dozzle_cert.pem` |
| `--key-out`  | 私钥的写入位置。 | `dozzle_key.pem`  |
| `--force`    | 覆盖已有文件。   | `false`           |

### healthcheck

检查服务器或代理是否正常运行。由于会带来少量额外的 CPU 开销，镜像默认没有配置它。参见[健康检查](/zh/guide/healthcheck)。
