---
title: App Icons
---

# App Icons

Dozzle matches well known container images to their project logo and shows it next to the container name in the sidebar, the container table, and the command palette. If you run an \*arr stack, Plex, or Home Assistant, the list becomes a lot faster to scan.

Icons are bundled with Dozzle. They are never fetched from a CDN, so nothing about your containers leaves your network and everything works air gapped.

## Turning it off

The toggle lives under **Settings → Options → Show app icons**. It is a per profile setting, so it applies to your browser only.

## How matching works

Dozzle looks at the image name, ignoring the registry, tag, and digest. The last path segment wins, so all of these resolve to Sonarr:

- `sonarr`
- `linuxserver/sonarr:latest`
- `lscr.io/linuxserver/sonarr`
- `ghcr.io/hotio/sonarr@sha256:...`

When the image name is generic, Dozzle falls back to the namespace. `ghcr.io/goauthentik/server` resolves to Authentik that way.

## Overriding the icon

Some images do not match, and a fork can match the wrong logo. Set the `dev.dozzle.icon` label to pick an icon yourself, or to `none` to hide it for that container.

::: code-group

```sh
docker run --label dev.dozzle.icon=plex my-custom-media-server
```

```yaml [docker-compose.yml]
services:
  media:
    image: my-custom-media-server
    labels:
      - dev.dozzle.icon=plex

  scratch:
    image: alpine
    labels:
      - dev.dozzle.icon=none
```

:::

The value is an icon name from [dashboard-icons](https://github.com/homarr-labs/dashboard-icons). Only the icons Dozzle bundles are available. An unknown name falls back to no icon.

## Missing an icon?

Dozzle ships a curated subset rather than the full 3,000 icon set, to keep the image small. If something popular is missing, [open an issue](https://github.com/amir20/dozzle/issues) with the image name and it can be added.
