---
title: Kubernetes 支持
sourceHash: dd54f394f340
---

# Kubernetes 支持

Dozzle 支持 Kubernetes，可以让你查看 Kubernetes Pod 的日志。

## <Icon icon="mdi:kubernetes" inline /> Kubernetes 配置

要在 Kubernetes 中部署 Dozzle，可以使用下面这份带有 `DOZZLE_MODE=k8s` 的 YAML 配置。它包含一个 Deployment 和一个用来暴露 Dozzle 的 Service。

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
# ReadWriteOnce 加上 Recreate 策略意味着在 Pod 滚动更新期间，云端配置和
# 通知规则会短暂不可用（旧 Pod 释放之前，新 Pod 无法挂载）。如果需要零停机的
# 配置持久化，请使用 ReadWriteMany 存储类（NFS、CephFS 等），并把下面的
# 策略改为 RollingUpdate。
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

这份配置创建了一个 ServiceAccount、一个 ClusterRole 和一个 ClusterRoleBinding，让 Dozzle 能访问所需的 Kubernetes 资源。它还创建了 Dozzle 的 Deployment，并通过 Service 把它暴露出去。

> [!WARNING]
> 如果你用 GitOps 工具（比如 Flux CD 或 Argo CD）把它部署到 `default` 之外的某个命名空间，记得修改 **ClusterRoleBinding Subject** 中的**命名空间**

其他所有功能同样受支持，包括身份验证、过滤等等。在 Kubernetes 中配置 Dozzle 时，可以使用和 Docker 中相同的环境变量。

> [!NOTE]
> Kubernetes 中的 Dozzle 是一个新功能，相比 Docker 版本可能还有一些限制。请通过这个[讨论](https://github.com/amir20/dozzle/discussions/3614)反馈问题或改进建议。

## <Icon icon="mdi:chart-line" inline /> Metrics API

Dozzle 依赖 [Kubernetes Metrics API](https://github.com/kubernetes-sigs/metrics-server) 来获取资源使用信息。可以用下面的命令安装该 API：

```bash
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
```

要确认 API 正在运行，可以执行下面的命令：

```bash
kubectl top pod
```

目前在 Kubernetes 中使用 Dozzle 必须安装它。

## <Icon icon="mdi:filter-variant" inline /> 命名空间与过滤

### 命名空间

默认情况下，Dozzle 会监控集群中的所有命名空间。如果你想把 Dozzle 限制在某个命名空间内，可以把 `DOZZLE_NAMESPACE` 环境变量设为该命名空间的名称。

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
> Dozzle 支持多个命名空间，你可以把 `DOZZLE_NAMESPACE` 环境变量设为用逗号分隔的命名空间列表。指定多个命名空间时，Dozzle 会分别监控每个命名空间，然后把结果合并起来。

### 标签与过滤

`DOZZLE_FILTER` 的行为与 Docker 过滤条件类似。你可以用 `DOZZLE_FILTER` 环境变量限制 Dozzle 的范围。例如，只限定在 `env=prod`：

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
