---
title: Setup Wizard
---

# Setup Wizard

<Badge type="warning" text="Docker Only" />

A fresh Dozzle install opens with a short setup wizard. It walks you through the few things most people change right after installing: turning on login, allowing container actions and shell access, and connecting Dozzle Cloud. Everything it saves can also be set with flags or environment variables, so the wizard is optional.

The wizard only appears on a fresh install running in server mode. Swarm and Kubernetes deployments never show it. You can open it again later from Settings.

## <Icon icon="mdi:format-list-numbered" inline /> Steps

### 1. Login

Login comes first, so nothing else can be changed on an instance anyone can reach.

The wizard first checks that `/data` is mounted on a volume. Settings and users are written there, and without a volume they would disappear the next time the container is recreated. If `/data` is not persisted, the wizard shows how to mount it and waits for you to click **Check again**.

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - dozzle_data:/data
    ports:
      - 8080:8080
volumes:
  dozzle_data:
```

Once `/data` is persisted, pick one of three options:

- **Dozzle account** creates a single user with a username, an optional email and a password. Dozzle writes `/data/users.yml` and sets `authProvider: simple`. See [Simple authentication](/guide/authentication/simple) to add more users or roles later.
- **My proxy** is for Authelia, Authentik, Cloudflare Access and similar. Dozzle trusts the `Remote-User` header, so publish only the proxy and never Dozzle's own port. This sets `authProvider: forward-proxy`. See [Forward Proxy](/guide/authentication/forward-proxy).
- **OIDC** shows a link to the [OpenID Connect](/guide/authentication/oidc) guide and the environment variables to add. OIDC needs a client secret, so nothing is written here and you configure it yourself.

If Dozzle is only reachable on your own network, **Continue without login** skips this step.

After an account or proxy is saved, Dozzle restarts right away so login is on before anything else is changed. You land on the login page, and the wizard continues with the next step once you sign in.

### 2. Actions and shell

Two toggles control what Dozzle is allowed to do to your containers:

- **Start, stop and restart** turns on [container actions](/guide/actions) (`enableActions`).
- **Shell** turns on [attaching and running commands](/guide/shell) inside containers (`enableShell`). It is off by default. Shell access to a container is often as good as access to the host, so only turn it on if you need it.

If a setting is already fixed by a flag or environment variable, its toggle is read-only and says so. Like login, these toggles need `/data` on a volume, so they stay read-only until it is.

### 3. Dozzle Cloud

[Dozzle Cloud](/guide/dozzle-cloud) sends alerts the moment something breaks, a morning summary of what to fix, and keeps history that survives restarts. **Connect Dozzle Cloud** links this instance, and **Not now** moves on. This step is skipped when the instance is already linked or when you are not allowed to link it.

### 4. Auto-update

Dozzle can keep itself up to date. Pick **Off**, **Daily** or **Weekly** (weekly runs on Sunday) and a time of day. The time is in the server's local time and defaults to `03:00`. At that time Dozzle checks its registry for a newer image and, only if there is one, [updates itself](#self-update).

This setting applies right away and does not need a restart.

Updating itself is an action, so while actions are off this step stays in the list but is greyed out with **Needs actions**. Turning actions on in step 2 makes it available right away. If this instance cannot update itself for another reason (for example it runs a pinned version tag), the step says why instead.

### 5. Restart

The last step lists the changes that are saved but not running yet. **Restart Dozzle** restarts the container, waits until it is back and reloads the page. If nothing is pending, the step just says you are done.

If Dozzle cannot restart itself (for example when it cannot find its own container), the wizard shows the environment variables to add to your compose file instead.

## <Icon icon="mdi:file-cog-outline" inline /> Where settings are saved

The wizard saves its choices to `/data/dozzle.yml`. Dozzle reads this file once at startup, which is why changes need a restart to apply. Dozzle restarts itself from the wizard, so you do not need to do it by hand. The auto-update keys are the exception: Dozzle checks them again every minute, so they apply without a restart.

```yaml [/data/dozzle.yml]
authProvider: simple
enableActions: true
enableShell: false
autoUpdate: weekly
autoUpdateTime: "03:00"
```

| Key              | Values                            | Same as                   |
| ---------------- | --------------------------------- | ------------------------- |
| `authProvider`   | `none`, `simple`, `forward-proxy` | `DOZZLE_AUTH_PROVIDER`    |
| `enableActions`  | `true`, `false`                   | `DOZZLE_ENABLE_ACTIONS`   |
| `enableShell`    | `true`, `false`                   | `DOZZLE_ENABLE_SHELL`     |
| `autoUpdate`     | `off`, `daily`, `weekly`          | `DOZZLE_AUTO_UPDATE`      |
| `autoUpdateTime` | `HH:MM`, server local time        | `DOZZLE_AUTO_UPDATE_TIME` |

Flags and environment variables always win over the file. If `DOZZLE_ENABLE_ACTIONS` is set, the value in `dozzle.yml` is ignored and the wizard shows the toggle as locked. To go back to managing a setting from the wizard, remove the variable from your compose file.

## <Icon icon="mdi:update" inline /> How self-update works {#self-update}

Dozzle updates itself from the `Update` action on its own container or on the auto-update schedule. Both do the same thing:

1. Dozzle pulls the image tag it is running. If the tag still points at the image already running, it stops there and reports it is up to date.
2. Dozzle starts a short-lived helper container from the new image, with access to the same Docker socket. Dozzle goes away a few seconds later.
3. The helper renames the old container and creates a replacement under the original name with the same configuration, networks and volumes. Only then does it stop the old container and start the replacement. Anonymous volumes are kept too, so data in `/data` survives even without a named volume.
4. The helper waits for the replacement to stay running (and healthy, if it has a healthcheck). If it does, the old container is removed and its volumes are left alone. If it does not, the replacement is removed and the old container is renamed back and started again.

Containers started with `--rm` update the same way. The old container deletes itself when it stops, but by then the replacement already holds its volumes, so they survive. If the update rolls back, the helper recreates the old container from its saved configuration.

The helper's logs are the only record of an update. It removes itself when it finishes, so to follow one, watch the `dozzle-self-update-*` container while it runs.

Some setups cannot update this way:

- **Actions must be on.** Self-update needs `DOZZLE_ENABLE_ACTIONS`, and the `Update` action needs the actions role when login is on.
- **Server mode only.** A Dozzle Swarm service updates through the Swarm manager like any other service. Kubernetes and Dozzle agents do not update themselves.
- **Pinned version tags never update.** Pulling `amir20/dozzle:v8.12.0` always returns the same image, so auto-update is unavailable and a manual update reports up to date. Use `latest` or change the tag yourself.

## <Icon icon="mdi:shield-lock-outline" inline /> Security

- **Login is the first step.** A restart after saving an account or proxy turns login on before any other setting can be changed.
- **Only a signed in user can change actions, shell, auto-update or restart Dozzle.** The user needs all roles.
- **Without login, only a new install gets a 15 minute window.** When `authProvider` is `none`, these settings can only be changed within 15 minutes of the first start of a new install, one whose `/data` was empty. An install that already has data from earlier runs never gets the window, so a reboot or an image update can't open it. Outside the window, use environment variables or turn on login.
- **Routes are still decided at startup.** The wizard only writes to `dozzle.yml`. The endpoints for actions and shell are registered when Dozzle starts, exactly as with environment variables, so nothing is enabled until Dozzle restarts.
