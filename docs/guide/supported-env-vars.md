---
title: Environment Variables and Subcommands
---

# Environment Variables

Every option can be set with a flag or with an environment variable. Flags and environment variables always win over settings saved by the [setup wizard](/guide/setup-wizard) in `dozzle.yml`.

Options that take a list (`DOZZLE_FILTER`, `DOZZLE_REMOTE_HOST`, `DOZZLE_REMOTE_AGENT`, `DOZZLE_NAMESPACE`) accept a comma-separated value in the environment variable, or the flag repeated once per item:

```sh
--remote-agent 167.99.1.1:7007 --remote-agent 167.99.1.2:7007
DOZZLE_REMOTE_AGENT=167.99.1.1:7007,167.99.1.2:7007
```

## Server

| Variable                                  | Description                                                                                    | Values                                                      | Default  |
| ----------------------------------------- | ---------------------------------------------------------------------------------------------- | ----------------------------------------------------------- | -------- |
| `DOZZLE_ADDR`<br>`--addr`                 | Address and port the web server listens on. Rarely needed inside a container.                  | `host:port`, e.g. `:9090`                                   | `:8080`  |
| `DOZZLE_BASE`<br>`--base`                 | Path prefix to serve Dozzle under. See [changing base](/guide/changing-base).                  | a path, e.g. `/logs`                                        | `/`      |
| `DOZZLE_HOSTNAME`<br>`--hostname`         | Name shown in the UI for this instance. See [hostname](/guide/hostname).                       | any string                                                  | none     |
| `DOZZLE_HOST_ID`<br>`--host-id`           | Overrides the id Dozzle derives for this host. Only needed when it collides with another host. | letters, digits, `_`, `.`, `-`                              | derived  |
| `DOZZLE_LEVEL`<br>`--level`               | Dozzle's own log level. See [debugging](/guide/debugging).                                     | `trace`, `debug`, `info`, `warn`, `error`                   | `info`   |
| `DOZZLE_MODE`<br>`--mode`                 | Deployment mode.                                                                               | `server`, [`swarm`](/guide/swarm-mode), [`k8s`](/guide/k8s) | `server` |
| `DOZZLE_TIMEOUT`<br>`--timeout`           | Timeout for calls to the Docker or Kubernetes API.                                             | a duration, e.g. `30s`                                      | `10s`    |
| `DOZZLE_NO_ANALYTICS`<br>`--no-analytics` | Turns off anonymous [analytics](/guide/analytics).                                             | `true`, `false`                                             | `false`  |

## Containers and Hosts

