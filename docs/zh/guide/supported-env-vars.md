---
title: 环境变量与子命令
sourceHash: 3930d18cbbb4
---

# 全局环境变量

配置既可以通过标志完成，也可以通过环境变量完成。下表列出了所有支持的选项及其对应的环境变量。

| 标志                   | 环境变量                    | 默认值            |
| ---------------------- | --------------------------- | ----------------- |
| `--addr`               | `DOZZLE_ADDR`               | `:8080`           |
| `--base`               | `DOZZLE_BASE`               | `/`               |
| `--hostname`           | `DOZZLE_HOSTNAME`           | `""`              |
| `--level`              | `DOZZLE_LEVEL`              | `info`            |
| `--auth-provider`      | `DOZZLE_AUTH_PROVIDER`      | `none`            |
| `--auth-header-user`   | `DOZZLE_AUTH_HEADER_USER`   | `Remote-User`     |
| `--auth-header-email`  | `DOZZLE_AUTH_HEADER_EMAIL`  | `Remote-Email`    |
| `--auth-header-name`   | `DOZZLE_AUTH_HEADER_NAME`   | `Remote-Name`     |
| `--auth-header-filter` | `DOZZLE_AUTH_HEADER_FILTER` | `Remote-Filter`   |
| `--auth-header-roles`  | `DOZZLE_AUTH_HEADER_ROLES`  | `Remote-Roles`    |
| `--auth-logout-url`    | `DOZZLE_AUTH_LOGOUT_URL`    | `""`              |
| `--auth-ttl`           | `DOZZLE_AUTH_TTL`           | `session`         |
| `--enable-actions`     | `DOZZLE_ENABLE_ACTIONS`     | `false`           |
| `--enable-shell`       | `DOZZLE_ENABLE_SHELL`       | `false`           |
| `--enable-mcp`         | `DOZZLE_ENABLE_MCP`         | `false`           |
| `--disable-avatars`    | `DOZZLE_DISABLE_AVATARS`    | `false`           |
| `--filter`             | `DOZZLE_FILTER`             | `""`              |
| `--no-analytics`       | `DOZZLE_NO_ANALYTICS`       | `false`           |
| `--mode`               | `DOZZLE_MODE`               | `server`          |
| `--release-check-mode` | `DOZZLE_RELEASE_CHECK_MODE` | `automatic`       |
| `--image-check-mode`   | `DOZZLE_IMAGE_CHECK_MODE`   | 继承              |
| `--remote-host`        | `DOZZLE_REMOTE_HOST`        |                   |
| `--remote-agent`       | `DOZZLE_REMOTE_AGENT`       |                   |
| `--timeout`            | `DOZZLE_TIMEOUT`            | `10s`             |
| `--namespace`          | `DOZZLE_NAMESPACE`          | `""`              |
| `--cert`               | `DOZZLE_CERT`               | `dozzle_cert.pem` |
| `--key`                | `DOZZLE_KEY`                | `dozzle_key.pem`  |

> [!TIP]
> 有些标志（例如 `--remote-host` 或 `--remote-agent`）可以多次使用。例如 `--remote-agent 167.99.1.1:7007 --remote-agent 167.99.1.2:7007`，或者用逗号分隔的 `DOZZLE_REMOTE_AGENT=167.99.1.1:7007,167.99.1.2:7007`。

## 生成 users.yml

Dozzle 支持生成 `users.yml` 文件。该文件用于验证用户身份。示例如下：

```sh
docker run -it --rm amir20/dozzle generate admin --password password --email test@email.net --name "John Doe" --user-filter name=foo --user-roles shell > users.yml
```

在这个例子中，`admin` 是用户名。邮箱和姓名是可选的，但建议填写，以便正确显示头像。运行 `docker run amir20/dozzle generate --help` 可以查看所有选项。

| 标志            | 说明           | 默认值 |
| --------------- | -------------- | ------ |
| `--password`    | 用户的密码     |        |
| `--email`       | 用户的邮箱     |        |
| `--name`        | 用户的全名     |        |
| `--user-filter` | 用户的过滤条件 |        |
| `--user-roles`  | 用户的角色     |        |

更多信息请参见[身份验证](/zh/guide/authentication)。

## 代理模式

Dozzle 支持以代理模式运行。当 Dozzle 运行在远程主机上、而你想监控另一台 Docker 主机时，代理模式会很有用。代理模式通过设置 `--remote-agent` 标志启用。示例如下：

```sh
docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --remote-agent remote-ip:7007
```

| 标志     | 环境变量            | 默认值  |
| -------- | ------------------- | ------- |
| `--addr` | `DOZZLE_AGENT_ADDR` | `:7007` |

更多信息请参见[代理](/zh/guide/agent)。

## 健康检查

Dozzle 支持通过 `dozzle healthcheck` 命令进行健康检查。由于会带来额外的 CPU 开销，它默认不启用。要使用 `healthcheck`，你需要自行配置。

更多信息请参见[健康检查](/zh/guide/healthcheck)。
