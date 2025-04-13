---
title: Environment Variables and Subcommands
---

# Global Environment Variables

Configurations can be done with flags or environment variables. The table below outlines all supported options and their respective env vars.

| Flag                  | Env Variable               | Default        |
| --------------------- | -------------------------- | -------------- |
| `--addr`              | `DOZZLE_ADDR`              | `:8080`        |
| `--base`              | `DOZZLE_BASE`              | `/`            |
| `--hostname`          | `DOZZLE_HOSTNAME`          | `""`           |
| `--level`             | `DOZZLE_LEVEL`             | `info`         |
| `--auth-provider`     | `DOZZLE_AUTH_PROVIDER`     | `none`         |
| `--auth-header-user`  | `DOZZLE_AUTH_HEADER_USER`  | `Remote-User`  |
| `--auth-header-email` | `DOZZLE_AUTH_HEADER_EMAIL` | `Remote-Email` |
| `--auth-header-name`  | `DOZZLE_AUTH_HEADER_NAME`  | `Remote-Name`  |
| `--enable-actions`    | `DOZZLE_ENABLE_ACTIONS`    | `false`        |
| `--enable-shell`      | `DOZZLE_ENABLE_SHELL`      | `false`        |
| `--filter`            | `DOZZLE_FILTER`            | `""`           |
| `--no-analytics`      | `DOZZLE_NO_ANALYTICS`      | `false`        |
| `--mode`              | `DOZZLE_MODE`              | `server`       |
| `--remote-host`       | `DOZZLE_REMOTE_HOST`       |                |
| `--remote-agent`      | `DOZZLE_REMOTE_AGENT`      |                |
| `--timeout`           | `DOZZLE_TIMEOUT`           | `10s`          |
| `--namespace`         | `DOZZLE_NAMESPACE`         | `""`           |

> [!TIP]
> Some flags like `--remote-host` or `--remote-agent` can be used multiple times. For example, `--remote-agent tcp://167.99.1.1:7007 --remote-agent tcp://167.99.1.2:7007` or comma-separated `DOZZLE_REMOTE_AGENT=tcp://167.99.1.1:7007,tcp://167.99.1.2:7007`.

## Generate users.yml

Dozzle supports generating `users.yml` file. This file is used to authenticate users. Here is an example:

```sh
docker run -it --rm amir20/dozzle generate admin --password password --email test@email.net --name "John Doe" > users.yml
```

In this example, `admin` is the username. Email and name are optional but recommended to display accurate avatars. `docker run amir20/dozzle generate --help` displays all options.

| Flag         | Description      | Default |
| ------------ | ---------------- | ------- |
| `--password` | User's password  |         |
| `--email`    | User's email     |         |
| `--name`     | User's full name |         |

See [authentication](/guide/authentication) for more information.

## Agent Mode

Dozzle supports running in agent mode. Agent mode is useful when running Dozzle on a remote host and you want to monitor a different Docker host. Agent mode is enabled by setting the `--remote-agent` flag. Here is an example:

```sh
docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --remote-agent remote-ip:7007
```

| Flag     | Env Variable        | Default |
| -------- | ------------------- | ------- |
| `--addr` | `DOZZLE_AGENT_ADDR` | `:7007` |

See [agent](/guide/agent) for more information.

## Healthcheck

Dozzle supports healthcheck using `dozzle healthcheck` command. It is not enabled by default as it adds extra CPU usage. To use `healthcheck`, you need to configure it.

See [healthcheck](/guide/healthcheck) for more information.
