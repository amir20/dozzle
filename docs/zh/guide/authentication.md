---
title: 身份验证
sourceHash: 4ce1bd2c3816
---

# 身份验证

Dozzle 支持两种身份验证配置方式。第一种是自带验证方案，通过代理来保护 Dozzle。Dozzle 开箱即可读取相应的请求头。

如果你没有现成的验证方案，Dozzle 也提供了一套简单的、基于文件的用户管理方案。验证方式通过 `--auth-provider` 参数设置。两种配置下，Dozzle 都会尝试把用户设置写入磁盘，数据写在 `/data`。

## <Icon icon="mdi:shield-alert-outline" inline /> 安全注意事项

Dozzle 可以访问 `docker.sock`，除非加以限制，否则这等同于**主机上的 root 权限**。在把 Dozzle 暴露到私有网络之外前，请先检查以下几点：

- 如果 Dozzle 可以从公网访问，**务必给它加上身份验证**。使用 `--auth-provider=simple`，或者 Authelia / Authentik / Cloudflare Access 这类前置代理。
- 除非确实需要，否则**保持[操作](/zh/guide/actions)和[终端访问](/zh/guide/shell)处于关闭状态**。它们允许启动、停止、重建容器，以及在容器内执行任意命令。
- 在多用户模式下，用[角色](#setting-specific-roles-for-users)和[过滤器](#setting-specific-filters-for-users)**限制用户权限**。如果不显式设置角色，用户能看到 Dozzle 实例能看到的每一个容器。
- **在前置代理模式下，绝不要直接暴露 Dozzle 的端口。** Dozzle 会信任每个请求上的 `Remote-User`，而当请求中没有角色头时，该用户会被授予全部角色。任何能绕过代理直接访问容器的人，只要设置一个请求头就能以任意身份登录。只对外发布代理，把 Dozzle 放在内部网络上，用 `expose` 而不是 `ports`。
- **在反向代理上启用 TLS**。Nginx / Traefik / Caddy 的示例见[反向代理与基础路径](/zh/guide/changing-base)。
- 如果你不需要操作功能，**用 socket proxy 限制对 `docker.sock` 的访问**。注意，只读挂载（`/var/run/docker.sock:/var/run/docker.sock:ro`）_并不_限制 API：`:ro` 只是把磁盘上的 socket 文件标记为只读，API 调用照样能通过这个 socket，创建、删除、更新依然可行。要真正限制操作，请在 daemon 前面放一个 socket proxy，例如 [`tecnativa/docker-socket-proxy`](https://github.com/Tecnativa/docker-socket-proxy)。

## <Icon icon="mdi:account-cog-outline" inline /> 基于文件的用户管理

把 `--auth-provider` 设为 `simple` 即可启用多用户验证。在这种模式下，Dozzle 会尝试从 `/data/` 读取用户文件，如果 `users.yml` 和 `users.yaml` 都存在，优先使用 `users.yml`。如果只存在其中一个，就使用那一个。日志会显示实际读取的是哪个文件（例如 `Reading users.yml file`）。

### 文件路径示例：

- `/data/users.yml`
- `/data/users.yaml`

文件内容大致如下：

```yaml
users:
  # "admin" 是用户名
  admin:
    email: me@email.net
    name: Admin
    # 用 docker run -it --rm amir20/dozzle generate admin --password password --email me@email.net --name "Admin" 生成
    password: $2a$11$9ho4vY2LdJ/WBopFcsAS0uORC0x2vuFHQgT/yBqZyzclhHsoaIkzK
    filter:
    roles:
```

Dozzle 用 `email` 通过 [Gravatar](https://gravatar.com/) 生成头像，这一项是可选的。密码使用 `bcrypt` 哈希，可以用 `docker run amir20/dozzle generate` 生成。

> [!WARNING]
> 不再支持 SHA-256 密码哈希。旧版 Dozzle 用 SHA-256 哈希密码，`users.yml` 中若仍留有这类哈希，加载时不会报错，但该用户一登录进程就会退出。升级前请用 `generate` 重新生成所有密码。详情见[这条安全公告](https://github.com/amir20/dozzle/security/advisories/GHSA-w7qr-q9fh-fj35)。

你需要把这个文件挂载进去，Dozzle 才能找到它。示例如下：

::: code-group

```sh [cli]
$ docker run -v /var/run/docker.sock:/var/run/docker.sock -v /path/to/dozzle/data:/data -p 8080:8080 amir20/dozzle --auth-provider simple
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
```

```yaml [users.yml]
users:
  admin:
    email: me@email.net
    name: Admin
    password: $2a$11$9ho4vY2LdJ/WBopFcsAS0uORC0x2vuFHQgT/yBqZyzclhHsoaIkzK
```

:::

或者使用 Docker secrets：

```yaml
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      - DOZZLE_AUTH_PROVIDER=simple
    secrets:
      - source: users
        target: /data/users.yml
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - dozzle:/data
secrets:
  users:
    file: users.yml
volumes:
  dozzle:
```

### 延长验证 Cookie 的有效期

默认情况下，Dozzle 使用会话 cookie，浏览器关闭后即失效。你可以把 `--auth-ttl` 设为一个时长来延长 cookie 的有效期。示例如下：

::: code-group

```sh [cli]
$ docker run -v /var/run/docker.sock:/var/run/docker.sock -v /path/to/dozzle/data:/data -p 8080:8080 amir20/dozzle --auth-provider simple --auth-ttl 48h
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
      DOZZLE_AUTH_TTL: 48h
```

:::

注意这里只支持时长格式，只能使用 `s`、`m`、`h`，分别表示秒、分钟和小时。

### 为用户设置过滤器

Dozzle 支持为用户设置过滤器。过滤器用于限制用户可见的容器，在 `users.yml` 文件中配置。示例如下：

```yaml
users:
  admin:
    email:
    name: Admin
    password: $2a$11$9ho4vY2LdJ/WBopFcsAS0uORC0x2vuFHQgT/yBqZyzclhHsoaIkzK
    filter:

  guest:
    email:
    name: Guest
    password: $2a$11$9ho4vY2LdJ/WBopFcsAS0uORC0x2vuFHQgT/yBqZyzclhHsoaIkzK
    filter: "label=com.example.app"
```

在这个例子中，`admin` 用户没有过滤器，因此能看到所有容器。`guest` 用户只能看到带有 `com.example.app` 标签的容器。这样就能把访问限制在特定容器上。

> [!NOTE]
> 过滤器也可以用 `--filter` 参数[全局设置](/zh/guide/filters)。该参数对所有用户生效。如果某个用户设置了自己的过滤器，则会覆盖全局过滤器。

### 为用户设置角色

Dozzle 支持给用户分配角色。角色决定用户可以对容器执行哪些操作，在 users.yml 文件中配置。

```yaml
users:
  admin:
    email:
    name: Admin
    password: $2a$11$9ho4vY2LdJ/WBopFcsAS0uORC0x2vuFHQgT/yBqZyzclhHsoaIkzK
    roles:

  guest:
    email:
    name: Guest
    password: $2a$11$9ho4vY2LdJ/WBopFcsAS0uORC0x2vuFHQgT/yBqZyzclhHsoaIkzK
    roles: shell
```

在这个例子中，`admin` 用户没有指定角色，因此拥有所有容器操作的完整权限。`guest` 用户拥有 shell 角色，只能在容器中打开终端。用角色可以很方便地控制和限制用户在 Dozzle 里能做什么。

Dozzle 支持以下角色：

| 角色            | 同时接受               | 授予的权限                                                         |
| --------------- | ---------------------- | ------------------------------------------------------------------ |
| `shell`         | `dozzle_shell`         | 连接容器并打开 exec 会话。实例还需要开启 `--enable-shell`。        |
| `actions`       | `dozzle_actions`       | 启动、停止和重启容器。实例还需要开启 `--enable-actions`。          |
| `download`      | `dozzle_download`      | 把容器日志下载为文件。                                             |
| `notifications` | `dozzle_notifications` | 创建和编辑通知规则与通知目标。                                     |
| `cloud`         | `dozzle_cloud`         | 关联、解除关联并配置 Dozzle Cloud。                                |
| `all`           | `dozzle_all`           | 以上所有角色。`roles` 为空时的默认值。                             |
| `none`          | `dozzle_none`          | 没有任何角色。日志仍可查看，受用户过滤器约束。会覆盖其他所有设置。 |

多个角色之间用逗号或竖线分隔（`shell,actions` 或 `shell|actions`），也可以写成 JSON 数组（`["shell", "actions"]`）。名称不区分大小写。带 `dozzle_` 前缀的别名是为了让身份提供方的用户组名称在前置代理模式下可以原样传入。

> [!WARNING]
> 通知规则是实例级别的。规则按表达式匹配容器，不受用户过滤器约束，因此拥有 `notifications` 角色的用户可以为其过滤器本应隐藏的容器创建规则，并把这些日志行发送到自己控制的目标。只把这个角色授予你信任其访问实例上所有容器的用户。

> [!WARNING]
> Dozzle Cloud 同样是实例级别的。关联操作会保存一个 API key，把警报派发、日志推送和工具执行统统指向某一个云账号，而且云端工具使用的是实例过滤器，而不是执行关联操作那位用户的过滤器。拥有 `cloud` 角色的用户可以把实例关联到自己的云账号，从而看到所有容器，也可以解除已有的关联。只把这个角色授予你信任其访问实例上所有容器的用户。

任何角色都可以加上 `^` 前缀表示排除。排除规则最后生效，所以顺序无关紧要：

```yaml
roles: all,^shell # 除 shell 外的所有角色
```

`none` 是唯一不能被取反的角色。`^none` 会被忽略，而列表中任何位置出现 `none` 都会让其他所有角色失效。

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

## <Icon icon="mdi:swap-horizontal" inline /> 前置代理

把 `--auth-provider` 设为 `forward-proxy`，Dozzle 就会读取代理传来的请求头。

::: code-group

```sh [cli]
$ docker run -v /var/run/docker.sock:/var/run/docker.sock -v /path/to/dozzle/data:/data -p 8080:8080 amir20/dozzle --auth-provider forward-proxy
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
      DOZZLE_AUTH_PROVIDER: forward-proxy
```

:::

这里同样要挂载 `/data`。前置代理模式下用户的个人设置也会写入磁盘，没有这个卷的话，容器每次重建设置就会丢失。

在这种模式下，Dozzle 期望收到以下请求头：

- `Remote-User` 对应用户名，例如 `johndoe`
- `Remote-Email` 对应用户的邮箱地址。这个邮箱也用于查找该用户对应的 [Gravatar](https://gravatar.com/) 头像。
- `Remote-Name` 是显示名称，例如 `John Doe`
- `Remote-Filter` 是允许该用户使用的过滤器列表，以逗号分隔。
- `Remote-Roles` 是允许该用户拥有的角色列表，以逗号分隔。

另外，你还可以配置一个登出 URL：

```yaml
DOZZLE_AUTH_LOGOUT_URL: http://oauth2.example.ru/oauth2/sign_out
```

### 配合 Authelia 使用 Dozzle

[Authelia](https://www.authelia.com/) 是一个开源的身份验证与授权服务器和门户，提供身份与访问管理能力。搭建 Authelia 本身超出了本节的范围，但下面的配置可以作为 Dozzle 配合 Authelia 的示例。

<details>
<summary>➡️ 点击展开 Authelia 示例</summary>

::: code-group

```yaml [docker-compose.yml]
networks:
  net:
    driver: bridge

services:
  authelia:
    image: authelia/authelia
    container_name: authelia
    volumes:
      - ./authelia:/config
    networks:
      - net
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.authelia.rule=Host(`authelia.example.com`)"
      - "traefik.http.routers.authelia.entrypoints=https"
      - "traefik.http.routers.authelia.tls=true"
      - "traefik.http.routers.authelia.tls.options=default"
      - "traefik.http.middlewares.authelia.forwardAuth.address=http://authelia:9091/api/authz/forward-auth"
      - "traefik.http.middlewares.authelia.forwardAuth.trustForwardHeader=true"
      - "traefik.http.middlewares.authelia.forwardAuth.authResponseHeaders=Remote-User,Remote-Groups,Remote-Name,Remote-Email"
    expose:
      - 9091
    restart: unless-stopped

  traefik:
    image: traefik:v3.5
    container_name: traefik
    volumes:
      - ./traefik:/etc/traefik
      - /var/run/docker.sock:/var/run/docker.sock
    networks:
      - net
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.api.rule=Host(`traefik.example.com`)"
      - "traefik.http.routers.api.entrypoints=https"
      - "traefik.http.routers.api.service=api@internal"
      - "traefik.http.routers.api.tls=true"
      - "traefik.http.routers.api.tls.options=default"
      - "traefik.http.routers.api.middlewares=authelia@docker"
    ports:
      - "80:80"
      - "443:443"
    command:
      - "--api"
      - "--providers.docker=true"
      - "--providers.docker.exposedByDefault=false"
      - "--providers.file.filename=/etc/traefik/certificates.yml"
      - "--entrypoints.http=true"
      - "--entrypoints.http.address=:80"
      - "--entrypoints.http.http.redirections.entrypoint.to=https"
      - "--entrypoints.http.http.redirections.entrypoint.scheme=https"
      - "--entrypoints.https=true"
      - "--entrypoints.https.address=:443"
      - "--log=true"
      - "--log.level=DEBUG"

  dozzle:
    image: amir20/dozzle:latest
    networks:
      - net
    environment:
      DOZZLE_AUTH_PROVIDER: forward-proxy
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - dozzle:/data
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.dozzle.rule=Host(`dozzle.example.com`)"
      - "traefik.http.routers.dozzle.entrypoints=https"
      - "traefik.http.routers.dozzle.tls=true"
      - "traefik.http.routers.dozzle.tls.options=default"
      - "traefik.http.routers.dozzle.middlewares=authelia@docker"
    expose:
      - 8080
    restart: unless-stopped

volumes:
  dozzle:
```

```yaml [configuration.yml]
###############################################################
#                   Authelia configuration                      #
###############################################################

server:
  address: tcp://0.0.0.0:9091

log:
  level: info

totp:
  issuer: authelia.com

identity_validation:
  reset_password:
    jwt_secret: a_very_important_secret

authentication_backend:
  file:
    path: /config/users_database.yml

access_control:
  default_policy: deny
  rules:
    - domain: traefik.example.com
      policy: one_factor
    - domain: dozzle.example.com
      policy: one_factor

session:
  secret: unsecure_session_secret
  cookies:
    - domain: example.com # 应与你受保护的根域名一致
      authelia_url: https://authelia.example.com
      default_redirection_url: https://public.example.com

regulation:
  max_retries: 3
  find_time: 120
  ban_time: 300

storage:
  encryption_key: you_must_generate_a_random_string_of_more_than_twenty_chars_and_configure_this
  local:
    path: /config/db.sqlite3

notifier:
  filesystem:
    filename: /config/notification.txt
```

:::

必须使用有效的 SSL 密钥，因为 Authelia 只支持 SSL。

Authelia 在 `Remote-Groups` 中发送用户的组信息，而 Dozzle 默认不读取这个头。要把 Authelia 的用户组映射到 Dozzle 的[角色](#setting-specific-roles-for-users)，请在 Dozzle 服务上设置 `DOZZLE_AUTH_HEADER_ROLES: Remote-Groups`，并按角色名称来命名用户组。带 `dozzle_` 前缀的别名就是为此准备的：名为 `dozzle_shell` 的组会授予 `shell` 角色，其他组名会被忽略。不做这个映射的话，每个通过验证的用户都会拿到全部角色。

</details>

### 配合 Cloudflare Zero Trust 使用 Dozzle

Cloudflare Zero Trust 是一项为自托管软件提供认证访问的服务。本节说明如何配置 Dozzle 使用 Cloudflare Zero Trust 进行身份验证。

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      DOZZLE_AUTH_PROVIDER: forward-proxy
      DOZZLE_AUTH_HEADER_USER: Cf-Access-Authenticated-User-Email
      DOZZLE_AUTH_HEADER_EMAIL: Cf-Access-Authenticated-User-Email
      DOZZLE_AUTH_HEADER_NAME: Cf-Access-Authenticated-User-Email
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - dozzle:/data
    expose:
      - 8080
    restart: unless-stopped

volumes:
  dozzle:
```

`expose` 让 8080 端口不暴露在主机上，唯一的入口就是隧道。如果用 `ports` 发布出去，主机上的任何人都能自己设置 `Cf-Access-Authenticated-User-Email`，完全绕过 Cloudflare。

启动 Dozzle 容器后，按照这份[指南](https://developers.cloudflare.com/cloudflare-one/applications/configure-apps/self-hosted-apps/)在 Cloudflare Zero Trust 控制台中配置应用。

### 配合 Pocket ID 使用 Dozzle

你需要先起一个容器，通过反向代理传递 OpenID Connect 验证信息。

下面是使用 [oauth2-proxy](https://github.com/oauth2-proxy/oauth2-proxy) 的示例。

<details>
<summary>➡️ 点击展开 oauth2-proxy 示例</summary>

1. 在 Pocket ID 中为 Dozzle 新建一个 OIDC 客户端：
   - **名称：** `Dozzle`
   - **回调 URL：** `https://dozzle.example.com/oauth2/callback`
   - **PKCE：** `Enabled`

   复制 **Client ID** 和 **Client Secret** 的值备用。

2. 在现有的 Dozzle compose 中加入以下内容：

   ```yml
   environment:
     DOZZLE_AUTH_PROVIDER: forward-proxy
     DOZZLE_AUTH_HEADER_USER: X-Forwarded-User
     DOZZLE_AUTH_HEADER_EMAIL: X-Forwarded-Email
     DOZZLE_AUTH_HEADER_NAME: X-Forwarded-Preferred-Username
   ```

   注释掉 Dozzle 的端口，因为流量会改为经过新的验证容器转发。

   这种做法一般不需要改动反向代理的配置。

   ```yml
   # ports:
   #   - 8080:8080
   ```

3. 在现有的 Dozzle compose 中新增一个 oauth2-proxy 服务：

   ```yml
   services:
     # ...
     oauth2-proxy:
       image: quay.io/oauth2-proxy/oauth2-proxy:latest
       restart: unless-stopped
       container_name: dozzle-oidc
       command: --config /oauth2-proxy.cfg
       volumes:
         - "./oauth2-proxy.cfg:/oauth2-proxy.cfg"
       ports:
         - 8080:4180
   ```

4. 创建 oauth2-proxy 的配置文件。

   在 compose 文件所在目录下创建 `oauth2-proxy.cfg`：

   ```toml
    client_id = "xxx"                            # 来自 Pocket ID
    client_secret = "xxx"                        # 来自 Pocket ID
    cookie_secret = "xxx"                        # 用 openssl rand -base64 32 | tr -- '+/' '-_' 生成
    upstreams = "http://dozzle:8080"             # 上游指向 Dozzle 容器的内部端口
    code_challenge_method = "S256"               # PKCE 挑战方式，plain 或 S256
    cookie_expire = "0"                          # 秒，0 表示会话级
    cookie_name = "__Host-oauth2-proxy"          # 或 __Secure-oauth2-proxy（安全性稍低）
    cookie_secure = true                         # 使用 secure 的 HTTPS cookie
    email_domains = ["*"]                        # 允许任意邮箱域名登录
    http_address = "0.0.0.0:4180"                # oauth2-proxy 监听的端口
    oidc_issuer_url = "https://id.example.com"   # 你的 Pocket 基础 URL
    provider_display_name = "Pocket ID"          # OIDC 登录时显示的名称
    provider = "oidc"                            # 使用 OpenID Connect
    reverse_proxy = true                         # 反向代理流量
    scope = "openid email profile groups"        # 透传这些 OIDC scope
   ```

   按注释填好各项变量。

5. 最后，重启你的 Docker compose 服务栈。

   现在反向代理应该会通过 oauth2-proxy 让你登录 Dozzle。

   如有问题，请查看日志排查。

</details>
