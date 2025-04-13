---
title: Authentication
---

# Setting Up Authentication <Badge type="tip" text="Updated" />

Dozzle supports two configurations for authentication. In the first configuration, you bring your own authentication method by protecting Dozzle through a proxy. Dozzle can read appropriate headers out of the box.

If you do not have an authentication solution, then Dozzle has a simple file-based user management solution. Authentication providers are set up using the `--auth-provider` flag. In both configurations, Dozzle will try to save user settings to disk. This data is written to `/data`.

## File-Based User Management

Dozzle supports multi-user authentication by setting `--auth-provider` to `simple`. In this mode, Dozzle will attempt to read the users file from `/data/`, prioritizing `users.yml` over `users.yaml` if both files are present. If only one of the files exists, it will be used. The log will indicate which file is being read (e.g., `Reading users.yml file`).

### Example file paths:

- `/data/users.yml`
- `/data/users.yaml`

The content of the file looks like:

```yaml
users:
  # "admin" here is username
  admin:
    email: me@email.net
    name: Admin
    # Generate with docker run run -it --rm amir20/dozzle generate --name Admin --email me@email.net --password secret admin
    password: $2a$11$9ho4vY2LdJ/WBopFcsAS0uORC0x2vuFHQgT/yBqZyzclhHsoaIkzK
    filter:
```

Dozzle uses `email` to generate avatars using [Gravatar](https://gravatar.com/). It is optional. The password is hashed using `bcrypt` which can be generated using `docker run amir20/dozzle generate`.

> [!WARNING]
> In previous versions of Dozzle, SHA-256 was used to hash passwords. Bcrypt is now more secure and is recommended for future use. Dozzle will revert to SHA-256 if it does not find a bcrypt hash. It is advisable to update the password hash to bcrypt using `generate`. For more details, see [this issue](https://github.com/amir20/dozzle/security/advisories/GHSA-w7qr-q9fh-fj35).

You will need to mount this file for Dozzle to find it. Here is an example:

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

Or using Docker secrets:

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

### Extending Authentication Cookie Lifetime

By default, Dozzle uses session cookies which expire when the browser is closed. You can extend the lifetime of the cookie by setting `--auth-ttl` to a duration. Here is an example:

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

Note that only duration is supported. You can only use `s`, `m`, `h` for seconds, minutes and hours respectively.

### Setting specific filters for users

Dozzle supports setting filters for users. Filters are used to restrict the containers that a user can see. Filters are set in the `users.yml` file. Here is an example:

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

In this example, the `admin` user has no filter, so they can see all containers. The `guest` user can only see containers with the label `com.example.app`. This is useful for restricting access to specific containers.

> [!NOTE]
> Filters can also be set [globally](/guide/filters) with the `--filter` flag. This flag is applied to all users. If a user has a filter set, it will override the global filter.

## Generating users.yml

Dozzle has a built-in `generate` command to generate `users.yml`. Here is an example:

```sh
docker run -it --rm amir20/dozzle generate admin --password password --email test@email.net --name "John Doe" --user-filter name=foo > users.yml
```

In this example, `admin` is the username. Email and name are optional but recommended to display accurate avatars. `docker run -it --rm amir20/dozzle generate --help` displays all options. The `--user-filter` flag is a comma-separated list of filters.

## Forward Proxy

Dozzle can be configured to read proxy headers by setting `--auth-provider` to `forward-proxy`.

::: code-group

```sh [cli]
$ docker run -v /var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --auth-provider forward-proxy
```

```yaml [docker-compose.yml]
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

- `Remote-User` to map to the username e.g. `johndoe`
- `Remote-Email` to map to the user's email address. This email is also used to find the right [Gravatar](https://gravatar.com/) for the user.
- `Remote-Name` to be a display name like `John Doe`
- `Remote-Filter` to be a comma-separated list of filters allowed for user.

### Setting up Dozzle with Authelia

[Authelia](https://www.authelia.com/) is an open-source authentication and authorization server and portal fulfilling the identity and access management. While setting up Authelia is out of scope for this section, the configuration can be shared as an example for setting up Dozzle with Authelia.

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
#                   Authelia configuration                      #
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

### Setting up Dozzle with Cloudflare Zero Trust

Cloudflare Zero Trust is a service for authenticated access to self-hosted software. This section defines how Dozzle can be set up to use Cloudflare Zero Trust for authentication.

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    networks:
      - net
    environment:
      DOZZLE_AUTH_PROVIDER: forward-proxy
      DOZZLE_AUTH_HEADER_USER: Cf-Access-Authenticated-User-Email
      DOZZLE_AUTH_HEADER_EMAIL: Cf-Access-Authenticated-User-Email
      DOZZLE_AUTH_HEADER_NAME: Cf-Access-Authenticated-User-Email
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    expose:
      - 8080
    restart: unless-stopped
```

After running the Dozzle container, configure the Application in Cloudflare Zero Trust dashboard by following the [guide](https://developers.cloudflare.com/cloudflare-one/applications/configure-apps/self-hosted-apps/).
