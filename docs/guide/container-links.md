---
title: Container Links
---

# Container Links

Most containers worth watching also serve a web UI. Add a `dev.dozzle.url` label and Dozzle shows a link to it next to the container name, so you can jump from the logs to the app itself.

::: code-group

```sh
docker run --label dev.dozzle.url=https://grafana.example.com grafana/grafana
```

```yaml [docker-compose.yml]
services:
  grafana:
    image: grafana/grafana
    labels:
      - dev.dozzle.url=https://grafana.example.com
```

:::

The link shows up in three places: the sidebar, the container table, and the title bar on the container page. It always opens in a new tab, and clicking it never navigates away from the logs.

## What the label accepts

The value must be an absolute `http` or `https` URL. Anything else, a relative path, a bare hostname, or another scheme, is ignored and no link is rendered.

Dozzle does not check that the URL resolves, and it does not rewrite it per host. Whatever you write is what the link opens, so use an address that works from the browser you view Dozzle in.

## Why there is no auto detection

Dozzle knows which ports a container publishes, but a published port is not the same as a reachable URL. Reverse proxies, custom paths, TLS, and split networks all make the guess wrong often enough to be annoying. The label keeps it explicit: Dozzle only shows the link you wrote down.

For containers with no label yet, Dozzle shows a faint link icon next to the name on the container page. It opens a snippet you can copy into your compose file, prefilled with a guess. Dismissing it hides the hint everywhere.

The guess comes from two places. Traefik router labels are read first, because a router rule names an address that really does reach the container from a browser:

```yaml
labels:
  - traefik.http.routers.grafana.rule=Host(`grafana.example.com`)
  - traefik.http.routers.grafana.tls=true
```

Path prefixes are appended, the scheme follows the router's TLS settings and entrypoints, and `traefik.enable=false` turns the whole thing off. Failing that, Dozzle falls back to a published host port paired with the hostname you are viewing Dozzle on. Both are only ever prefilled into the snippet. Neither becomes a link until you write it into `dev.dozzle.url` yourself.

Containers behind a reverse proxy publish no host port at all, so the Traefik labels are usually the only signal there is. If you use a different proxy, the hint stays hidden and you add the label by hand.

## Swarm

In Swarm, `deploy.labels` sets labels on the service and the top-level `labels` key sets them on the container. Traefik's swarm provider reads service labels, so that is where everyone puts them:

```yaml
services:
  ui:
    image: my/ui
    deploy:
      labels:
        - traefik.http.routers.ui.rule=Host(`app.example.com`)
        - dev.dozzle.url=https://app.example.com
```

Dozzle reads service labels back onto each task container, so `dev.dozzle.url` and the Traefik hint both work from `deploy.labels`. The same goes for `dev.dozzle.name`, `dev.dozzle.group` and `dev.dozzle.icon`. A label set on the container itself wins over the service's.

Listing services is a manager-only API. In a multi-node swarm the agents on worker nodes cannot read service labels, so containers scheduled there see only their own.

## Related labels

- [`dev.dozzle.name`](/guide/container-names) sets a custom display name
- [`dev.dozzle.group`](/guide/container-groups) groups containers together
- [`dev.dozzle.icon`](/guide/app-icons) picks the app icon