| Variable                                  | Description                                                                                          | Values                            | Default           |
| ----------------------------------------- | ---------------------------------------------------------------------------------------------------- | --------------------------------- | ----------------- |
| `DOZZLE_FILTER`<br>`--filter`             | Only shows containers matching a Docker filter. See [filters](/guide/filters).                       | `key=value`, e.g. `label=app=web` | none              |
| `DOZZLE_REMOTE_AGENT`<br>`--remote-agent` | [Agents](/guide/agent) to connect to, with an optional display name and group.                       | `host:port[\|name[\|group]]`      | none              |
| `DOZZLE_REMOTE_HOST`<br>`--remote-host`   | Docker hosts to connect to over TCP. See [remote hosts](/guide/remote-hosts).                        | `tcp://host:port[\|label]`        | none              |
| `DOZZLE_NAMESPACE`<br>`--namespace`       | Kubernetes namespaces to watch. Only used with `DOZZLE_MODE=k8s`.                                    | namespace names                   | all               |
| `DOZZLE_CERT`<br>`--cert`                 | TLS certificate used to talk to agents. See [custom certificates](/guide/agent#custom-certificates). | a file path                       | `dozzle_cert.pem` |
| `DOZZLE_KEY`<br>`--key`                   | TLS private key used to talk to agents.                                                              | a file path                       | `dozzle_key.pem`  |

## Features

| Variable                                              | Description                                                                                                         | Values                       | Default                             |
| ----------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------- | ---------------------------- | ----------------------------------- |
| `DOZZLE_ENABLE_ACTIONS`<br>`--enable-actions`         | Allows starting, stopping, restarting, removing and updating containers from the UI. See [actions](/guide/actions). | `true`, `false`              | `false`                             |
| `DOZZLE_ENABLE_SHELL`<br>`--enable-shell`             | Allows attaching to and running a shell in containers from the UI. See [shell](/guide/shell).                       | `true`, `false`              | `false`                             |
| `DOZZLE_ENABLE_MCP`<br>`--enable-mcp`                 | Exposes the [MCP](/guide/mcp) endpoint for LLM clients.                                                             | `true`, `false`              | `false`                             |
| `DOZZLE_DISABLE_AVATARS`<br>`--disable-avatars`       | Hides user avatars when authentication is on.                                                                       | `true`, `false`              | `false`                             |
| `DOZZLE_RELEASE_CHECK_MODE`<br>`--release-check-mode` | Whether Dozzle checks for its own new releases. `manual` only checks when you ask.                                  | `automatic`, `manual`        | `automatic`                         |
| `DOZZLE_IMAGE_CHECK_MODE`<br>`--image-check-mode`     | Whether Dozzle checks registries for newer container images. See [update checking](/guide/actions#update-checking). | `automatic`, `manual`, `off` | same as `DOZZLE_RELEASE_CHECK_MODE` |
| `DOZZLE_AUTO_UPDATE`<br>`--auto-update`               | Updates Dozzle's own container on a schedule. `weekly` runs on Sunday. Requires `DOZZLE_ENABLE_ACTIONS`.            | `off`, `daily`, `weekly`     | `off`                               |
| `DOZZLE_AUTO_UPDATE_TIME`<br>`--auto-update-time`     | Time of day the auto update runs, in the server's local time.                                                       | `HH:MM`, e.g. `04:30`        | `03:00`                             |

## Authentication

See [authentication](/guide/authentication) for how the providers work.

| Variable                                        | Description                                                                                                   | Values                                                    | Default   |
| ----------------------------------------------- | ------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------- | --------- |
| `DOZZLE_AUTH_PROVIDER`<br>`--auth-provider`     | Which authentication provider to use. `github` and `google` are aliases for `simple`.                         | `none`, `simple`, `oidc`, `forward-proxy`                 | `none`    |
| `DOZZLE_AUTH_TTL`<br>`--auth-ttl`               | How long a login lasts. `session` logs out when the browser closes.                                           | `session` or a duration, e.g. `48h` (units `s`, `m`, `h`) | `session` |
| `DOZZLE_AUTH_LOGOUT_URL`<br>`--auth-logout-url` | Where to send the user on logout with `forward-proxy`. With `oidc` it overrides the issuer's logout endpoint. | a URL                                                     | none      |

### GitHub OAuth

Adds "Sign in with GitHub" to `simple` auth. See [OAuth](/guide/authentication/oauth).

| Variable                                                            | Description                            | Values   | Default |
| ------------------------------------------------------------------- | -------------------------------------- | -------- | ------- |
| `DOZZLE_AUTH_GITHUB_CLIENT_ID`<br>`--auth-github-client-id`         | Client id of the GitHub OAuth app.     | a string | none    |
| `DOZZLE_AUTH_GITHUB_CLIENT_SECRET`<br>`--auth-github-client-secret` | Client secret of the GitHub OAuth app. | a string | none    |

### OpenID Connect

Used with `DOZZLE_AUTH_PROVIDER=oidc`. See [OIDC](/guide/authentication/oidc).

| Variable                                                        | Description                                                                                                                                                                                   | Values                     | Default |
| --------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------- | ------- |
| `DOZZLE_AUTH_OIDC_ISSUER`<br>`--auth-oidc-issuer`               | Issuer URL of the identity provider.                                                                                                                                                          | a URL                      | none    |
| `DOZZLE_AUTH_OIDC_CLIENT_ID`<br>`--auth-oidc-client-id`         | Client id registered with the provider.                                                                                                                                                       | a string                   | none    |
| `DOZZLE_AUTH_OIDC_CLIENT_SECRET`<br>`--auth-oidc-client-secret` | Client secret registered with the provider.                                                                                                                                                   | a string                   | none    |
| `DOZZLE_AUTH_OIDC_NAME`<br>`--auth-oidc-name`                   | Label on the login button.                                                                                                                                                                    | any string                 | `SSO`   |
| `DOZZLE_AUTH_OIDC_ROLES_CLAIM`<br>`--auth-oidc-roles-claim`     | Claim to read roles from. Only needed when the default search (`dozzle_roles`, `resource_access.<client-id>.roles`, `roles`) misses it.                                                       | a dot-separated claim path | none    |
| `DOZZLE_AUTH_OIDC_FILTERS_CLAIM`<br>`--auth-oidc-filters-claim` | Claim to read container filters from. Only needed when the default search (`dozzle_filters`, `resource_access.<client-id>.filters`, `filters`) misses it.                                     | a dot-separated claim path | none    |
| `DOZZLE_AUTH_OIDC_SCOPES`<br>`--auth-oidc-scopes`               | Extra scopes to request on top of `openid`, `profile` and `email`, for providers that only release a claim [when its scope is asked for](/guide/authentication/oidc#requesting-extra-scopes). | comma-separated scopes     | none    |

> [!TIP]
> `DOZZLE_AUTH_GITHUB_CLIENT_SECRET` and `DOZZLE_AUTH_OIDC_CLIENT_SECRET` also accept a `_FILE` counterpart naming a file to read the value from, for use with [Docker secrets](/guide/authentication/oauth#using-docker-secrets-for-the-client-secret).

### Forward Proxy

Used with `DOZZLE_AUTH_PROVIDER=forward-proxy`. Each variable names the HTTP header the proxy sets. See [forward proxy](/guide/authentication/forward-proxy).

| Variable                                              | Description                                  | Values        | Default         |
| ----------------------------------------------------- | -------------------------------------------- | ------------- | --------------- |
| `DOZZLE_AUTH_HEADER_USER`<br>`--auth-header-user`     | Header carrying the username.                | a header name | `Remote-User`   |
| `DOZZLE_AUTH_HEADER_EMAIL`<br>`--auth-header-email`   | Header carrying the email.                   | a header name | `Remote-Email`  |
| `DOZZLE_AUTH_HEADER_NAME`<br>`--auth-header-name`     | Header carrying the display name.            | a header name | `Remote-Name`   |
| `DOZZLE_AUTH_HEADER_FILTER`<br>`--auth-header-filter` | Header carrying the user's container filter. | a header name | `Remote-Filter` |
| `DOZZLE_AUTH_HEADER_ROLES`<br>`--auth-header-roles`   | Header carrying the user's roles.            | a header name | `Remote-Roles`  |

## Subcommands

### generate

Generates a `users.yml` entry for [simple authentication](/guide/authentication/simple). The username is the first argument.

```sh
docker run -it --rm amir20/dozzle generate admin --password password --email test@email.net --name "John Doe" --user-filter name=foo --user-roles shell > users.yml
```

| Flag               | Description                                                                   | Values                                                                  |
| ------------------ | ----------------------------------------------------------------------------- | ----------------------------------------------------------------------- |
| `--password`, `-p` | The user's password.                                                          | a string                                                                |
| `--email`, `-e`    | The user's email, used for the avatar.                                        | an email                                                                |
| `--name`, `-n`     | The user's display name.                                                      | a string                                                                |
| `--user-filter`    | Containers the user can see.                                                  | comma-separated `key=value` filters                                     |
| `--user-roles`     | What the user can do. Prefix a role with `^` to remove it, e.g. `all,^shell`. | `all`, `none`, `shell`, `actions`, `download`, `notifications`, `cloud` |

### agent

Runs Dozzle as an [agent](/guide/agent) that another Dozzle instance connects to.

```sh
docker run -v /var/run/docker.sock:/var/run/docker.sock -p 7007:7007 amir20/dozzle agent
```

| Variable                              | Description                            | Values      | Default |
| ------------------------------------- | -------------------------------------- | ----------- | ------- |
| `DOZZLE_AGENT_ADDR`<br>`--agent-addr` | Address and port the agent listens on. | `host:port` | `:7007` |

### generate-certs

Generates a unique certificate and key for agent connections, in place of the shared one bundled with Dozzle. See [custom certificates](/guide/agent#custom-certificates).

| Flag         | Description                     | Default           |
| ------------ | ------------------------------- | ----------------- |
| `--cert-out` | Where to write the certificate. | `dozzle_cert.pem` |
| `--key-out`  | Where to write the private key. | `dozzle_key.pem`  |
| `--force`    | Overwrites existing files.      | `false`           |

### healthcheck

Checks that the server or agent is up. It is not wired into the image by default because it adds a little CPU overhead. See [healthcheck](/guide/healthcheck).
