---
title: Getting Started
---

# Getting Started

Dozzle runs as a single container. Pick Docker CLI, Docker Compose, Swarm, or Kubernetes below.

## <Icon icon="mdi:docker" inline /> Standalone Docker

Mount `docker.sock` so Dozzle can read containers, mount a volume at `/data` so settings survive a restart, and publish port 8080.

::: code-group

```sh [docker run]
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

Open `http://localhost:8080` and you are done. Everything else, including actions, shell access, authentication, and remote agents, is optional and off by default. The commented environment variables in the Compose file link to each guide.

> [!WARNING]
> Mounting `docker.sock` gives Dozzle root-equivalent access to the host. If you plan to expose Dozzle beyond your private network, read [Security Considerations](/guide/authentication#security-considerations) first.

Dozzle needs Docker Engine 19.03 or newer (API version 1.40+). If Docker Hub is blocked on your network, pull `ghcr.io/amir20/dozzle:latest` from the [GitHub Container Registry](https://ghcr.io/amir20/dozzle:latest) instead.

## <Icon icon="mdi:hexagon-multiple-outline" inline /> Docker Swarm

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

## <Icon icon="mdi:kubernetes" inline /> K8s

Dozzle supports running in Kubernetes. It only needs to be deployed on one node within the cluster. You'll need to set `DOZZLE_MODE=k8s` and configure RBAC for pod log access.

See [Kubernetes mode](/guide/k8s) for the full setup configuration including RBAC, deployment, and service manifests.
