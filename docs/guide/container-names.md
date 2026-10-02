---
title: Container Names
---

# Container Names

By default, Dozzle retrieves container names directly from Docker. This is usually sufficient, as these names can be customized using the `--name` flag in `docker run` commands or through the `container_name` field in Docker Compose services.

## Custom Names

In cases where modifying the container name itself isn't possible, you can override it by adding a `dev.dozzle.name` label to your container.

Here is an example using Docker Compose or Docker CLI:

::: code-group

```sh
docker run --label dev.dozzle.name=hello hello-world
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: hello-world
    labels:
      - dev.dozzle.name=hello
```

:::

## Kubernetes

In Kubernetes mode, containers are named `<pod>/<container>` by default. Set `dev.dozzle.name` on the pod template to override it. An annotation is the better place for it, because label values cannot contain spaces and are limited to 63 characters. If both are set, the annotation wins. The same goes for every other `dev.dozzle.*` setting, such as `dev.dozzle.url` and `dev.dozzle.icon`.

```yaml [deployment.yaml]
spec:
  template:
    metadata:
      annotations:
        dev.dozzle.name: Public API
```

The name applies to the whole pod. Init containers always keep their own name as a suffix, for example `Public API/migrate`, and so does every container in a pod with more than one app container, for example `Public API/proxy`.

Replicas of the same deployment all get the same name, and Dozzle treats containers with the same name as one. Pinning one replica in the sidebar pins all of them, and the container title menu lists the others as earlier runs of the same container.

## Coolify Integration

If you're using [Coolify](https://coolify.io/), Dozzle automatically recognizes Coolify's labels as fallbacks:

- `coolify.serviceName` → Used as container name if `dev.dozzle.name` is not set
- `coolify.projectName` → Used for grouping if `dev.dozzle.group` is not set

No additional configuration is needed for Coolify deployments.
