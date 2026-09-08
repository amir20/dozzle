---
title: Authentication
---

# Authentication

Dozzle supports two configurations for authentication. In the first configuration, you bring your own authentication method by protecting Dozzle through a proxy. Dozzle can read appropriate headers out of the box.

If you do not have an authentication solution, then Dozzle has a simple file-based user management solution. Authentication providers are set up using the `--auth-provider` flag. In both configurations, Dozzle will try to save user settings to disk. This data is written to `/data`.

## <Icon icon="mdi:shield-alert-outline" inline /> Security Considerations

Dozzle has access to `docker.sock`, which — unless restricted — is equivalent to **root on the host**. Before exposing Dozzle beyond your private network, review the following:

- **Always put Dozzle behind authentication** if it is reachable from the public internet. Use `--auth-provider=simple` or a forward-proxy like Authelia / Authentik / Cloudflare Access.
- **Keep [actions](/guide/actions) and [shell access](/guide/shell) disabled** unless you need them. They allow starting, stopping, recreating, and executing arbitrary commands inside containers.
- **Restrict users with [roles](/guide/authentication/simple#setting-specific-roles-for-users) and [filters](/guide/authentication/simple#setting-specific-filters-for-users)** in multi-user mode. Without explicit roles, a user can see every container the Dozzle instance can.
- **Never expose Dozzle's port directly in forward-proxy mode.** Dozzle trusts `Remote-User` on every request, and when no roles header is present the user is granted all roles. Anyone who can reach the container without passing through the proxy authenticates as whoever they like by setting one header. Publish only the proxy, and keep Dozzle on an internal network with `expose` rather than `ports`.
- **Run TLS at the reverse proxy**. See [Reverse Proxy & Base Path](/guide/changing-base) for Nginx / Traefik / Caddy examples.
- **Restrict `docker.sock` access with a proxy** if you don't need actions. Note that a read-only mount (`/var/run/docker.sock:/var/run/docker.sock:ro`) does _not_ limit the API: the `:ro` flag only marks the socket file read-only on disk, while API calls still pass through the socket normally, so create/delete/update remain possible. To actually restrict operations, put a socket proxy like [`tecnativa/docker-socket-proxy`](https://github.com/Tecnativa/docker-socket-proxy) in front of the daemon.

## <Icon icon="mdi:key-outline" inline /> Choosing a method

| Method                                               | Who owns the users     | Use it when                                                                                                                          |
| ---------------------------------------------------- | ---------------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| [Simple](/guide/authentication/simple)               | Dozzle, in `users.yml` | You have no authentication solution and want Dozzle to handle logins.                                                                |
| [GitHub & OIDC](/guide/authentication/oauth)         | Dozzle, in `users.yml` | You want the same `users.yml` users to sign in with GitHub, Google, Keycloak, Pocket ID, Zitadel or Authentik instead of a password. |
| [Forward Proxy](/guide/authentication/forward-proxy) | Your proxy             | You already run Authelia, Authentik, Cloudflare Access or similar, and want it to own authentication entirely.                       |

Simple and OAuth are the same provider: `users.yml` is the user list either way, and OAuth only adds a second way to prove you are one of the users in it. Forward proxy is the separate one, and it is the right choice when you need org-wide or domain-wide access rules, which `users.yml` deliberately does not do.

## <Icon icon="mdi:file-document-edit-outline" inline /> Generating users.yml

Dozzle has a built-in `generate` command to generate `users.yml`. Here is an example:

```sh
docker run -it --rm amir20/dozzle generate admin --password password --email test@email.net --name "John Doe" --user-filter name=foo --user-roles shell > users.yml
```

In this example, `admin` is the username. Email and name are optional but recommended to display accurate avatars. `docker run -it --rm amir20/dozzle generate --help` displays all options. The `--user-filter` flag is a comma-separated list of filters. The `--user-roles` flag is a comma-separated list of roles.

If you omit `--password`, Dozzle prompts for it on stdin so the password never lands in your shell history. This requires an interactive terminal, so keep the `-it` flags:

```sh
docker run -it --rm amir20/dozzle generate admin --email test@email.net --name "John Doe" > users.yml
```

The prompt is written to stderr, so redirecting stdout to `users.yml` still works. You can also pipe the password in, for example `echo "$PASSWORD" | docker run -i --rm amir20/dozzle generate admin > users.yml`.
