---
title: Sign in with GitHub & OIDC
---

# <Icon icon="mdi:shield-account" inline /> Sign in with GitHub & OIDC

Dozzle can let users sign in with an external account instead of typing a password. This is part of the [`simple`](/guide/authentication/simple) provider rather than a provider of its own, so `users.yml` is still read on every request and still decides who gets in.

That has one consequence worth stating up front: **`users.yml` is the allowlist.** An external account that no entry links to cannot sign in, and no account is ever created automatically.

Password login keeps working alongside it, which matters when an OAuth app breaks and you need to get in to fix it.

## Sign in with GitHub

Dozzle can let users sign in with their GitHub account instead of typing a password. This is part of the `simple` provider and not a separate auth provider, so `users.yml` is still read on every request and still decides who gets in. Keep using `--auth-provider simple`. `github` is accepted as an alias if you prefer to spell out what the instance uses.

First create an OAuth App under [Developer settings](https://github.com/settings/developers) on GitHub and set the **Authorization callback URL** to:

```
https://your-dozzle-host/api/auth/callback
```

If Dozzle is served under a [base path](/guide/changing-base), include it, for example `https://example.com/dozzle/api/auth/callback`. Dozzle does not send a `redirect_uri` when it starts the flow, so GitHub always redirects to the callback URL registered on the OAuth App. A mismatch here is the most common reason sign in fails.

Then copy the client ID, generate a client secret, and pass both to Dozzle:

::: code-group

```sh [cli]
$ docker run -v /var/run/docker.sock:/var/run/docker.sock -v /path/to/dozzle/data:/data -p 8080:8080 amir20/dozzle --auth-provider simple --auth-github-client-id Ov23liABCDEFGHIJKLMN --auth-github-client-secret 0123456789abcdef0123456789abcdef01234567
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
      DOZZLE_AUTH_GITHUB_CLIENT_ID: Ov23liABCDEFGHIJKLMN
      DOZZLE_AUTH_GITHUB_CLIENT_SECRET: 0123456789abcdef0123456789abcdef01234567
```

:::

Link a user to their GitHub account with a `github` key in `users.yml`:

```yaml
users:
  admin:
    email: me@email.net
    name: Admin
    password: $2a$11$9ho4vY2LdJ/WBopFcsAS0uORC0x2vuFHQgT/yBqZyzclhHsoaIkzK
    github: octocat

  guest:
    email: guest@email.net
    name: Guest
    github: hubot
    filter: "label=com.example.app"
    roles: none
```

`password` is optional once `github` is set, as `guest` shows above. `admin` has both, so they can sign in either way. Password login stays available as a fallback for anyone who still has a password, and the login page shows both options.

The value is the GitHub **login** (the handle in `github.com/octocat`), not the email address. Logins are stable and always visible, while an account's email can be private or changed at any time.

`users.yml` is the allowlist. A GitHub account that is not listed in `users.yml` cannot sign in, no matter which org it belongs to. There is no auto-provisioning: adding someone means adding them to the file. Filters and roles are resolved from `users.yml` on every request, exactly as they are for password users, so a GitHub user with `roles: none` is restricted the same way.

> [!NOTE]
> Dozzle intentionally does not support allowlisting a whole GitHub org or an entire email domain. Every user is listed individually. If you need group or domain based access, use `forward-proxy` with [Authelia](/guide/authentication/forward-proxy#setting-up-dozzle-with-authelia) or Authentik, which are built for that.

> [!WARNING]
> Editing `users.yml` rotates the JWT signing key and signs every user out. This is already the case today when you add or remove a user, and it applies when you add a `github` key too.

## Sign in with OIDC

Any provider that publishes an OpenID Connect discovery document works through the same callback: Google, Keycloak, Pocket ID, Zitadel, Authentik and others. Point Dozzle at the issuer URL and give it a client id and secret.

Register Dozzle as a confidential client with your provider and set the redirect URI to:

```
https://your-dozzle-host/api/auth/callback
```

Include the base path if Dozzle runs under one, for example `https://example.com/dozzle/api/auth/callback`. Unlike GitHub, OIDC requires Dozzle to send `redirect_uri`, so this value has to match what you registered exactly.

::: code-group

```sh [cli]
$ docker run -v /var/run/docker.sock:/var/run/docker.sock -v /path/to/dozzle/data:/data -p 8080:8080 amir20/dozzle --auth-provider simple --auth-oidc-issuer https://id.example.com --auth-oidc-client-id dozzle --auth-oidc-client-secret secret --auth-oidc-name "Pocket ID"
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
      DOZZLE_AUTH_OIDC_ISSUER: https://id.example.com
      DOZZLE_AUTH_OIDC_CLIENT_ID: dozzle
      DOZZLE_AUTH_OIDC_CLIENT_SECRET: secret
      DOZZLE_AUTH_OIDC_NAME: Pocket ID
```

:::

`DOZZLE_AUTH_OIDC_NAME` is only the label on the login button. It defaults to `SSO`.

The issuer URL is the one that serves `/.well-known/openid-configuration`. Dozzle fetches that document to find the authorization, token and userinfo endpoints, and refuses to start the flow if the document names a different issuer than the one you configured.

### Linking users

OIDC matches on the **verified email**, not the login. Set `email` on the user in `users.yml`:

```yaml
users:
  admin:
    email: me@email.net
    name: Admin
    # password is optional once the account is linked
```

The email must be marked verified by your provider. Dozzle rejects a sign-in when `email_verified` is false, because matching an unverified address would let anyone who can register at a permissive provider claim an account by typing in someone else's email.

> [!NOTE]
> GitHub matches on the login and OIDC matches on the email, and that difference is deliberate. A GitHub login is stable and always present, while a GitHub email can be private or changed. OIDC has no stable human-readable equivalent, so the verified email is the claim operators actually know.

### Google

Google is a normal OIDC provider. Create an OAuth client in the Google Cloud console and use:

```
DOZZLE_AUTH_OIDC_ISSUER: https://accounts.google.com
DOZZLE_AUTH_OIDC_NAME: Google
```

`--auth-provider google` is accepted as an alias for `simple`, so either spelling works.

## Using Docker secrets for the client secret

Putting a client secret straight into `environment:` means it shows up in `docker inspect`, in your compose file, and in the shell history of anyone who started the container by hand. Both client secrets therefore accept a `_FILE` counterpart naming a file to read the value from, which is the convention Docker's own images use:

| Instead of                         | Use                                     |
| ---------------------------------- | --------------------------------------- |
| `DOZZLE_AUTH_GITHUB_CLIENT_SECRET` | `DOZZLE_AUTH_GITHUB_CLIENT_SECRET_FILE` |
| `DOZZLE_AUTH_OIDC_CLIENT_SECRET`   | `DOZZLE_AUTH_OIDC_CLIENT_SECRET_FILE`   |

Dozzle reads the file at startup and trims surrounding whitespace, so a trailing newline from `echo secret > file` is fine. Setting both a variable and its `_FILE` counterpart is an error rather than a silent preference for one, and pointing `_FILE` at a missing or empty file stops Dozzle at startup instead of quietly disabling the login button.

### Docker Compose

Outside Swarm there is no `docker secret create`, so a Compose secret is either a file on disk or an environment variable. The environment form is usually what you want: it pairs with a gitignored `.env`, and there is no plaintext file sitting next to your compose file waiting to be committed.

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    ports:
      - 8080:8080
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./data:/data
    environment:
      DOZZLE_AUTH_PROVIDER: simple
      DOZZLE_AUTH_GITHUB_CLIENT_ID: Ov23liABCDEFGHIJKLMN
      DOZZLE_AUTH_GITHUB_CLIENT_SECRET_FILE: /run/secrets/dozzle_github_secret
    secrets:
      - dozzle_github_secret

secrets:
  dozzle_github_secret:
    environment: GITHUB_CLIENT_SECRET
```

```ini [.env]
GITHUB_CLIENT_SECRET=your-github-client-secret
```

Compose reads the variable itself and mounts the value at `/run/secrets/dozzle_github_secret`. It never becomes part of the container's environment, so it stays out of `docker inspect` the same way a file-backed secret does.

Use the file form instead when the secret already exists as a file, for example one written by a secret manager:

```yaml [docker-compose.yml]
secrets:
  dozzle_github_secret:
    file: /run/secrets/github_client_secret
```

Either way Dozzle trims surrounding whitespace, so a trailing newline in the file does not matter.

### Docker Swarm

In Swarm the secret is managed by the cluster rather than a file on disk, so declare it as `external` and create it with `docker secret create`:

```sh
printf '%s' 'your-oidc-client-secret' | docker secret create dozzle_oidc_secret -
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      DOZZLE_MODE: swarm
      DOZZLE_AUTH_PROVIDER: simple
      DOZZLE_AUTH_OIDC_ISSUER: https://id.example.com
      DOZZLE_AUTH_OIDC_CLIENT_ID: dozzle
      DOZZLE_AUTH_OIDC_CLIENT_SECRET_FILE: /run/secrets/dozzle_oidc_secret
      DOZZLE_AUTH_OIDC_NAME: Pocket ID
    secrets:
      - dozzle_oidc_secret
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    deploy:
      mode: global

secrets:
  dozzle_oidc_secret:
    external: true
```

> [!NOTE]
> A secret mounts at `/run/secrets/<name>` by default, which is why `_FILE` points there. If you set an explicit `target:`, point `_FILE` at that path instead.

### Verifying it worked

Dozzle logs the providers it enabled at startup. Run with `--level debug` and look for the line naming the provider:

```sh
$ docker compose logs dozzle | grep -i 'sign in'
DBG Enabling Sign in with GitHub
```

If the secret file is missing or empty, Dozzle exits at startup with a message naming the variable, so a broken mount fails loudly instead of silently dropping the login button.
