---
title: Forward Proxy
---

# <Icon icon="mdi:swap-horizontal" inline /> Forward Proxy

Dozzle can be configured to read proxy headers by setting `--auth-provider` to `forward-proxy`.

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

Mount `/data` here as well. Per-user settings are written to disk in forward-proxy mode too, and without the volume they are lost every time the container is recreated.

In this mode, Dozzle expects the following headers:

- `Remote-User` to map to the username e.g. `johndoe`
- `Remote-Email` to map to the user's email address. This email is also used to find the right [Gravatar](https://gravatar.com/) for the user.
- `Remote-Name` to be a display name like `John Doe`
- `Remote-Filter` to be a comma-separated list of filters allowed for user.
- `Remote-Roles` to be a comma-separated list of roles allowed for user.

Additionally, you can configure a logout URL with:

```yaml
DOZZLE_AUTH_LOGOUT_URL: http://oauth2.example.ru/oauth2/sign_out
```

## Setting up Dozzle with Authelia

[Authelia](https://www.authelia.com/) is an open-source authentication and authorization server and portal fulfilling the identity and access management. While setting up Authelia is out of scope for this section, the configuration can be shared as an example for setting up Dozzle with Authelia.

<details>
<summary>➡️ Click to expand Authelia example</summary>

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
    - domain: example.com # Should match whatever your root protected domain is
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

Valid SSL keys are required because Authelia only supports SSL.

Authelia sends group membership in `Remote-Groups`, and Dozzle does not read that header by default. To map Authelia groups onto Dozzle [roles](/guide/authentication/simple#setting-specific-roles-for-users), set `DOZZLE_AUTH_HEADER_ROLES: Remote-Groups` on the Dozzle service and name the groups after the roles. The `dozzle_` prefixed aliases exist for this, so a group called `dozzle_shell` grants the `shell` role and other group names are ignored. Without that mapping every authenticated user gets all roles.

</details>

## Setting up Dozzle with Cloudflare Zero Trust

Cloudflare Zero Trust is a service for authenticated access to self-hosted software. This section defines how Dozzle can be set up to use Cloudflare Zero Trust for authentication.

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

`expose` keeps port 8080 off the host, so the only way in is through the tunnel. Publishing it with `ports` would let anyone on the host set `Cf-Access-Authenticated-User-Email` themselves and skip Cloudflare entirely.

After running the Dozzle container, configure the Application in Cloudflare Zero Trust dashboard by following the [guide](https://developers.cloudflare.com/cloudflare-one/applications/configure-apps/self-hosted-apps/).

## Setting up Dozzle with Pocket ID

> [!TIP]
> Dozzle speaks OpenID Connect natively now, so you can point `--auth-oidc-issuer` straight at Pocket ID and skip oauth2-proxy entirely. See [Sign in with GitHub & OIDC](/guide/authentication/oauth#sign-in-with-oidc). The recipe below is still here for setups that already run oauth2-proxy or want the proxy to guard more than Dozzle.

You must first setup a container to pass OpenID Connect authentication through your reverse proxy.

Below is an example using [oauth2-proxy](https://github.com/oauth2-proxy/oauth2-proxy).

<details>
<summary>➡️ Click to expand oauth2-proxy example</summary>

1. Create a new OIDC client in Pocket ID for Dozzle:
   - **Name:** `Dozzle`
   - **Callback URLs:** `https://dozzle.example.com/oauth2/callback`
   - **PKCE:** `Enabled`

   Copy the **Client ID** and **Client Secret** values for use later.

2. Add the following to your existing Dozzle compose:

   ```yml
   environment:
     DOZZLE_AUTH_PROVIDER: forward-proxy
     DOZZLE_AUTH_HEADER_USER: X-Forwarded-User
     DOZZLE_AUTH_HEADER_EMAIL: X-Forwarded-Email
     DOZZLE_AUTH_HEADER_NAME: X-Forwarded-Preferred-Username
   ```

   Comment out the Dozzle ports, as we will redirect these through the new authentication container.

   This method should not require any changes to your reverse proxy configuration.

   ```yml
   # ports:
   #   - 8080:8080
   ```

3. Add a new oauth2-proxy container service to your existing Dozzle compose:

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

4. Create the oauth2-proxy config file.

   In the directory beside your compose file, create `oauth2-proxy.cfg` :

   ```toml
    client_id = "xxx"                            # from Pocket ID
    client_secret = "xxx"                        # from Pocket ID
    cookie_secret = "xxx"                        # generate with openssl rand -base64 32 | tr -- '+/' '-_'
    upstreams = "http://dozzle:8080"             # upstream to Dozzle containers internal port
    code_challenge_method = "S256"               # PKCE challenges plain or S256
    cookie_expire = "0"                          # seconds, 0 for session
    cookie_name = "__Host-oauth2-proxy"          # or __Secure-oauth2-proxy (less secure)
    cookie_secure = true                         # uses the secure HTTPS cookie
    email_domains = ["*"]                        # allows any email domain to authenticate
    http_address = "0.0.0.0:4180"                # port oauth2-proxy listens on
    oidc_issuer_url = "https://id.example.com"   # your Pocket base URL
    provider_display_name = "Pocket ID"          # display name for OIDC login
    provider = "oidc"                            # use OpenID connect
    reverse_proxy = true                         # reverse proxy the traffic
    scope = "openid email profile groups"        # passthru these OIDC scopes
   ```

   Fill in the variables per the comments.

5. Finally - restart your Docker compose stack.

   Your reverse proxy should now authenticate you to Dozzle via oauth2-proxy.

   Check logs for troubleshooting.

</details>
