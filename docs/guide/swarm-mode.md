---
title: Swarm Mode
---

# Introducing Swarm Mode

Dozzle added "Swarm Mode" in version 7 which supports Docker [stacks](https://docs.docker.com/reference/cli/docker/stack/deploy/), [services](https://docs.docker.com/engine/swarm/how-swarm-mode-works/services/) and custom groups for joining logs together. Dozzle does not use Swarm API internally as it is limited. Dozzle implements its own grouping using swarm labels. Additionally, Dozzle merges stats for containers in a group. This means that you can see logs and stats for all containers in a group in one view. But it does mean that each host needs to be setup with Dozzle.

Dozzle swarm mode is automatically enabled when services or customer groups are found. If you are not using services, you can still take advantage of Dozzle's grouping feature by adding a label to your containers.

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

## Merging Logs and Stats

Dozzle merges logs and stats for containers in a group. This means that you can see logs and stats for all containers in a group in one view. This is useful for applications that have multiple containers that work together. Dozzle will automatically find new containers in a group and add them to the view as they are started.

> [!INFO]
> Automatic discovery of new containers is only available for services and custom groups. If you using merging logs in host mode, only specific containers will be shown. You can still use custom groups to merge logs for containers in swarm mode.

## Service Discovery

Dozzle uses Docker API to discover services and custom groups. This means that Dozzle will automatically find new containers in a group and add them to the view as they are started. This is useful for applications that have multiple containers that work together. Labels that are used are `com.docker.stack.namespace` and `com.docker.compose.project` for grouping containers. For services, Dozzle uses the service name as the group name which is `com.docker.swarm.service.name`.
