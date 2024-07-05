---
title: Swarm Mode
---

# Swarm Mode <Badge text="New" type="tip" />

Dozzle supports Docker Swarm Mode starting from version 8. When using Swarm Mode, Dozzle will automatically discover services and custom groups. Dozzle does not use Swarm API internally as it is [limited](https://github.com/moby/moby/issues/33183). Dozzle implements its own grouping using swarm labels. Additionally, Dozzle merges stats for containers in a group. This means that you can see logs and stats for all containers in a group in one view. But it does mean each host needs to be setup with Dozzle.

## How does it work?

When deployed in Swarm Mode, Dozzle will create a secured mesh network between all the nodes in the swarm. This network is used to communicate between the different Dozzle instances. The mesh network is created using [mTLS](https://www.cloudflare.com/learning/access-management/what-is-mutual-tls) with a private TLS certificate. This means that all communication between the different Dozzle instances is encrypted and safe to deploy any where.

Dozzle supports Docker [stacks](https://docs.docker.com/reference/cli/docker/stack/deploy/), [services](https://docs.docker.com/engine/swarm/how-swarm-mode-works/services/) and custom groups for joining logs together. `com.docker.stack.namespace` and `com.docker.compose.project` labels are used for grouping containers. For services, Dozzle uses the service name as the group name which is `com.docker.swarm.service.name`.

## How to enable Swarm Mode?

To deploy on every node in the swarm, you can use `mode: global`. This will deploy Dozzle on every node in the swarm. Here is an example using Docker Stack:

```yml
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

Note that the `DOZZLE_MODE` environment variable is set to `swarm`. This tells Dozzle to automatically discover other Dozzle instances in the swarm. The `overlay` network is used to create the mesh network between the different Dozzle instances.

## Custom Groups

Custom groups are created by adding a label to your container. The label is `dev.dozzle.group` and the value is the name of the group. All containers with the same group name will be joined together in the UI. For example, if you have a group named `myapp`, all containers with the label `dozzle.group=myapp` will be joined together.

Here is an example using Docker Compose or Docker CLI:

::: code-group

```sh
docker run --label dev.dozzle.group=myapp hello-world
```

```yaml [docker-compose.yml]
version: "3"
services:
  dozzle:
    image: hello-world
    labels:
      - dev.dozzle.group=myapp
```

:::
