---
title: Kubernetes Support
---

# Kubernetes Support <Badge type="warning" text="beta" /> <Badge type="tip" text="v8.11.x" />

Dozzle now supports Kubernetes, allowing you to view logs from your Kubernetes pods. This feature is available in `v8.11` version of Dozzle.

## Kubernetes Setup

To set up Dozzle in Kubernetes, you can use the following YAML configuration using `DOZZLE_MODE=k8s`. This configuration includes a deployment and a service to expose Dozzle.

```yaml
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
---
# service.yaml
apiVersion: v1
kind: Service
metadata:
  name: dozzle-service
spec:
  type: ClusterIP
  selector:
    app: dozzle
  ports:
    - port: 8080
      targetPort: 8080
      protocol: TCP
```

This configuration creates a service account, a cluster role, and a cluster role binding to allow Dozzle to access the necessary Kubernetes resources. It also creates a deployment for Dozzle and exposes it via a service.

All other features are supported as well, including authentication, filtering, and more. You can use the same environment variables as you would in Docker to configure Dozzle in Kubernetes.

> [!NOTE]
> Dozzle in Kubernetes is a new feature and may have some limitations compared to the Docker version. Please use this [discussion](https://github.com/amir20/dozzle/discussions/3614) to report any issues or suggestions for improvement.
