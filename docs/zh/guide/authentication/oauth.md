---
title: 使用 GitHub 与 OIDC 登录
sourceHash: eece13f5dae2
---

# <Icon icon="mdi:shield-account" inline /> 使用 GitHub 与 OIDC 登录

Dozzle 可以让用户用外部账号登录，而不必输入密码。这是 [`simple`](/zh/guide/authentication/simple) 验证方式的一部分，并不是一个独立的验证方式，所以 `users.yml` 仍然会在每个请求上被读取，也仍然由它决定谁能进来。

这里有一点值得先说清楚：**`users.yml` 就是白名单。** 没有任何条目关联到的外部账号无法登录，也不会自动创建任何账号。

密码登录会继续保留，这在 OAuth 应用出问题、你需要登录进去修复时很重要。

## 使用 GitHub 登录

Dozzle 可以让用户直接用自己的 GitHub 账号登录，而不必输入密码。这是 `simple` 验证方式的一部分，并不是一个独立的验证方式，所以 `users.yml` 仍然会在每个请求上被读取，也仍然由它决定谁能进来。继续使用 `--auth-provider simple` 即可。如果你更想在配置里写明这个实例用的是什么，`github` 也可以作为别名。

首先在 GitHub 的[开发者设置](https://github.com/settings/developers)里创建一个 OAuth App，把 **Authorization callback URL** 设为：

```
https://your-dozzle-host/api/auth/callback
```

如果 Dozzle 部署在某个[基础路径](/zh/guide/changing-base)下，这个 URL 里也要带上它，例如 `https://example.com/dozzle/api/auth/callback`。Dozzle 发起登录流程时不会传 `redirect_uri`，因此 GitHub 总是跳转到 OAuth App 上登记的那个回调 URL。这里对不上是登录失败最常见的原因。

然后复制 client ID，生成一个 client secret，把两者都传给 Dozzle：

::: code-group

```sh [cli]
$ docker run -v /var/run/docker.sock:/var/run/docker.sock -v /path/to/dozzle/data:/data -p 8080:8080 amir20/dozzle --auth-provider simple --auth-github-client-id Ov23liABCDEFGHIJKLMN --auth-github-client-secret 0123456789abcdef0123456789abcdef01234567
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
      DOZZLE_AUTH_PROVIDER: simple
      DOZZLE_AUTH_GITHUB_CLIENT_ID: Ov23liABCDEFGHIJKLMN
      DOZZLE_AUTH_GITHUB_CLIENT_SECRET: 0123456789abcdef0123456789abcdef01234567
```

:::

在 `users.yml` 里用 `github` 键把用户和他的 GitHub 账号关联起来：

```yaml
users:
  admin:
    email: me@email.net
    name: Admin
    password: $2a$11$9ho4vY2LdJ/WBopFcsAS0uORC0x2vuFHQgT/yBqZyzclhHsoaIkzK
    github: octocat

  guest:
    email: guest@email.net
    name: Guest
    github: hubot
    filter: "label=com.example.app"
    roles: none
```

设置了 `github` 之后 `password` 就是可选的，上面的 `guest` 就是这样。`admin` 两者都有，所以两种方式都能登录。对仍然设有密码的用户来说，密码登录会作为备用方式继续可用，登录页面上两种方式都会显示。

这里填的值是 GitHub 的**登录名**（`github.com/octocat` 中的那个用户名），不是邮箱地址。登录名是稳定的，而且始终可见，账号的邮箱则可能被设为私密或随时更改。

`users.yml` 就是白名单。没有列在 `users.yml` 里的 GitHub 账号无法登录，无论它属于哪个组织。这里也没有自动创建用户一说：要加人就得改这个文件。过滤器和角色和密码用户完全一样，每个请求都从 `users.yml` 解析，因此设置了 `roles: none` 的 GitHub 用户同样会被限制。

> [!NOTE]
> Dozzle 有意不支持把整个 GitHub 组织或整个邮箱域名加入白名单。每个用户都要单独列出。如果你需要基于用户组或域名的访问控制，请使用 `forward-proxy`，配合 [Authelia](/zh/guide/authentication/forward-proxy#配合-authelia-使用-dozzle) 或 Authentik，它们本来就是为此设计的。

> [!WARNING]
> 修改 `users.yml` 会轮换 JWT 签名密钥，并把所有用户登出。今天增删用户时就已经是这样了，添加 `github` 键同样如此。

## 使用 OIDC 登录

任何发布了 OpenID Connect 发现文档的身份提供方都能通过同一个回调地址接入：Google、Keycloak、Pocket ID、Zitadel、Authentik 等等。把 Dozzle 指向 issuer URL，再给它一个 client id 和 client secret 即可。

在你的身份提供方那里把 Dozzle 注册为机密客户端，并把重定向 URI 设为：

```
https://your-dozzle-host/api/auth/callback
```

如果 Dozzle 部署在某个基础路径下，这里也要带上它，例如 `https://example.com/dozzle/api/auth/callback`。和 GitHub 不同，OIDC 要求 Dozzle 发送 `redirect_uri`，所以这个值必须和你登记的完全一致。

::: code-group

```sh [cli]
$ docker run -v /var/run/docker.sock:/var/run/docker.sock -v /path/to/dozzle/data:/data -p 8080:8080 amir20/dozzle --auth-provider simple --auth-oidc-issuer https://id.example.com --auth-oidc-client-id dozzle --auth-oidc-client-secret secret --auth-oidc-name "Pocket ID"
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
      DOZZLE_AUTH_PROVIDER: simple
      DOZZLE_AUTH_OIDC_ISSUER: https://id.example.com
      DOZZLE_AUTH_OIDC_CLIENT_ID: dozzle
      DOZZLE_AUTH_OIDC_CLIENT_SECRET: secret
      DOZZLE_AUTH_OIDC_NAME: Pocket ID
```

:::

`DOZZLE_AUTH_OIDC_NAME` 只是登录按钮上的文字，默认值是 `SSO`。

issuer URL 就是提供 `/.well-known/openid-configuration` 的那个地址。Dozzle 会拉取这份文档来找到授权端点、令牌端点和 userinfo 端点；如果文档里声明的 issuer 和你配置的不一致，Dozzle 会拒绝发起登录流程。

### 关联用户

OIDC 按**已验证的邮箱**匹配，而不是登录名。请在 `users.yml` 里给用户设置 `email`：

```yaml
users:
  admin:
    email: me@email.net
    name: Admin
    # 账号关联之后 password 就是可选的
```

这个邮箱必须被你的身份提供方标记为已验证。如果 `email_verified` 为 false，Dozzle 会拒绝登录，因为匹配一个未验证的地址，意味着任何人只要能在一个宽松的提供方那里注册，填上别人的邮箱就能冒用账号。

> [!NOTE]
> GitHub 按登录名匹配，OIDC 按邮箱匹配，这个区别是有意为之。GitHub 的登录名稳定且始终存在，而 GitHub 邮箱可能被设为私密或被更改。OIDC 没有对应的、稳定又便于阅读的标识，因此已验证的邮箱才是运维人员真正掌握的那个 claim。

### Google

Google 就是一个普通的 OIDC 提供方。在 Google Cloud 控制台里创建一个 OAuth 客户端，然后使用：

```
DOZZLE_AUTH_OIDC_ISSUER: https://accounts.google.com
DOZZLE_AUTH_OIDC_NAME: Google
```

`--auth-provider google` 会被当作 `simple` 的别名，因此两种写法都可以。

## 用 Docker secrets 保存 client secret

把 client secret 直接写在 `environment:` 里，意味着它会出现在 `docker inspect` 的输出里、你的 compose 文件里，以及任何手动启动过容器的人的 shell 历史里。所以两个 client secret 都接受一个 `_FILE` 形式的对应变量，用来指明从哪个文件读取这个值，这也是 Docker 官方镜像一贯的做法：

| 不要用                             | 改用                                    |
| ---------------------------------- | --------------------------------------- |
| `DOZZLE_AUTH_GITHUB_CLIENT_SECRET` | `DOZZLE_AUTH_GITHUB_CLIENT_SECRET_FILE` |
| `DOZZLE_AUTH_OIDC_CLIENT_SECRET`   | `DOZZLE_AUTH_OIDC_CLIENT_SECRET_FILE`   |

Dozzle 会在启动时读取这个文件，并去掉首尾的空白字符，所以 `echo secret > file` 留下的行尾换行不会有问题。同时设置一个变量和它的 `_FILE` 对应变量会直接报错，而不是悄悄地只认其中一个；把 `_FILE` 指向一个不存在或者内容为空的文件，Dozzle 会在启动时停下来，而不是不声不响地把登录按钮去掉。

### Docker Compose

在 Swarm 之外没有 `docker secret create`，所以 Compose 的 secret 要么是磁盘上的一个文件，要么是一个环境变量。通常你想要的是环境变量这种形式：它可以配合一个已被 gitignore 的 `.env` 使用，也不会在 compose 文件旁边留下一个明文文件等着被提交进仓库。

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    ports:
      - 8080:8080
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./data:/data
    environment:
      DOZZLE_AUTH_PROVIDER: simple
      DOZZLE_AUTH_GITHUB_CLIENT_ID: Ov23liABCDEFGHIJKLMN
      DOZZLE_AUTH_GITHUB_CLIENT_SECRET_FILE: /run/secrets/dozzle_github_secret
    secrets:
      - dozzle_github_secret

secrets:
  dozzle_github_secret:
    environment: GITHUB_CLIENT_SECRET
```

```ini [.env]
GITHUB_CLIENT_SECRET=your-github-client-secret
```

Compose 会自己读取这个变量，并把它的值挂载到 `/run/secrets/dozzle_github_secret`。这个值不会成为容器环境变量的一部分，所以它和基于文件的 secret 一样，不会出现在 `docker inspect` 里。

如果这个 secret 本来就已经是一个文件，比如由某个 secret 管理工具写出来的文件，那就改用文件形式：

```yaml [docker-compose.yml]
secrets:
  dozzle_github_secret:
    file: /run/secrets/github_client_secret
```

两种写法下 Dozzle 都会去掉首尾的空白字符，所以文件末尾有没有换行都无所谓。

### Docker Swarm

在 Swarm 里，secret 由集群管理，而不是磁盘上的某个文件，所以要把它声明为 `external`，并用 `docker secret create` 创建：

```sh
printf '%s' 'your-oidc-client-secret' | docker secret create dozzle_oidc_secret -
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      DOZZLE_MODE: swarm
      DOZZLE_AUTH_PROVIDER: simple
      DOZZLE_AUTH_OIDC_ISSUER: https://id.example.com
      DOZZLE_AUTH_OIDC_CLIENT_ID: dozzle
      DOZZLE_AUTH_OIDC_CLIENT_SECRET_FILE: /run/secrets/dozzle_oidc_secret
      DOZZLE_AUTH_OIDC_NAME: Pocket ID
    secrets:
      - dozzle_oidc_secret
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    deploy:
      mode: global

secrets:
  dozzle_oidc_secret:
    external: true
```

> [!NOTE]
> secret 默认挂载在 `/run/secrets/<name>`，`_FILE` 指向的就是这个路径。如果你显式设置了 `target:`，就把 `_FILE` 指向你设置的那个路径。

### 确认是否生效

Dozzle 会在启动时把已启用的登录方式打进日志。加上 `--level debug` 运行，然后找那行写着登录方式名字的日志：

```sh
$ docker compose logs dozzle | grep -i 'sign in'
DBG Enabling Sign in with GitHub
```

如果 secret 文件不存在或者是空的，Dozzle 会在启动时退出，并在提示信息里指出是哪个变量，这样挂载出了问题会明确报错，而不是悄无声息地少掉一个登录按钮。
