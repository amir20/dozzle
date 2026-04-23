---
title: Getting Started
---

# Getting Started

Dozzle supports multiple ways to run the application. You can run it using Docker CLI, Docker Compose, Swarm, or Kubernetes. The following sections will guide you through the process of setting up Dozzle.

> [!TIP]
> If Docker Hub is blocked in your network, you can use the [GitHub Container Registry](https://ghcr.io/amir20/dozzle:latest) to pull the image. Use `ghcr.io/amir20/dozzle:latest` instead of `amir20/dozzle:latest`.

## Standalone Docker

The easiest way to set up Dozzle is to use the CLI and mount `docker.sock` file. This file is usually located at `/var/run/docker.sock` and can be mounted with the `--volume` flag. You also need to expose the port to view Dozzle. By default, Dozzle listens on port 8080, but you can change the external port using `-p`. You can also run using compose or as a service in Swarm.

::: code-group

```sh
docker run -d -v /var/run/docker.sock:/var/run/docker.sock -v dozzle_data:/data -p 8080:8080 amir20/dozzle:latest
```

```yaml [docker-compose.yml]
# Run with docker compose up -d
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - dozzle_data:/data
    ports:
      - 8080:8080
    environment:
      # Uncomment to enable container actions (stop, start, restart). See https://dozzle.dev/guide/actions
      # - DOZZLE_ENABLE_ACTIONS=true
      #
      # Uncomment to allow access to container shells. See https://dozzle.dev/guide/shell
      # - DOZZLE_ENABLE_SHELL=true
      #
      # Uncomment to enable authentication. See https://dozzle.dev/guide/authentication
      # - DOZZLE_AUTH_PROVIDER=simple
      #
      # Label this Dozzle instance (shown in the header and multi-host menu). See https://dozzle.dev/guide/hostname
      # - DOZZLE_HOSTNAME=my-server
      #
      # Connect to one or more remote agents to monitor other Docker hosts. See https://dozzle.dev/guide/agent
      # - DOZZLE_REMOTE_AGENT=192.168.1.10:7007,192.168.1.11:7007
      #
      # Only show containers matching a filter. See https://dozzle.dev/guide/filters
      # - DOZZLE_FILTER=label=com.example.app
volumes:
  dozzle_data:
```

:::

> [!TIP]
> Dozzle supports actions, such as stopping, starting, and restarting containers, or attaching to container shells. But they are disabled by default for security reasons. To enable them, uncomment the corresponding environment variables.
> Dozzle also supports connecting to remote agents to monitor multiple Docker hosts. See [agent](/guide/agent) to learn more.

> [!IMPORTANT]
> Dozzle stores notification settings and other data in `/data` inside the container. To persist these settings across container restarts, you need to mount a volume to `/data`. Without this mount, notification settings will be lost when the container is recreated. See the Docker Compose example above for the recommended volume configuration.

> [!WARNING]
> Mounting `docker.sock` gives Dozzle root-equivalent access to the host. If you plan to expose Dozzle beyond your private network, read [Security Considerations](/guide/authentication#security-considerations) first.

## Docker Swarm

Dozzle supports running in Swarm mode by deploying it on every node. To run Dozzle in Swarm mode, you can use the following configuration:

```yaml [dozzle-stack.yml]
# Run with docker stack deploy -c dozzle-stack.yml <name>
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      - DOZZLE_MODE=swarm
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    ports:
      - 8080:8080
    networks:
      - dozzle
    deploy:
      mode: global
networks:
  dozzle:
    driver: overlay
```

Then you can deploy the stack using the following command:

```bash
docker stack deploy -c dozzle-stack.yml <name>
```

See [swarm mode](/guide/swarm-mode) for more information.

## K8s <Badge type="tip" text="New" />

Dozzle supports running in Kubernetes. It only needs to be deployed on one node within the cluster. You'll need to set `DOZZLE_MODE=k8s` and configure RBAC for pod log access.

See [Kubernetes mode](/guide/k8s) for the full setup configuration including RBAC, deployment, and service manifests.
