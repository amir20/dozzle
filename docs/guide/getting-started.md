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
docker run -d -v /var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle:latest
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
    environment:
      # Uncomment to enable container actions (stop, start, restart). See https://dozzle.dev/guide/actions
      # - DOZZLE_ENABLE_ACTIONS=true
      #
      # Uncomment to allow access to container shells. See https://dozzle.dev/guide/shell
      # - DOZZLE_ENABLE_SHELL=true
      #
      # Uncomment to enable authentication. See https://dozzle.dev/guide/authentication
      # - DOZZLE_AUTH_PROVIDER=simple
```

:::

> [!TIP]
> Dozzle supports actions, such as stopping, starting, and restarting containers, or attaching to container shells. But they are disabled by default for security reasons. To enable them, uncomment the corresponding environment variables.
> Dozzle also supports connecting to remote agents to monitor multiple Docker hosts. See [agent](/guide/agent) to learn more.

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

Dozzle supports running in Kubernetes. It only needs to be deployed on one node within the cluster.

<details>
<summary>Kubernetes Configuration</summary>

```yaml [k8s-dozzle.yml]
# rbac.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: pod-viewer
---
# clusterrole.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: pod-viewer-role
rules:
  - apiGroups: [""]
    resources: ["pods", "pods/log", "nodes"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["metrics.k8s.io"]
    resources: ["pods"]
    verbs: ["get", "list"]
---
# clusterrolebinding.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: pod-viewer-binding
subjects:
  - kind: ServiceAccount
    name: pod-viewer
    namespace: default
roleRef:
  kind: ClusterRole
  name: pod-viewer-role
  apiGroup: rbac.authorization.k8s.io
---
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dozzle
spec:
  replicas: 1
  selector:
    matchLabels:
      app: dozzle
  template:
    metadata:
      labels:
        app: dozzle
    spec:
      serviceAccountName: pod-viewer
      containers:
        - name: dozzle
          image: amir20/dozzle:latest
          ports:
            - containerPort: 8080
          env:
            - name: DOZZLE_MODE
              value: "k8s"
```

</details>

Apply the configuration using the following command:

```sh
kubectl apply -f k8s-dozzle.yml
```

See [Kubernetes mode](/guide/k8s) for more information.
