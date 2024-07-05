---
title: Getting Started
---

# Getting Started

Dozzle supports multiple ways to run the application. You can run it using Docker CLI, Docker Compose, or in Swarm. The following sections will guide you through the process of setting up Dozzle.

## Running with Docker <Badge type="tip" text="Updated" />

The easiest way to setup Dozzle is to use the CLI and mount `docker.sock` file. This file is usually located at `/var/run/docker.sock` and can be mounted with the `--volume` flag. You also need to expose the port to view Dozzle. By default, Dozzle listens on port 8080, but you can change the external port using `-p`. You can also run using compose or as a service in Swarm.

::: code-group

```sh
docker run -d -v /var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle
```

```yaml [docker-compose.yml]
# Run with docker compose up -d
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    ports:
      - 8080:8080
```

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

:::

See [swarm mode](/guide/swarm-mode) for more information on running Dozzle in Swarm.
