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

### 4. Restart

The last step lists the changes that are saved but not running yet. **Restart Dozzle** restarts the container, waits until it is back and reloads the page. If nothing is pending, the step just says you are done.

If Dozzle cannot restart itself (for example when it cannot find its own container), the wizard shows the environment variables to add to your compose file instead.

## <Icon icon="mdi:file-cog-outline" inline /> Where settings are saved

The wizard saves its choices to `/data/dozzle.yml`. Dozzle reads this file once at startup, which is why changes need a restart to apply. Dozzle restarts itself from the wizard, so you do not need to do it by hand.

```yaml [/data/dozzle.yml]
authProvider: simple
enableActions: true
enableShell: false
```

| Key             | Values                            | Same as                 |
| --------------- | --------------------------------- | ----------------------- |
| `authProvider`  | `none`, `simple`, `forward-proxy` | `DOZZLE_AUTH_PROVIDER`  |
| `enableActions` | `true`, `false`                   | `DOZZLE_ENABLE_ACTIONS` |
| `enableShell`   | `true`, `false`                   | `DOZZLE_ENABLE_SHELL`   |

Flags and environment variables always win over the file. If `DOZZLE_ENABLE_ACTIONS` is set, the value in `dozzle.yml` is ignored and the wizard shows the toggle as locked. To go back to managing a setting from the wizard, remove the variable from your compose file.

## <Icon icon="mdi:shield-lock-outline" inline /> Security

- **Login is the first step.** A restart after saving an account or proxy turns login on before any other setting can be changed.
- **Only a signed in user can change actions, shell or restart Dozzle.** The user needs all roles.
- **Without login, there is a 15 minute window.** When `authProvider` is `none`, these settings can only be changed within 15 minutes of Dozzle starting. After that, the wizard is read-only until you turn on login or restart Dozzle.
- **Routes are still decided at startup.** The wizard only writes to `dozzle.yml`. The endpoints for actions and shell are registered when Dozzle starts, exactly as with environment variables, so nothing is enabled until Dozzle restarts.
