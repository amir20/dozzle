---
title: Container Disk Usage
---

# Container Disk Usage

The **Disk** column in the container list shows how much disk each container takes up, for running and stopped containers alike. It adds up two things Docker can measure: what the container wrote to its own filesystem, and the Docker volumes it mounts. Hover the value to see each part.

Sizes are in binary units, so Docker's `20.5kB` reads as `20 KB` here.

## What it counts

- **The writable layer**: every file a container creates or changes outside its mounts, such as temp files, caches, or packages installed after it started. This is the first value in the `SIZE` column of `docker ps -s`.
- **Volumes**, named or anonymous, that the container mounts. These are the sizes `docker system df -v` lists. A volume used by more than one container counts in each of them, and the tooltip says it is shared.

It does not count:

- **Bind mounts**, such as `./data:/var/lib/postgresql/data`
- **The image** the container runs, which every container from that image shares
- **Docker's own log file** for the container
- **Volumes no container uses**, since they have no row to show up on. They count toward the host's [reclaimable space](#reclaimable-space) instead.

## Bind mounts

Docker knows nothing about what is inside a host folder you bind mount, and measuring one means walking every file in it, which Dozzle does not do. A database that keeps its data in a bind mount shows a few KB here even when it holds many GB. To measure one, run this on the host:

```sh
du -sh /data/postgres
```

To watch the disk those folders live on, mount it on the host card as described in [Host Metrics](/guide/host-metrics#more-drives).

## Reclaimable space

The host card shows **Reclaimable** next to its disk meter: space held by things no container is using, the same total as the `RECLAIMABLE` column of `docker system df`. Hover it to see how much sits in unused images, unused volumes, stopped containers and build cache. It refreshes along with the volumes, so at most every 20 minutes.

## When it updates

Docker does not keep these numbers anywhere. It works them out by walking the files each time it is asked, so Dozzle asks rarely.

The writable layer is measured:

- once for every container, shortly after Dozzle starts
- once more when a container stops, since its layer cannot change after that
- for a running container, after it has written about 100 MB, or every 5 minutes if it has written anything at all

Volumes are measured right after that first pass, then at most every 20 minutes. Docker can only measure all volumes at once, so each refresh walks every volume on the host. A container created in between shows its volumes at the next refresh.

Everything after the first pass only happens while someone has Dozzle open. A container shows `–` until its first measurement.

## Limits

- Kubernetes has no writable layer or Docker volumes to report, so the column is hidden in [k8s mode](/guide/k8s).
- Volumes from a plugin driver usually cannot report a size and are left out.
- An [agent](/guide/agent) measures its own containers. One older than the Dozzle it reports to sends no sizes, and its containers show `–`.
