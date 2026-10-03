---
title: Host Metrics
---

# Host Metrics

The host card can show three read-outs for the machine Docker runs on. They sit in a small box on the right of the card's header, marked with a pulse icon, so they are not mistaken for the container totals in the CPU and memory meters below. On a phone, disk becomes a third meter next to CPU and memory, and uptime and load move to a line under the meters:

- **Uptime**, how long the host has been up
- **Load**, the 1 minute load average, which turns yellow once it passes the number of cores and red past twice that (hover it for the 5 and 15 minute ones and the core count)
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

A Dozzle binary running directly on the host reads `/proc` as is, so load and uptime need no setup. Disk is read from Docker's data directory (`docker info | grep "Docker Root Dir"`, usually `/var/lib/docker`), so it works as long as the user Dozzle runs as can see that directory.

## Agents

Each [agent](/guide/agent) reads its own machine and sends the values to the Dozzle you are looking at, so every agent's card shows its own uptime, load and disk. Give the agent container the same mounts you would give Dozzle:

```yaml [docker-compose.yml]
services:
  dozzle-agent:
    image: amir20/dozzle:latest
    command: agent
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /proc:/host/proc:ro
    ports:
      - 7007:7007
```

Extra drives work the same way, mounted under `/host/disks` on the agent. In [Swarm mode](/guide/swarm-mode) every node runs Dozzle as its own agent, so add the mounts to the service and each node reports itself.

An agent older than the Dozzle it reports to sends no metrics, and its card stays as it was. Update the agent to see them.

## More drives

Disk covers Docker's own disk out of the box. To watch other drives too, mount each one under `/host/disks/<name>`. The folder name becomes the drive's label.

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /proc:/host/proc:ro
      - /mnt/media:/host/disks/media:ro
      - /mnt/backup:/host/disks/backup:ro
```

The bar then shows the fullest drive, since that is the one that runs out first, and hovering it lists every drive with its used and total.

Dozzle only needs the mount point to measure a drive, not its files. Mounting a drive's root still lets Dozzle read what is on it, so if that matters, create an empty folder on the drive and mount that instead (`/mnt/media/.dozzle:/host/disks/media:ro`). It sits on the same filesystem and reports the same numbers.

A native install can do the same with symlinks: `ln -s /mnt/media /host/disks/media`.

## Limits

- A [remote host](/guide/remote-hosts) connected over TCP has nothing on its side to read the machine, so it shows CPU and memory as before, without the box. Run an agent there instead to get them.
- If `DOCKER_HOST` points at another machine (`tcp://` or `ssh://`), Dozzle skips the read-outs, since its own `/proc` and disks say nothing about that engine.
- Docker Desktop runs the engine in a VM. A native Dozzle binary on macOS or Windows has no `/proc` to read, and Dozzle in a container there reports the VM's numbers, not your machine's.
