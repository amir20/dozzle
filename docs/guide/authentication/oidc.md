---
title: OpenID Connect
---

# <Icon icon="mdi:shield-account" inline /> OpenID Connect

With `--auth-provider oidc`, your identity provider is the user database. Dozzle never reads `users.yml` in this mode: who a user is, which roles they have and which containers they may see all come from the OpenID Connect token. Add a user in Keycloak, Authentik, Zitadel or Pocket ID and they can sign in; remove their role and they cannot.

This is different from [signing `users.yml` users in with OIDC](/guide/authentication/oauth#sign-in-with-oidc) under the `simple` provider. There the provider only proves who you are and `users.yml` still decides what you get. Pick `simple` when you want to list every user by hand, and `oidc` when the provider should own the list.

## Minimum configuration

Register Dozzle as a confidential client with your provider and set the redirect URI to:

```
https://your-dozzle-host/api/auth/callback
```

Include the base path if Dozzle runs under one, for example `https://example.com/dozzle/api/auth/callback`. Then point Dozzle at the issuer:

::: code-group

```sh [cli]
$ docker run -v /var/run/docker.sock:/var/run/docker.sock -v /path/to/dozzle/data:/data -p 8080:8080 amir20/dozzle --auth-provider oidc --auth-oidc-issuer https://keycloak.example.com/realms/main --auth-oidc-client-id dozzle --auth-oidc-client-secret secret
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
      DOZZLE_AUTH_PROVIDER: oidc
      DOZZLE_AUTH_OIDC_ISSUER: https://keycloak.example.com/realms/main
      DOZZLE_AUTH_OIDC_CLIENT_ID: dozzle
      DOZZLE_AUTH_OIDC_CLIENT_SECRET: secret
```

:::

That is all of it. No claim paths need to be set for the common layouts, and `DOZZLE_AUTH_OIDC_NAME` only changes the label on the login button. The client secret also accepts a `_FILE` counterpart, see [Using Docker secrets](/guide/authentication/oauth#using-docker-secrets-for-the-client-secret).

The issuer URL is the one that serves `/.well-known/openid-configuration`. Dozzle fetches that document to find the authorization, token and userinfo endpoints, and refuses to start the flow if the document names a different issuer than the one you configured.

## Roles

A user's roles are read from the token. Dozzle tries these claims in order and takes the first one that exists:

1. `dozzle_roles`
2. `resource_access.<client-id>.roles`
3. `roles`

The client id is already configured, so with `DOZZLE_AUTH_OIDC_CLIENT_ID=dozzle` the second path is `resource_access.dozzle.roles`, which is where Keycloak puts client roles. Nothing else needs to be set for that layout.

If your roles live somewhere else, `DOZZLE_AUTH_OIDC_ROLES_CLAIM` replaces the search with the one dot-separated path you give it:

```yaml
DOZZLE_AUTH_OIDC_ROLES_CLAIM: realm_access.roles
```

Each claim is looked for in the ID token first and then in the userinfo response, so it does not matter which of the two your provider puts it in. Three shapes are accepted: an array of strings, a single comma or space separated string, and an object whose keys are the roles, which is how Zitadel encodes project roles.

The role names are the same as in `users.yml`: `shell`, `actions`, `download`, `notifications`, `cloud` and `all`, with `^` to subtract, so `all,^shell` grants everything except shell access. Names prefixed with `dozzle_` are accepted too, which helps when the provider shares one roles claim across several applications. See [roles](/guide/authentication/simple#setting-specific-roles-for-users) for what each one unlocks.

> [!WARNING]
> `groups` is deliberately not on the list. At Authentik or Google every user belongs to at least one group, so searching it would turn "denied" into "signed in and can read every container". If groups are what you have, map them into a `dozzle_roles` claim at the provider, see the examples below.

### When a login is refused

Sign in is rejected when none of the claims exist, or when the first one that exists is empty. The log names the paths that were tried:

```
WRN OIDC login rejected: no roles claim found in the ID token or userinfo, or it was empty sub=... tried="dozzle_roles, resource_access.dozzle.roles, roles"
```

A claim that is present but contains nothing Dozzle recognizes as a role is different. That user signs in with no privileges, the same as `roles: none` in `users.yml`: they can read logs for the containers their filter allows, and nothing more. Keycloak realm roles behave this way, because every user carries `offline_access` and `uma_authorization` there, which is why the examples below use client roles instead.

## Filters

Container filters work the same way, read from the first of `dozzle_filters`, `resource_access.<client-id>.filters` and `filters` that exists, or from the single path in `DOZZLE_AUTH_OIDC_FILTERS_CLAIM`. Each value is one filter in the [same syntax as `users.yml`](/guide/authentication/simple#setting-specific-filters-for-users), for example `label=com.example.app` or `name=web`:

```json
"resource_access": {
  "dozzle": {
    "roles": ["shell", "actions"],
    "filters": ["label=com.example.app"]
  }
}
```

A user with no filters claim can see every container the Dozzle instance can. A filter that does not parse fails the login rather than being dropped, so a typo at the provider cannot quietly widen what someone sees.

## Identity

The `sub` claim is the user's stable id. It keys the profile directory under `/data`, so settings follow the person even if their username or email changes at the provider. The name shown in the menu is `name`, falling back to `preferred_username`, then `email`, then `sub`. `email` and `picture` feed the avatar, and a `picture` URL is used directly when the provider sends one.

Unlike the `simple` provider, the email does not have to be verified here. It is only displayed, never matched against anything, so an issuer that grants no email scope works fine.

## Sessions

After sign in, Dozzle issues its own session cookie carrying the roles and filters it read from the token. They are applied on every request, but they are not fetched again: a role change at the provider takes effect at the user's next login. If that gap matters, set [`--auth-ttl`](/guide/supported-env-vars) to something like `8h` so sessions expire and are re-established from a fresh token.

## Logout

Logging out clears Dozzle's session. If `--auth-logout-url` is set, the browser is then sent there, so point it at your provider's end session URL to sign the user out of the provider as well:

```yaml
DOZZLE_AUTH_LOGOUT_URL: https://keycloak.example.com/realms/main/protocol/openid-connect/logout
```

## What is different from `simple`

Both providers share the `--auth-oidc-*` flags, so the split shows up in behavior:

- `users.yml` is never read. If one exists under `/data`, Dozzle logs that it is ignoring it.
- There is no password form and no `/api/token` endpoint. The identity provider is the only way in, so a wrong callback URL or an expired client secret locks everyone out until it is fixed.
- `--auth-github-*` is a startup error. GitHub is not an OpenID Connect issuer and publishes no claims to read roles from.
- Startup logs the issuer and the claim paths roles will be read from.

## Provider examples

### Keycloak

Create a client `dozzle` in your realm with client authentication on, and add the redirect URI above. Then, under the client's **Roles** tab, create the client roles you want to hand out: `shell`, `actions`, `download`, `notifications`, `cloud`, or `all`. Assign them to users or groups under **Role mapping**.

Keycloak emits client roles as `resource_access.<client-id>.roles`, which Dozzle already searches. Check the client's **Client scopes**, open the dedicated scope and confirm the **client roles** mapper adds the claim to the ID token or to userinfo; Dozzle reads both but not the access token.

For filters, add a user attribute `dozzle_filters` and a **User Attribute** mapper on the dedicated scope with the same token claim name and **Multivalued** on. Each value is one filter, such as `label=com.example.app`.

### Authentik

Add a scope mapping under **Customization** → **Property Mappings** that returns the roles from the user's groups, and attach it to the Dozzle provider:

```python
roles = []
if request.user.ak_groups.filter(name="dozzle-admins").exists():
    roles.append("all")
elif request.user.ak_groups.filter(name="dozzle-users").exists():
    roles.append("download")
return {"dozzle_roles": roles}
```

A user in neither group gets an empty list and is refused.

### Zitadel

Grant users project roles named after Dozzle roles and enable **Assert Roles on Authentication** on the application. Zitadel's role claim is an object keyed by role name, which Dozzle accepts, so set:

```yaml
DOZZLE_AUTH_OIDC_ROLES_CLAIM: urn:zitadel:iam:org:project:roles
```

### Google

Google's tokens carry no roles claim, so `oidc` cannot be used with it. Use the [`simple` provider with Google sign in](/guide/authentication/oauth#google) instead, where `users.yml` decides who gets in.
