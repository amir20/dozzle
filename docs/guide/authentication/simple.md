---
title: Simple Authentication
---

# <Icon icon="mdi:account-cog-outline" inline /> Simple Authentication

Dozzle's own user management. Users live in a `users.yml` file that Dozzle owns, and Dozzle serves its own login page. Set `--auth-provider` to `simple` to turn it on.

Passwords are one way to prove you are one of those users. [Sign in with GitHub or OIDC](/guide/authentication/oauth) is the other, and both read the same `users.yml`.

> [!TIP]
> Use the built-in [`generate` command](/guide/authentication#generating-users-yml) to create `users.yml` rather than writing bcrypt hashes by hand.

Dozzle supports multi-user authentication by setting `--auth-provider` to `simple`. In this mode, Dozzle will attempt to read the users file from `/data/`, prioritizing `users.yml` over `users.yaml` if both files are present. If only one of the files exists, it will be used. The log will indicate which file is being read (e.g., `Reading users.yml file`).

## Example file paths:

- `/data/users.yml`
- `/data/users.yaml`

The content of the file looks like:

```yaml
users:
  # "admin" here is username
  admin:
    email: me@email.net
    name: Admin
    # Generate with docker run -it --rm amir20/dozzle generate admin --password password --email me@email.net --name "Admin"
    password: $2a$11$9ho4vY2LdJ/WBopFcsAS0uORC0x2vuFHQgT/yBqZyzclhHsoaIkzK
    filter:
    roles:
```

Dozzle uses `email` to generate avatars using [Gravatar](https://gravatar.com/). It is optional. The password is hashed using `bcrypt` which can be generated using `docker run amir20/dozzle generate`.

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

## Extending Authentication Cookie Lifetime

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

## Setting specific filters for users

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

## Setting specific roles for users

Dozzle allows assigning roles to users. Roles define what actions a user can perform on containers. Roles are configured in the users.yml file.

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

In this example, the `admin` user has no roles specified, so they have full access to all container actions. The `guest` user has the shell role, meaning they can only open a shell in the containers. Roles make it easy to control and restrict what users can do in Dozzle.

Dozzle supports the following roles:

| Role            | Also accepted          | Grants                                                                                    |
| --------------- | ---------------------- | ----------------------------------------------------------------------------------------- |
| `shell`         | `dozzle_shell`         | Attach to a container and open an exec session. The instance also needs `--enable-shell`. |
| `actions`       | `dozzle_actions`       | Start, stop and restart containers. The instance also needs `--enable-actions`.           |
| `download`      | `dozzle_download`      | Download container logs as a file.                                                        |
| `notifications` | `dozzle_notifications` | Create and edit notification rules and destinations.                                      |
| `cloud`         | `dozzle_cloud`         | Link, unlink and configure Dozzle Cloud.                                                  |
| `all`           | `dozzle_all`           | Every role above. This is the default when `roles` is empty.                              |
| `none`          | `dozzle_none`          | No roles. Logs are still viewable, subject to the user's filter. Overrides anything else. |

Roles are separated by commas or pipes (`shell,actions` or `shell|actions`), and a JSON array works too (`["shell", "actions"]`). Names are case insensitive. The `dozzle_` prefixed aliases exist so group names from an identity provider can be passed through unchanged in forward proxy mode.

> [!WARNING]
> Notification rules are instance wide. A rule matches containers by expression, not by the user's filter, so a user with the `notifications` role can create a rule for containers their filter otherwise hides and receive those log lines at a destination they control. Only grant it to users you trust with every container on the instance.

> [!WARNING]
> Dozzle Cloud is also instance wide. Linking stores a single API key that repoints alert dispatch, log streaming and tool execution at one cloud account, and cloud tools run with the instance filter rather than the linking user's filter. A user with the `cloud` role can link the instance to their own cloud account and see every container through it, or unlink an existing connection. Only grant it to users you trust with every container on the instance.

Any role can be prefixed with `^` to exclude it. Exclusions are applied last, so order doesn't matter:

```yaml
roles: all,^shell # everything except shell
```

`none` is the one role that cannot be negated. `^none` is ignored, and a plain `none` anywhere in the list drops every other role.
