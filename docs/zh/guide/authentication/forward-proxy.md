---
title: 前置代理
sourceHash: 37a5c3119fb2
---

# <Icon icon="mdi:swap-horizontal" inline /> 前置代理

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

## 配合 Authelia 使用 Dozzle

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

Authelia 在 `Remote-Groups` 中发送用户的组信息，而 Dozzle 默认不读取这个头。要把 Authelia 的用户组映射到 Dozzle 的[角色](/zh/guide/authentication/simple#为用户设置角色)，请在 Dozzle 服务上设置 `DOZZLE_AUTH_HEADER_ROLES: Remote-Groups`，并按角色名称来命名用户组。带 `dozzle_` 前缀的别名就是为此准备的：名为 `dozzle_shell` 的组会授予 `shell` 角色，其他组名会被忽略。不做这个映射的话，每个通过验证的用户都会拿到全部角色。

</details>

## 配合 Cloudflare Zero Trust 使用 Dozzle

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

## 配合 Pocket ID 使用 Dozzle

> [!TIP]
> Dozzle 现在原生支持 OpenID Connect，你可以直接把 `--auth-oidc-issuer` 指向 Pocket ID，完全不需要 oauth2-proxy。参见[使用 GitHub 与 OIDC 登录](/zh/guide/authentication/oauth#使用-oidc-登录)。下面这套方案仍然保留，供已经在跑 oauth2-proxy、或者希望代理保护的不只是 Dozzle 的场景参考。

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
