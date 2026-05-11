---
title: Reverse Proxy & Base Path
---

# Reverse Proxy & Base Path

Dozzle is commonly placed behind a reverse proxy for TLS termination, authentication, or to share a hostname with other services. This page covers both mounting Dozzle at a sub-path and the proxy settings needed to make streaming work correctly.

## Changing the Base Path

Dozzle by default mounts to `/`. This can be changed with the `--base` flag or the `DOZZLE_BASE` environment variable. For example, to mount at `/foobar`:

::: code-group

```sh
docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --base /foobar
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
      DOZZLE_BASE: /foobar
```

:::

Dozzle will be available at `http://localhost:8080/foobar/`. This option rewrites all assets to `/foobar/{file.path}` and automatically redirects `/foobar` to `/foobar/`.

## Proxy Requirements

Dozzle streams logs over **Server-Sent Events (SSE)** and uses **WebSocket** for container shells. Reverse proxies must:

1. **Disable response buffering** — SSE delivers events as they happen. Any buffering causes logs to arrive in bursts or never arrive at all. Dozzle sends `X-Accel-Buffering: no`, but some proxies ignore it.
2. **Forward WebSocket upgrade headers** — required for the shell and attach features.
3. **Avoid compressing `text/event-stream`** — compression middleware often breaks SSE.

## Nginx

```nginx
location ^~ /foobar/ {
    proxy_pass http://dozzle:8080;

    chunked_transfer_encoding off;
    proxy_buffering off;
    proxy_cache off;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
}
```

Drop the `^~ /foobar/` prefix if Dozzle is mounted at the root. See also the FAQ entry on [disabling buffering](/guide/faq#disabling-buffering-in-nginx).

## Traefik

Traefik handles WebSocket upgrades automatically, but the default `compress` middleware will break SSE. Exclude `text/event-stream`:

```yaml
http:
  middlewares:
    middlewares-compress:
      compress:
        excludedContentTypes:
          - text/event-stream
```

Then a typical labels block on the Dozzle service:

```yaml
services:
  dozzle:
    image: amir20/dozzle:latest
    labels:
      - traefik.enable=true
      - traefik.http.routers.dozzle.rule=Host(`dozzle.example.com`)
      - traefik.http.routers.dozzle.entrypoints=websecure
      - traefik.http.routers.dozzle.tls.certresolver=letsencrypt
      - traefik.http.services.dozzle.loadbalancer.server.port=8080
```

## Caddy

```caddyfile
dozzle.example.com {
    reverse_proxy dozzle:8080 {
        flush_interval -1
    }
}
```

`flush_interval -1` disables response buffering for streaming endpoints.

## Common Pitfalls

- **Blank page or assets 404 when using `--base`** — the proxy is stripping the path prefix before forwarding. Configure it to pass the full path through to Dozzle.
- **Logs stop after a few seconds** — connection timeouts on the proxy are too short. Increase read/send timeouts to at least a few minutes (e.g. Nginx `proxy_read_timeout 3600s`).
- **Shell disconnects immediately** — WebSocket upgrade headers are not being forwarded. Verify `Upgrade` and `Connection` headers.
