---
title: 身份验证
sourceHash: c0e3f963afbe
---

# 身份验证

Dozzle 支持两种身份验证配置方式。第一种是自带验证方案，通过代理来保护 Dozzle。Dozzle 开箱即可读取相应的请求头。

如果你没有现成的验证方案，Dozzle 也提供了一套简单的、基于文件的用户管理方案。验证方式通过 `--auth-provider` 参数设置。两种配置下，Dozzle 都会尝试把用户设置写入磁盘，数据写在 `/data`。

## <Icon icon="mdi:shield-alert-outline" inline /> 安全注意事项

Dozzle 可以访问 `docker.sock`，除非加以限制，否则这等同于**主机上的 root 权限**。在把 Dozzle 暴露到私有网络之外前，请先检查以下几点：

- 如果 Dozzle 可以从公网访问，**务必给它加上身份验证**。使用 `--auth-provider=simple`，或者 Authelia / Authentik / Cloudflare Access 这类前置代理。
- 除非确实需要，否则**保持[操作](/zh/guide/actions)和[终端访问](/zh/guide/shell)处于关闭状态**。它们允许启动、停止、重建容器，以及在容器内执行任意命令。
- 在多用户模式下，用[角色](/zh/guide/authentication/simple#为用户设置角色)和[过滤器](/zh/guide/authentication/simple#为用户设置过滤器)**限制用户权限**。如果不显式设置角色，用户能看到 Dozzle 实例能看到的每一个容器。
- **在前置代理模式下，绝不要直接暴露 Dozzle 的端口。** Dozzle 会信任每个请求上的 `Remote-User`，而当请求中没有角色头时，该用户会被授予全部角色。任何能绕过代理直接访问容器的人，只要设置一个请求头就能以任意身份登录。只对外发布代理，把 Dozzle 放在内部网络上，用 `expose` 而不是 `ports`。
- **在反向代理上启用 TLS**。Nginx / Traefik / Caddy 的示例见[反向代理与基础路径](/zh/guide/changing-base)。
- 如果你不需要操作功能，**用 socket proxy 限制对 `docker.sock` 的访问**。注意，只读挂载（`/var/run/docker.sock:/var/run/docker.sock:ro`）_并不_限制 API：`:ro` 只是把磁盘上的 socket 文件标记为只读，API 调用照样能通过这个 socket，创建、删除、更新依然可行。要真正限制操作，请在 daemon 前面放一个 socket proxy，例如 [`tecnativa/docker-socket-proxy`](https://github.com/Tecnativa/docker-socket-proxy)。

## <Icon icon="mdi:key-outline" inline /> 选择验证方式

| 验证方式                                           | 用户由谁管理                | 适用场景                                                                                                             |
| -------------------------------------------------- | --------------------------- | -------------------------------------------------------------------------------------------------------------------- |
| [简单模式](/zh/guide/authentication/simple)        | Dozzle，写在 `users.yml` 里 | 你没有现成的验证方案，希望由 Dozzle 来处理登录。                                                                     |
| [GitHub 与 OIDC](/zh/guide/authentication/oauth)   | Dozzle，写在 `users.yml` 里 | 你希望 `users.yml` 里的同一批用户用 GitHub、Google、Keycloak、Pocket ID、Zitadel 或 Authentik 登录，而不是输入密码。 |
| [前置代理](/zh/guide/authentication/forward-proxy) | 你的代理                    | 你已经在运行 Authelia、Authentik、Cloudflare Access 之类的服务，并且希望完全由它来负责身份验证。                     |

简单模式和 OAuth 其实是同一种验证方式：两者的用户列表都是 `users.yml`，OAuth 只是多提供了一种方式来证明你是列表里的某个用户。前置代理才是另一类，当你需要基于组织或域名的访问规则时它才是正确的选择，而这正是 `users.yml` 有意不做的事。

## <Icon icon="mdi:file-document-edit-outline" inline /> 生成 users.yml

Dozzle 内置了 `generate` 命令来生成 `users.yml`。示例如下：

```sh
docker run -it --rm amir20/dozzle generate admin --password password --email test@email.net --name "John Doe" --user-filter name=foo --user-roles shell > users.yml
```

这个例子中 `admin` 是用户名。email 和 name 是可选的，但建议填写，这样头像才准确。`docker run -it --rm amir20/dozzle generate --help` 会列出所有选项。`--user-filter` 参数是以逗号分隔的过滤器列表，`--user-roles` 参数是以逗号分隔的角色列表。

如果省略 `--password`，Dozzle 会在 stdin 上提示你输入，这样密码就不会留在 shell 历史里。这需要交互式终端，所以要保留 `-it` 参数：

```sh
docker run -it --rm amir20/dozzle generate admin --email test@email.net --name "John Doe" > users.yml
```

提示信息写到 stderr，因此把 stdout 重定向到 `users.yml` 依然可行。你也可以用管道传入密码，例如 `echo "$PASSWORD" | docker run -i --rm amir20/dozzle generate admin > users.yml`。
