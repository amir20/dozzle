---
title: Authentication
---

# Setting Up Authentication

Dozzle support two models of authentication. If you have already setup authentication behind a proxy then Dozzle can work out of the box by reading forwarded headers. If you do not currently have an authentication solution then Dozzle has a simple file based user management solution. Authentication providers are setup using `--auth-provider` flag.

## Forward Proxy

Dozzle can be configured to read proxy headers by setting `--auth-provider` to `forward-proxy`.

::: code-group

```sh [cli]
$ docker run -v /var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --auth-provider forward-proxy
```

```yaml [docker-compose.yml]
version: "3"
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    ports:
      - 8080:8080
    environment:
      DOZZLE_AUTH_PROVIDER: forward-proxy
```

:::

In this mode, Dozzle expects the following headers:

- `Remote-User` to map to the username eg. `johndoe`
- `Remote-Email` to map to the user's email address. This email is also used to find the right [Gravatar](https://gravatar.com/) for the user.
- `Remote-Name` to be a display name like `John Doe`

### Setting up Dozzle with Authelia

[Authelia](https://www.authelia.com/) is an open-source authentication and authorization server and portal fulfilling the identity and access management. While setting up Authelia is out of scope for this section, the configuration can be shared as an example for setting up Dozzle with Authelia.

::: code-group

```yaml [docker-compose.yml]
version: "3.3"

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
      - "traefik.http.middlewares.authelia.forwardauth.address=http://authelia:9091/api/verify?rd=https://authelia.example.com"
      - "traefik.http.middlewares.authelia.forwardauth.trustForwardHeader=true"
      - "traefik.http.middlewares.authelia.forwardauth.authResponseHeaders=Remote-User,Remote-Groups,Remote-Name,Remote-Email"
    expose:
      - 9091
    restart: unless-stopped

  traefik:
    image: traefik:2.10.5
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
```

```yaml [configuration.yml]
###############################################################
#                   Authelia configuration                    #
###############################################################

jwt_secret: a_very_important_secret
default_redirection_url: https://public.example.com

server:
  host: 0.0.0.0
  port: 9091

log:
  level: info

totp:
  issuer: authelia.com

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
  domain: example.com # Should match whatever your root protected domain is

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

## File Based User Management

::: info
More details coming soon.
:::

## Single Username/Password

::: danger
This authentication method will be removed in v6.x in favor of a file based solution currently in development.
:::

Dozzle supports a very simple authentication out of the box with just username and password. You should deploy using SSL to keep the credentials safe. See configuration to use `--username` and `--password`. You can also use docker secrets `--usernamefile` and `--passwordfile`.

::: code-group

```sh [cli]
$ docker run -v /var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --username amirraminfar --password supersecretpassword
```

```yaml [docker-compose.yml]
version: "3"
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    ports:
      - 8080:8080
    environment:
      DOZZLE_USERNAME: amirraminfar
      DOZZLE_PASSWORD: supersecretpassword
```

:::

## Setting up authentication with Docker secrets

Dozzle also support path to file for username and password which can be used to with Docker Secrets.

```yaml
version: "3"
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      DOZZLE_LEVEL: debug
      DOZZLE_USERNAME_FILE: /run/secrets/dozzle_user
      DOZZLE_PASSWORD_FILE: /run/secrets/dozzle_password
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    secrets:
      - dozzle_user
      - dozzle_password
    ports:
      - 8080:8080

secrets:
  dozzle_user:
    file: dozzle_user.txt
  dozzle_password:
    file: dozzle_password.txt
```
