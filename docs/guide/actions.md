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
- Private registries, since Dozzle has no credentials of its own
- Kubernetes, where image rollout belongs to the cluster

### Updating Dozzle itself

Dozzle cannot stop itself to update in place, so a standalone Dozzle container shows the update notice with a link to the release notes instead of an `Update` button. Running Dozzle as a Swarm service works normally, since the update is handed to the orchestrator. Dozzle agents on other hosts are ordinary containers and update like anything else.
