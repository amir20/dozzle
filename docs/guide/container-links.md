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

For containers that publish a port and have no label yet, Dozzle shows a faint link icon in the container list and on the container page. It opens a snippet you can copy into your compose file, prefilled with the published port. Dismissing it hides the hint everywhere.

## Related labels

- [`dev.dozzle.name`](/guide/container-names) sets a custom display name
- [`dev.dozzle.group`](/guide/container-groups) groups containers together
- [`dev.dozzle.icon`](/guide/app-icons) picks the app icon
