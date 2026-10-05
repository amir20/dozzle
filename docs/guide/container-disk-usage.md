---
title: Container Disk Usage
---

# Container Disk Usage

The **Disk** column in the container list shows how much each container has written to its own filesystem, for running and stopped containers alike. It is the same number as the first value in the `SIZE` column of `docker ps -s`, shown in binary units, so Docker's `20.5kB` reads as `20 KB` here.

## What it counts

Docker calls this the container's writable layer: every file a container creates or changes outside its mounts. A container that writes temp files, caches or logs inside its own filesystem grows here, and so does one that runs `apt install` after it starts.

It does not count:

- **Volumes**, named or anonymous
- **Bind mounts**, such as `./data:/var/lib/postgresql/data`
- **The image** the container runs, which every container from that image shares
- **Docker's own log file** for the container

Most databases and log stores keep their data in a volume or a bind mount, so a Postgres holding 30 GB can show a few KB here. That is correct: the 30 GB sits in the mount, not in the container.

## Finding the rest

Docker can measure volumes, but it knows nothing about what is inside a host folder you bind mount. Measuring one means walking every file in it, which Dozzle does not do. To see where the rest of the space goes, run these on the host:

```sh
# volumes, with how many containers use each
docker system df -v

# a bind-mounted folder
du -sh /data/postgres
```

To watch the disk those folders live on, mount it on the host card as described in [Host Metrics](/guide/host-metrics#more-drives).

## When it updates

Docker does not keep this number anywhere. It works it out by walking the layer each time it is asked, so Dozzle asks rarely:

- once for every container, shortly after Dozzle starts
- once more when a container stops, since its layer cannot change after that
- for a running container, after it has written about 100 MB, or every 5 minutes if it has written anything at all

The running checks only happen while someone has Dozzle open. A container shows `–` until its first measurement.

## Limits

- Kubernetes has no writable layer to report, so the column is hidden in [k8s mode](/guide/k8s).
- An [agent](/guide/agent) measures its own containers. One older than the Dozzle it reports to sends no size, and its containers show `–`.
