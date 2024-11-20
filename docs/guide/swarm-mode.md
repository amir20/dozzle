---
title: Swarm Mode
---

# Swarm Mode

Dozzle supports Docker Swarm Mode starting from version 8. When using Swarm Mode, Dozzle will automatically discover services and custom groups. Dozzle does not use Swarm API internally as it is [limited](https://github.com/moby/moby/issues/33183). Instead, Dozzle implements its own grouping using swarm labels. Additionally, Dozzle merges stats for containers in a group. This means that you can see logs and stats for all containers in a group in one view. However, it does mean each host needs to be set up with Dozzle.

## How Does It Work?

When deployed in Swarm Mode, Dozzle will create a secured mesh network between all the nodes in the swarm. This network is used to communicate between the different Dozzle instances. The mesh network is created using [mTLS](https://www.cloudflare.com/learning/access-management/what-is-mutual-tls) with a private TLS certificate. This means that all communication between the different Dozzle instances is encrypted and safe to deploy anywhere.

Dozzle supports Docker [stacks](https://docs.docker.com/reference/cli/docker/stack/deploy/), [services](https://docs.docker.com/engine/swarm/how-swarm-mode-works/services/) and custom groups for joining logs together. `com.docker.stack.namespace` and `com.docker.compose.project` labels are used for grouping containers. For services, Dozzle uses the service name as the group name which is `com.docker.swarm.service.name`.

## How to Enable Swarm Mode?

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

> [!NOTE]
> Due to implementation details, <strike>the name for the service must be exactly `dozzle`</strike>. This is no longer required starting with version `v8.2`. You can name the service anything you want. The service name is automatically detected by Dozzle using `com.docker.swarm.service.name` label.

## Setting Up Simple Authentication in Swarm Mode

To set up simple authentication, you can use Docker secrets to store `users.yml` file. Here is an example using Docker Stack:

```yml
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      - DOZZLE_LEVEL=debug
      - DOZZLE_MODE=swarm
      - DOZZLE_AUTH_PROVIDER=simple
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    secrets:
      - source: users
        target: /data/users.yml

    ports:
      - "8080:8080"
    networks:
      - dozzle
    deploy:
      mode: global

networks:
  dozzle:
    driver: overlay
secrets:
  users:
    file: users.yml
```

In this example, `users.yml` file is stored in a Docker secret. It is the same as the [simple authentication](/guide/authentication#generating-users-yml) example.

## Adding Standalone Agents to Swarm Mode

From version v8.8.x, Dozzle supports adding standalone [Agents](/guide/agent) when running in Swarm Mode.

Simply [add the remote agent](/guide/agent#how-to-connect-to-an-agent) to your Swarm compose in the same way you normally would.

> [!NOTE]
> While remote agents are supported, remote connections such as socket proxy are not supported.

```yml
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      - DOZZLE_MODE=swarm
      - DOZZLE_REMOTE_AGENT=agent:7007
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

The remote agent(s) will now display alongside the other nodes in Dozzle.
