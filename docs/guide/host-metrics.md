---
title: Host Metrics
---

# Host Metrics

The host card can show three read-outs for the machine Docker runs on. They sit in a small box on the right of the card's header, marked with the host's icon, so they are not mistaken for the container totals in the CPU and memory meters below:

- **Uptime**, how long the host has been up
- **Load**, the 1 minute load average (hover it for the 5 and 15 minute ones)
- **Disk**, how full the filesystem holding Docker's data directory is, as a small bar that turns yellow past 70% and red past 90% (hover it for used and total)

They refresh every 15 seconds while a tab is open. Each one only shows up when Dozzle can read a real value for it, so on a default install the box may be missing or carry only some of them.

## Running Dozzle in a container

Inside a container, `/proc` describes the container and not the host. Dozzle will not pass the container's numbers off as the host's, so load and uptime stay hidden until you mount the host's `/proc` at `/host/proc`.

::: code-group

```sh
docker run -d \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /proc:/host/proc:ro \
  -v dozzle_data:/data \
  -p 8080:8080 amir20/dozzle:latest
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /proc:/host/proc:ro
      - dozzle_data:/data
    ports:
      - 8080:8080
volumes:
  dozzle_data:
```

:::

Disk needs no extra mount. Dozzle measures the filesystem behind its own `/data`, which sits on Docker's disk whether `/data` is a named volume, as above, or not mounted at all. If you bind mount a host folder there instead (`./data:/data`), the read-out describes the disk that folder lives on, which is usually the same one.

## Running Dozzle natively

A Dozzle binary running directly on the host reads `/proc` as is, so load and uptime need no setup. Disk is read from Docker's data directory (`docker info --format '{{.DockerRootDir}}'`, usually `/var/lib/docker`), so it works as long as the user Dozzle runs as can see that directory.

## Limits

- Only the local host reports metrics for now. Hosts connected through an [agent](/guide/agent) or as a [remote host](/guide/remote-hosts) show CPU and memory as before, without this line.
- If `DOCKER_HOST` points at another machine (`tcp://` or `ssh://`), Dozzle skips the read-outs, since its own `/proc` and disks say nothing about that engine.
- Docker Desktop runs the engine in a VM. A native Dozzle binary on macOS or Windows has no `/proc` to read, and Dozzle in a container there reports the VM's numbers, not your machine's.
