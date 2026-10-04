---
title: Container Actions
---

# Container Actions

<Badge type="warning" text="Docker Only" />

Dozzle supports container actions, which allows you to `start`, `stop`, `restart`, `remove`, and `update` containers from the dropdown menu on the right next to the container stats. This feature is **disabled** by default and can be enabled by setting the environment variable `DOZZLE_ENABLE_ACTIONS` to `true`.

The `update` action pulls the latest image for the container and recreates it with the same configuration — useful for upgrading a container in place without editing its compose file. `update` only has a meaningful effect when the image uses a moving tag (e.g. `latest`, `stable`); a pinned tag will simply re-pull the same image.

The old container is kept, renamed, until the new one has run for 10 seconds without restarting, and has reported healthy if its image has a healthcheck. If the new container fails to start, exits, restarts or turns unhealthy, Dozzle removes it and puts the old one back, and the update reports **rolled back** with the reason. The new container is labelled `dev.dozzle.previous-image` with the image id it replaced, and `dev.dozzle.previous-ref` with that image's `repo@sha256:…` digest (absent for images built locally). A stopped container is never updated, since it may be stopped on purpose. It still shows when a newer image is available, but **Update** is not offered for it, the schedule skips it, and an update asked for anyway, from Dozzle Cloud for one, is refused with "Start the container first". Start it, and it can be updated.

> [!WARNING]
> `remove` deletes the container: data in its writable layer is lost, and its anonymous volumes are left behind, detached. `update` recreates the container and keeps every volume, anonymous ones included, and every bind mount. Only data written to the container's writable layer is lost.

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

To stop checking a single container, such as one deliberately pinned to a version, label it. That also keeps it off the [auto-update schedule](#auto-updating-containers).

```yaml [docker-compose.yml]
services:
  database:
    image: postgres:18-alpine
    labels:
      dev.dozzle.update: off
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

## Auto-updating containers {#auto-updating-containers}

Dozzle can update containers on a schedule. Set it up under **Settings → Updates** or in the [setup wizard](/guide/setup-wizard#auto-update):

- **When:** off, daily or weekly on Sunday, at a time of day. The same as `DOZZLE_AUTO_UPDATE` and `DOZZLE_AUTO_UPDATE_TIME`.
- **Which containers:** **Dozzle only**, **Labelled containers** (the default) or **Everything**. Dozzle itself follows the schedule in all three. The same as `DOZZLE_UPDATE_CONTAINERS` (`off`, `labelled` or `all`).

One label on a container decides the rest:

| `dev.dozzle.update` | What happens                                                                                            |
| ------------------- | ------------------------------------------------------------------------------------------------------- |
| `auto`              | Updated on the schedule, unless **Which containers** is **Dozzle only**                                 |
| _(no label)_        | Updated on the schedule under **Everything**. Otherwise checked and shown as an update, which you apply |
| `off`               | Never checked, never updated                                                                            |

```yaml
services:
  app:
    image: ghcr.io/example/app:latest
    labels:
      dev.dozzle.update: auto
```

Older labels still work: `dev.dozzle.auto-update=true` reads as `auto`, and `dev.dozzle.update-check=false` as `off`.

At the scheduled time Dozzle checks each container on the schedule against its registry and updates only the ones with a newer image, Dozzle itself last. Every update is the safe swap described above, so a new container that fails to stay up is replaced by the old one again. Containers that are stopped or unhealthy, that Dozzle [cannot check](#what-cannot-be-checked), or that were [rolled back](#rolling-back) from the image on offer are skipped. **Settings → Updates** lists the containers the next run will update.

Under **Everything**, a database on a floating tag like `postgres:latest` can move to a major version its data files cannot read. Picking **Everything** lists the containers that keep data in named volumes. Label those `dev.dozzle.update: off` to keep them out.

**Which containers** is saved in [`dozzle.yml`](/guide/setup-wizard#dozzle-yml) as `updateContainers`, so changing it from the UI needs `/data` on a volume. Auto-update runs in server mode, including containers on [remote agents](/guide/agent), and needs actions on. Coming from Watchtower? See [Moving from Watchtower](/guide/moving-from-watchtower).

## Cleaning up old images {#cleaning-up-old-images}

Every update leaves the image it replaced on the host, so Dozzle removes old images after an update, like Watchtower's `--cleanup`. It always runs, for every update: scheduled, from a container's `Update` action, or from the Updates drawer. There is nothing to turn on.

Dozzle keeps the image the container ran until now, so the container can still go back to it, and removes the one before that. An update from 1.4.1 to 1.4.2 removes 1.4.0 and keeps 1.4.1, so each container keeps at most one spare image. Dozzle reads which image to remove from the old container's `dev.dozzle.previous-image` label, so a container's first update removes nothing.

Cleanup only runs after the update has gone through and the old container is gone. A rolled back update removes nothing. Dozzle only removes an untagged image that no container uses: an image that still has a tag, such as one you pulled or built yourself, is kept, and the removal is not forced, so Docker refuses while any other container, running or stopped, still uses it. A refusal never fails the update.

Containers on [remote agents](/guide/agent) are cleaned up the same way, and so is Dozzle's own container: the [self-update](/guide/setup-wizard#self-update) helper removes the image before the previous one once the new Dozzle has stayed up. Swarm services, including Dozzle running as one, are not cleaned up, since each node keeps its own images and Swarm prunes its own task history.

## Rolling back {#rolling-back}

Rolling an update back is a [Dozzle Cloud](/guide/dozzle-cloud) feature. Dozzle Cloud watches each update the schedule makes and, when the new version starts failing, offers to roll it back. Dozzle then swaps the container back to the image it ran before, the one its `dev.dozzle.previous-image` label names, the same way an update swaps it: the current container is kept until the previous image has stayed up, and is put back if it does not. Settings and volumes stay as they are. Nothing is pulled, so if the previous image is no longer on the host the rollback fails and the container is left alone.

The rolled back container is labelled `dev.dozzle.rolled-back-from` with the image it left, and the auto-update schedule leaves it alone until its tag points to a newer image. Once the rollback has stayed up, the image rolled back from is [cleaned up](#cleaning-up-old-images) like any old image, so it is removed only when no tag points to it any more.

Rollback works for standalone containers, including containers on [remote agents](/guide/agent). It is not available for Swarm services, Kubernetes, or Dozzle's own container.

## Updates in the log view

When Dozzle updates a container, on the schedule, from the Dozzle UI or from Dozzle Cloud, the logs of the new container start with a marker that names the image it moved from and to and what started the update. A rollback and an update that was undone are marked too. With Dozzle Cloud linked, the marker also shows what Dozzle Cloud made of the update, with a link to it there. Dozzle keeps its recent updates in memory, so the marker is gone after Dozzle restarts.
