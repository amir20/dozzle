---
title: OpenID Connect
sourceHash: 9dcc3df4a715
---

# <Icon icon="mdi:shield-account" inline /> OpenID Connect

使用 `--auth-provider oidc` 时，你的身份提供方就是用户数据库。这种模式下 Dozzle 从不读取 `users.yml`：用户是谁、拥有哪些角色、能看到哪些容器，全部来自 OpenID Connect 令牌。在 Keycloak、Authentik、Zitadel 或 Pocket ID 里添加一个用户，他就能登录；把他的角色移除，他就登不进来。

这和在 `simple` 验证方式下[让 `users.yml` 里的用户用 OIDC 登录](/zh/guide/authentication/oauth#使用-oidc-登录)是两回事。那种情况下提供方只负责证明你是谁，你能得到什么仍由 `users.yml` 决定。想手动列出每一个用户就选 `simple`，想让提供方掌管用户列表就选 `oidc`。

## 最小配置

在你的身份提供方那里把 Dozzle 注册为机密客户端，并把重定向 URI 设为：

```
https://your-dozzle-host/api/auth/callback
```

如果 Dozzle 部署在某个基础路径下，这里也要带上它，例如 `https://example.com/dozzle/api/auth/callback`。然后把 Dozzle 指向 issuer：

::: code-group

```sh [cli]
$ docker run -v /var/run/docker.sock:/var/run/docker.sock -v /path/to/dozzle/data:/data -p 8080:8080 amir20/dozzle --auth-provider oidc --auth-oidc-issuer https://keycloak.example.com/realms/main --auth-oidc-client-id dozzle --auth-oidc-client-secret secret
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /path/to/dozzle/data:/data
    ports:
      - 8080:8080
    environment:
      DOZZLE_AUTH_PROVIDER: oidc
      DOZZLE_AUTH_OIDC_ISSUER: https://keycloak.example.com/realms/main
      DOZZLE_AUTH_OIDC_CLIENT_ID: dozzle
      DOZZLE_AUTH_OIDC_CLIENT_SECRET: secret
```

:::

这就是全部配置。对常见的令牌布局来说不需要设置任何 claim 路径，`DOZZLE_AUTH_OIDC_NAME` 也只是改变登录按钮上的文字。client secret 同样接受一个 `_FILE` 形式的对应变量，参见[用 Docker secrets 保存 client secret](/zh/guide/authentication/oauth#用-docker-secrets-保存-client-secret)。

issuer URL 就是提供 `/.well-known/openid-configuration` 的那个地址。Dozzle 会拉取这份文档来找到授权端点、令牌端点和 userinfo 端点；如果文档里声明的 issuer 和你配置的不一致，Dozzle 会拒绝发起登录流程。

## 角色

用户的角色从令牌中读取。Dozzle 会按顺序尝试下面这些 claim，并取第一个存在的：

1. `dozzle_roles`
2. `resource_access.<client-id>.roles`
3. `roles`

client id 已经配置过了，所以当 `DOZZLE_AUTH_OIDC_CLIENT_ID=dozzle` 时，第二条路径就是 `resource_access.dozzle.roles`，这正是 Keycloak 存放客户端角色的地方。对这种布局不需要再设置任何东西。

如果你的角色放在别的地方，`DOZZLE_AUTH_OIDC_ROLES_CLAIM` 会用你给出的那一条点分路径取代整个搜索过程：

```yaml
DOZZLE_AUTH_OIDC_ROLES_CLAIM: realm_access.roles
```

每个 claim 都会先在 ID 令牌里查找，再到 userinfo 响应里查找，所以你的提供方把它放在两者中的哪一个并不重要。接受三种形式：字符串数组、用逗号或空格分隔的单个字符串，以及以角色名为键的对象，最后这种是 Zitadel 编码项目角色的方式。

角色名和 `users.yml` 里的一样：`shell`、`actions`、`download`、`notifications`、`cloud` 和 `all`，并且可以用 `^` 做排除，所以 `all,^shell` 授予除 shell 访问之外的全部权限。带 `dozzle_` 前缀的名称同样被接受，这在提供方用同一个角色 claim 服务多个应用时很有用。每个角色解锁什么，参见[角色](/zh/guide/authentication/simple#为用户设置角色)。

> [!WARNING]
> `groups` 是有意不在这个列表里的。在 Authentik 或 Google 那里，每个用户都至少属于一个组，搜索它会把"拒绝登录"变成"登录成功并能读取所有容器"。如果你手头只有用户组，请在提供方那里把它们映射成一个 `dozzle_roles` claim，参见下面的示例。

### 登录被拒绝时

当这些 claim 一个都不存在，或者第一个存在的 claim 为空时，登录会被拒绝。日志会列出尝试过的路径：

```
WRN OIDC login rejected: no roles claim found in the ID token or userinfo, or it was empty sub=... tried="dozzle_roles, resource_access.dozzle.roles, roles"
```

claim 存在但里面没有任何 Dozzle 能识别为角色的内容，则是另一种情况。这样的用户会以无权限的身份登录，和 `users.yml` 里的 `roles: none` 一样：他可以读取过滤器允许的那些容器的日志，仅此而已。Keycloak 的 realm 角色就是这种表现，因为那里的每个用户都带有 `offline_access` 和 `uma_authorization`，所以下面的示例改用客户端角色。

## 过滤器

容器过滤器的工作方式相同，从 `dozzle_filters`、`resource_access.<client-id>.filters` 和 `filters` 中第一个存在的读取，或者从 `DOZZLE_AUTH_OIDC_FILTERS_CLAIM` 指定的那一条路径读取。每个值就是一个过滤器，[语法和 `users.yml` 一样](/zh/guide/authentication/simple#为用户设置过滤器)，例如 `label=com.example.app` 或 `name=web`：

```json
"resource_access": {
  "dozzle": {
    "roles": ["shell", "actions"],
    "filters": ["label=com.example.app"]
  }
}
```

没有过滤器 claim 的用户可以看到这个 Dozzle 实例能看到的所有容器。无法解析的过滤器会让登录失败，而不是被丢弃，因此提供方那边的一个笔误不会悄悄扩大某人的可见范围。

## 身份

`sub` claim 是用户的稳定标识。它是 `/data` 下配置目录的键，所以即使用户名或邮箱在提供方那里发生了变化，设置也会跟着这个人走。菜单里显示的名字取 `name`，没有则依次回退到 `preferred_username`、`email`、`sub`。`email` 和 `picture` 用于生成头像，如果提供方发来了 `picture` URL，就直接使用它。

和 `simple` 验证方式不同，这里的邮箱不需要经过验证。它只用于显示，从不拿来和任何东西匹配，所以不授予 email scope 的 issuer 也能正常工作。

## 会话

登录之后，Dozzle 会签发自己的会话 cookie，其中带着从令牌里读到的角色和过滤器。它们在每个请求上都会生效，但不会被重新拉取：在提供方那里更改角色，要到用户下次登录才会生效。如果这个间隔对你很重要，把 [`--auth-ttl`](/zh/guide/supported-env-vars) 设成 `8h` 之类的值，让会话过期后从新的令牌重新建立。

## 登出

登出会清除 Dozzle 的会话。如果设置了 `--auth-logout-url`，浏览器接着会被送到那个地址，所以把它指向你的提供方的 end session URL，就能把用户从提供方那里也登出：

```yaml
DOZZLE_AUTH_LOGOUT_URL: https://keycloak.example.com/realms/main/protocol/openid-connect/logout
```

## 和 `simple` 有什么不同

两种验证方式共用 `--auth-oidc-*` 这组标志，所以区别体现在行为上：

- 从不读取 `users.yml`。如果 `/data` 下存在这个文件，Dozzle 会在日志里说明它被忽略了。
- 没有密码表单，也没有 `/api/token` 端点。身份提供方是唯一的入口，所以一个填错的回调 URL 或一个过期的 client secret 会把所有人挡在外面，直到问题被修复。
- `--auth-github-*` 会在启动时报错。GitHub 不是 OpenID Connect issuer，也不发布任何可以用来读取角色的 claim。
- 启动时会在日志里打印 issuer 以及将从哪些 claim 路径读取角色。

## 提供方示例

### Keycloak

在你的 realm 里创建一个名为 `dozzle` 的客户端，开启 client authentication，并添加上面的重定向 URI。然后在客户端的 **Roles** 标签页下创建你想分发的客户端角色：`shell`、`actions`、`download`、`notifications`、`cloud` 或 `all`。在 **Role mapping** 下把它们分配给用户或用户组。

Keycloak 以 `resource_access.<client-id>.roles` 的形式输出客户端角色，Dozzle 本来就会搜索这条路径。检查客户端的 **Client scopes**，打开专属 scope，确认 **client roles** 这个 mapper 会把该 claim 加进 ID 令牌或 userinfo；Dozzle 会读这两者，但不读 access token。

至于过滤器，添加一个名为 `dozzle_filters` 的用户属性，并在专属 scope 上添加一个 **User Attribute** mapper，令牌 claim 名称保持一致，并开启 **Multivalued**。每个值就是一个过滤器，例如 `label=com.example.app`。

### Authentik

在 **Customization** → **Property Mappings** 下添加一个 scope mapping，根据用户所属的组返回角色，然后把它挂到 Dozzle 的 provider 上：

```python
roles = []
if request.user.ak_groups.filter(name="dozzle-admins").exists():
    roles.append("all")
elif request.user.ak_groups.filter(name="dozzle-users").exists():
    roles.append("download")
return {"dozzle_roles": roles}
```

两个组都不属于的用户会得到一个空列表，并被拒绝登录。

### Zitadel

给用户授予以 Dozzle 角色命名的项目角色，并在应用上启用 **Assert Roles on Authentication**。Zitadel 的角色 claim 是一个以角色名为键的对象，Dozzle 能接受这种形式，所以设置：

```yaml
DOZZLE_AUTH_OIDC_ROLES_CLAIM: urn:zitadel:iam:org:project:roles
```

### Google

Google 的令牌不带任何角色 claim，所以 `oidc` 无法配合它使用。请改用[带 Google 登录的 `simple` 验证方式](/zh/guide/authentication/oauth#google)，由 `users.yml` 决定谁能进来。
