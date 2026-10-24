---
title: Kubernetes Support
description: Run Dozzle in Kubernetes to view pod logs, with the RBAC, Metrics API and kubelet setup for stats, plus pod restarts, namespaces and filters.
---

# Kubernetes Support

Dozzle supports Kubernetes, allowing you to view logs from your Kubernetes pods.

## <Icon icon="mdi:package-variant" inline /> Install with Helm

The quickest way to run Dozzle in a cluster is the Helm chart. It creates everything the manual setup below does, and binds the role to whatever namespace you install into.

```bash
helm install dozzle oci://ghcr.io/amir20/charts/dozzle --namespace dozzle --create-namespace
```

The chart's version matches the Dozzle release. Settings go through `env`, and the extra permissions for container actions and network and disk stats are switched on with `rbac.actions` and `rbac.nodeStats`:

```yaml
# values.yaml
env:
  - name: DOZZLE_ENABLE_ACTIONS
    value: "true"
rbac:
  actions: true
  nodeStats: true
ingress:
  enabled: true
  hosts:
    - host: dozzle.example.com
      paths:
        - path: /
          pathType: Prefix
```

See [`values.yaml`](https://github.com/amir20/dozzle/blob/main/charts/dozzle/values.yaml) for every option.

## <Icon icon="mdi:kubernetes" inline /> Kubernetes Setup

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
  - apiGroups: ["apps"]
    resources: ["deployments", "replicasets", "daemonsets", "statefulsets"]
    verbs: ["get"]
  - apiGroups: ["batch"]
    resources: ["jobs", "cronjobs"]
    verbs: ["get"]
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
# pvc.yaml
# ReadWriteOnce + Recreate strategy means cloud config / notification rules
# briefly become unavailable during pod rollouts (the new pod can't mount until
# the old one releases). For zero-downtime config persistence, use a
# ReadWriteMany storage class (NFS, CephFS, etc.) and switch the strategy
# below to RollingUpdate.
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: dozzle-data
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
---
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dozzle
spec:
  selector:
    matchLabels:
      app: dozzle
  strategy:
    type: Recreate
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
          volumeMounts:
            - name: data
              mountPath: /data
      volumes:
        - name: data
          persistentVolumeClaim:
            claimName: dozzle-data
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

> [!WARNING]
> When deploying this with any GitOps tool (like Flux CD or Argo CD) in a specific namespace apart from `default`, make sure to change the **namespace** in the **ClusterRoleBinding Subject**

All other features are supported as well, including authentication, filtering, and more. You can use the same environment variables as you would in Docker to configure Dozzle in Kubernetes.

> [!NOTE]
> Dozzle in Kubernetes is a new feature and may have some limitations compared to the Docker version. Please use this [discussion](https://github.com/amir20/dozzle/discussions/3614) to report any issues or suggestions for improvement.

## <Icon icon="mdi:chart-line" inline /> Metrics API

Dozzle relies on the [Kubernetes Metrics API](https://github.com/kubernetes-sigs/metrics-server) to retrieve resource usage information. The API can be installed using the following command:

```bash
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
```

To verify that the API is running, you can run the following command:

```bash
kubectl top pod
```

Without it, logs still work, and CPU and memory stay empty.

The memory bar measures usage against the container's memory limit. A container without a limit is measured against its node's memory, as Docker does.

## <Icon icon="mdi:harddisk" inline /> Network and Disk

The Metrics API has no network or disk numbers. Dozzle can read them from each node's kubelet (`/stats/summary`), the same place metrics-server reads from. This is opt-in: add this rule to the ClusterRole and the network and disk columns appear.

```yaml
- apiGroups: [""]
  resources: ["nodes/stats"]
  verbs: ["get"]
```

Network is counted per pod, so the pod's traffic shows on the container whose name sorts first and its other containers show none. Disk is the container's writable layer plus the PVCs and `emptyDir` volumes it mounts.

Dozzle calls each kubelet directly on the node's internal IP (port 10250), never through the API server's node proxy. That would need `nodes/proxy`, which allows exec into any container. The kubelet's serving certificate has to be signed by the cluster CA, as it is on k3s. Where kubelets use self-signed certificates (kubeadm's default, kind), the columns stay hidden and Dozzle logs why once.

## <Icon icon="mdi:restart" inline /> Container Actions

With `DOZZLE_ENABLE_ACTIONS=true`, Dozzle can restart a container. Kubernetes has no stop or start for a single container, so restart is the only action. Dozzle restarts by deleting the pod so its controller (a Deployment, StatefulSet, DaemonSet or Job) creates a new one, the same as `kubectl delete pod`. A pod without a controller is never deleted.

Deleting pods needs a permission the read-only role above does not grant. Add this rule to the ClusterRole:

```yaml
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["delete"]
```

### Rollout Restart

Restarting one pod at a time is the wrong tool for a workload with several replicas. A container that belongs to a Deployment, StatefulSet or DaemonSet also gets a **Restart Deployment** (or StatefulSet, or DaemonSet) action in its menu, and the same action sits in the menu of the workload's merged view in the sidebar. It does what `kubectl rollout restart` does: the controller replaces every pod following the workload's own rollout strategy, so `maxUnavailable` is respected. A pod with `imagePullPolicy: Always` pulls its image again on the way, which is how a `:latest` tag picks up a new image.

Jobs, CronJobs and bare ReplicaSets cannot be rolled out, so the action does not appear for them. Like every other action, it only reaches workloads whose pods pass Dozzle's filters.

Rolling out needs permission to patch the workload. Add this rule to the ClusterRole:

```yaml
- apiGroups: ["apps"]
  resources: ["deployments", "statefulsets", "daemonsets"]
  verbs: ["patch"]
```

## <Icon icon="mdi:filter-variant" inline /> Namespaces and Filters

### Namespaces

By default, Dozzle will monitor all namespaces in the cluster. If you want to restrict Dozzle to a specific namespace, you can set the `DOZZLE_NAMESPACE` environment variable to the name of the namespace.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dozzle
spec:
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
            - name: DOZZLE_NAMESPACE
              value: "default"
```

> [!NOTE]
> Dozzle supports multiple namespaces, you can set the `DOZZLE_NAMESPACE` environment variable to a comma-separated list of namespaces. When multiple namespaces are specified, Dozzle will monitor each namespace separately and combine the results.

### Labels and Filters

`DOZZLE_FILTER` behave similarly to Docker filters. You can limit the scope of Dozzle using the `DOZZLE_FILTER` environment variable. For example, to scope only to `env=prod`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dozzle
spec:
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
            - name: DOZZLE_FILTER
              value: "env=prod"
```
