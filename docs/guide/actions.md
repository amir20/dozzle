---
title: Container Actions
---

# Container Actions

<Badge type="warning" text="Docker Only" />

Dozzle supports container actions, which allows you to `start`, `stop`, `restart`, `remove`, and `update` containers from the dropdown menu on the right next to the container stats. This feature is **disabled** by default and can be enabled by setting the environment variable `DOZZLE_ENABLE_ACTIONS` to `true`.

The `update` action pulls the latest image for the container and recreates it with the same configuration — useful for upgrading a container in place without editing its compose file. `update` only has a meaningful effect when the image uses a moving tag (e.g. `latest`, `stable`); a pinned tag will simply re-pull the same image.

> [!WARNING]
> `remove` and `update` recreate the container. Data written to **anonymous volumes** or the container's writable layer will be lost. Named volumes and bind mounts are preserved.

::: code-group

```sh
docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --enable-actions
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
      DOZZLE_ENABLE_ACTIONS: true
```

:::

## Update checking

Dozzle checks whether the image a container is running is still the one its registry serves. When they differ, a dot appears on the container's menu and the menu says an update is available.

The check asks the registry for the digest of the tag the container was created from, and compares it against the digest the container is actually running. It does this with a `HEAD` request for the image manifest, so no layers are downloaded and it does not count against Docker Hub pull rate limits. Answers are cached for six hours, and the same image is only ever looked up once no matter how many containers or hosts run it.

Because the comparison is against what the container is _running_, a container stays out of date until it is recreated, even if a newer image was already pulled onto the host.

Checking is separate from actions. Knowing a container is out of date is useful whether or not Dozzle is allowed to do anything about it, so the notice appears even when `DOZZLE_ENABLE_ACTIONS` is off. Only the `Update` button requires actions.

### Turning it off

`DOZZLE_IMAGE_CHECK_MODE` controls whether Dozzle contacts registries at all.

| Value       | Behavior                                                                 |
| ----------- | ------------------------------------------------------------------------ |
| `automatic` | Checks in the background when a container is viewed.                     |
| `manual`    | Never checks on its own. The menu offers a "Check for updates" action.   |
| `off`       | The feature is gone. No endpoint is registered and no requests are made. |

It defaults to whatever `DOZZLE_RELEASE_CHECK_MODE` is set to, so if you have already told Dozzle not to fetch releases automatically, it will not check images automatically either.

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      DOZZLE_IMAGE_CHECK_MODE: off
```

To silence a single container, such as one deliberately pinned to a version, label it:

```yaml [docker-compose.yml]
services:
  database:
    image: postgres:18-alpine
    labels:
      dev.dozzle.update-check: false
```

A notification can also be shown when an update is found. It is off by default and lives under Settings.

### What cannot be checked

Some containers have nothing to compare, and Dozzle stays quiet rather than guessing:

- Images built locally, which carry no registry digest
- References pinned to a digest, which cannot drift
- Private registries, since Dozzle has no credentials of its own. In Kubernetes that includes images pulled with `imagePullSecrets`.

### Kubernetes

In Kubernetes mode the check compares the digest in the pod's status against the registry, so it works on containerd clusters such as k3s, EKS and GKE without extra permissions. The notice is informational and never comes with an `Update` button, because a pod's image belongs to its workload. For a moving tag like `:latest` with `imagePullPolicy: Always`, a [rollout restart](/guide/k8s#rollout-restart) picks up the new image. Anything else is a change to the workload's spec. The dashboard's updates button and drawer are not shown in Kubernetes mode.

### Updating Dozzle itself

The `Update` action on Dozzle's own container updates Dozzle in place. It pulls the new image and hands the swap to a short-lived helper container, so Dozzle goes away for a few seconds and comes back on the new version with the same configuration and volumes. It can also run on a schedule. See [How self-update works](/guide/setup-wizard#self-update) for what is kept and the setups it does not cover. Running Dozzle as a Swarm service updates through the orchestrator. Dozzle agents on other hosts are ordinary containers and update like anything else.

## Updating several containers at once

With actions on, the dashboard checks every container in one pass. Out-of-date containers get a small ring next to their name, and an **N updates** button appears above the container list. Both open the Updates drawer, which lists every container with a newer image, all selected. Untick anything you want to leave alone, then press **Update**.

Hosts update in parallel, and each host updates one container at a time, so no daemon has to pull a dozen images at once. The drawer shows each container moving through pulling, recreating and updated, and a failure on one container does not stop the rest. The update runs on the server, so closing the tab does not interrupt it. Opening the drawer again picks up where it is.

If Dozzle's own container is in the list, it always goes last, because updating it restarts Dozzle.

With `DOZZLE_IMAGE_CHECK_MODE=manual`, the button reads **Check for updates** until you press it. With actions off, the dashboard looks exactly as it does today, and each container's own menu still says when an update is available.

## Auto-updating containers

Dozzle can update containers on a schedule. Opt a container in with a label:

```yaml [docker-compose.yml]
services:
  whoami:
    image: traefik/whoami:latest
    labels:
      dev.dozzle.auto-update: true
```

Labelled containers follow the same schedule as [Dozzle's own auto-update](/guide/setup-wizard#_4-auto-update), which you set in the setup wizard or with `DOZZLE_AUTO_UPDATE` and `DOZZLE_AUTO_UPDATE_TIME`. At that time Dozzle checks each labelled container against its registry and updates only the ones with a newer image. Containers go first and Dozzle goes last.

Auto-update is opt in on purpose. A database on a floating tag like `postgres:latest` can move to a new major version that its data files cannot read, so only label containers you are happy to see replaced without watching. Containers Dozzle [cannot check](#what-cannot-be-checked), such as ones from a private registry, are never auto-updated.

Auto-update runs in server mode, including containers on [remote agents](/guide/agent). It needs actions on.
