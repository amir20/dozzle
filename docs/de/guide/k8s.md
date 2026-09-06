---
title: Kubernetes-Unterstützung
sourceHash: dd54f394f340
---

# Kubernetes-Unterstützung

Dozzle unterstützt Kubernetes, sodass du die Logs deiner Kubernetes-Pods ansehen kannst.

## <Icon icon="mdi:kubernetes" inline /> Kubernetes einrichten

Für die Einrichtung von Dozzle in Kubernetes kannst du die folgende YAML-Konfiguration mit `DOZZLE_MODE=k8s` verwenden. Sie enthält ein Deployment und einen Service, um Dozzle bereitzustellen.

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
# ReadWriteOnce zusammen mit der Recreate-Strategie führt dazu, dass Cloud-Konfiguration
# und Benachrichtigungsregeln während eines Pod-Rollouts kurz nicht verfügbar sind (der
# neue Pod kann erst mounten, wenn der alte freigibt). Für Konfigurationspersistenz ohne
# Ausfall nimmst du eine ReadWriteMany-Storage-Class (NFS, CephFS usw.) und stellst die
# Strategie unten auf RollingUpdate um.
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

Diese Konfiguration legt ein Service Account, eine Cluster Role und ein Cluster Role Binding an, damit Dozzle auf die nötigen Kubernetes-Ressourcen zugreifen kann. Außerdem erzeugt sie ein Deployment für Dozzle und stellt es über einen Service bereit.

> [!WARNING]
> Wenn du das mit einem GitOps-Werkzeug (etwa Flux CD oder Argo CD) in einem anderen Namespace als `default` ausrollst, denk daran, den **Namespace** im **Subject des ClusterRoleBinding** anzupassen.

Alle übrigen Funktionen werden ebenfalls unterstützt, darunter Authentifizierung, Filter und mehr. Du kannst dieselben Umgebungsvariablen wie unter Docker verwenden, um Dozzle in Kubernetes zu konfigurieren.

> [!NOTE]
> Dozzle in Kubernetes ist eine neue Funktion und hat gegenüber der Docker-Variante möglicherweise noch Einschränkungen. Nutze bitte diese [Diskussion](https://github.com/amir20/dozzle/discussions/3614), um Probleme oder Verbesserungsvorschläge zu melden.

## <Icon icon="mdi:chart-line" inline /> Metrics-API

Dozzle nutzt die [Kubernetes Metrics API](https://github.com/kubernetes-sigs/metrics-server), um Informationen zur Ressourcennutzung abzurufen. Die API lässt sich mit folgendem Befehl installieren:

```bash
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
```

Ob die API läuft, prüfst du mit diesem Befehl:

```bash
kubectl top pod
```

Aktuell ist das Voraussetzung für den Einsatz von Dozzle in Kubernetes.

## <Icon icon="mdi:filter-variant" inline /> Namespaces und Filter

### Namespaces

Standardmäßig überwacht Dozzle alle Namespaces im Cluster. Willst du Dozzle auf einen bestimmten Namespace beschränken, setze die Umgebungsvariable `DOZZLE_NAMESPACE` auf dessen Namen.

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
> Dozzle unterstützt mehrere Namespaces, du kannst die Umgebungsvariable `DOZZLE_NAMESPACE` auf eine kommagetrennte Liste von Namespaces setzen. Sind mehrere Namespaces angegeben, überwacht Dozzle jeden einzeln und führt die Ergebnisse zusammen.

### Labels und Filter

`DOZZLE_FILTER` verhält sich ähnlich wie Docker-Filter. Mit der Umgebungsvariable `DOZZLE_FILTER` kannst du den Umfang von Dozzle einschränken. Zum Beispiel, um nur auf `env=prod` einzugrenzen:

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
