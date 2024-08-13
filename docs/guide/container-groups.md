---
title: Container Groups
---

# Container Groups

Dozzle performs automatic grouping of containers based on their stack name or service name. You can also create custom groups using labels.

## Default Groups

By default, containers are grouped by their stack name in host mode. If `com.docker.swarm.service.name` label is present, Dozzle will automatically enable a "swarm mode" where all containers with the same service name will be joined together.

## Custom Groups

Additionally, you can create custom groups by adding a label to your container. The label is `dev.dozzle.group` and the value is the name of the group. All containers with the same group name will be joined together in the UI. For example, if you have a group named `myapp`, all containers with the label `dozzle.group=myapp` will be joined together.

Here is an example using Docker Compose or Docker CLI:

::: code-group

```sh
docker run --label dev.dozzle.group=myapp hello-world
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: hello-world
    labels:
      - dev.dozzle.group=myapp
```

:::
