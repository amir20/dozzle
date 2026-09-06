---
title: Prise en charge de Kubernetes
sourceHash: dd54f394f340
---

# Prise en charge de Kubernetes

Dozzle prend en charge Kubernetes, ce qui vous permet de consulter les logs de vos pods Kubernetes.

## <Icon icon="mdi:kubernetes" inline /> Installation dans Kubernetes

Pour installer Dozzle dans Kubernetes, utilisez la configuration YAML suivante avec `DOZZLE_MODE=k8s`. Cette configuration comprend un déploiement et un service pour exposer Dozzle.

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
# ReadWriteOnce combiné à la stratégie Recreate signifie que la configuration cloud
# et les règles de notification deviennent brièvement indisponibles pendant le
# redéploiement des pods (le nouveau pod ne peut pas monter le volume tant que
# l'ancien ne l'a pas libéré). Pour une persistance sans interruption, utilisez une
# classe de stockage ReadWriteMany (NFS, CephFS, etc.) et passez la stratégie
# ci-dessous à RollingUpdate.
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

Cette configuration crée un compte de service, un cluster role et un cluster role binding pour permettre à Dozzle d'accéder aux ressources Kubernetes nécessaires. Elle crée également un déploiement pour Dozzle et l'expose via un service.

> [!WARNING]
> Si vous déployez ceci avec un outil GitOps (comme Flux CD ou Argo CD) dans un namespace autre que `default`, pensez à changer le **namespace** dans le **Subject du ClusterRoleBinding**

Toutes les autres fonctionnalités sont aussi prises en charge, y compris l'authentification, le filtrage, etc. Vous pouvez utiliser les mêmes variables d'environnement que sous Docker pour configurer Dozzle dans Kubernetes.

> [!NOTE]
> Dozzle dans Kubernetes est une fonctionnalité récente qui peut présenter quelques limites par rapport à la version Docker. Utilisez cette [discussion](https://github.com/amir20/dozzle/discussions/3614) pour signaler des problèmes ou proposer des améliorations.

## <Icon icon="mdi:chart-line" inline /> API Metrics

Dozzle s'appuie sur l'[API Metrics de Kubernetes](https://github.com/kubernetes-sigs/metrics-server) pour récupérer les informations d'utilisation des ressources. Cette API peut être installée avec la commande suivante :

```bash
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
```

Pour vérifier que l'API fonctionne, exécutez la commande suivante :

```bash
kubectl top pod
```

Pour l'instant, elle est requise pour utiliser Dozzle dans Kubernetes.

## <Icon icon="mdi:filter-variant" inline /> Namespaces et filtres

### Namespaces

Par défaut, Dozzle surveille tous les namespaces du cluster. Si vous voulez restreindre Dozzle à un namespace précis, définissez la variable d'environnement `DOZZLE_NAMESPACE` avec le nom du namespace.

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
> Dozzle gère plusieurs namespaces : vous pouvez définir la variable d'environnement `DOZZLE_NAMESPACE` avec une liste de namespaces séparés par des virgules. Quand plusieurs namespaces sont indiqués, Dozzle surveille chacun séparément et combine les résultats.

### Labels et filtres

`DOZZLE_FILTER` se comporte comme les filtres Docker. Vous pouvez limiter la portée de Dozzle avec la variable d'environnement `DOZZLE_FILTER`. Par exemple, pour se limiter à `env=prod` :

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
