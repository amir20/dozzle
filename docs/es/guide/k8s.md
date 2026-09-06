---
title: Compatibilidad con Kubernetes
sourceHash: dd54f394f340
---

# Compatibilidad con Kubernetes

Dozzle funciona con Kubernetes y te permite ver los logs de tus pods.

## <Icon icon="mdi:kubernetes" inline /> Configuración de Kubernetes

Para montar Dozzle en Kubernetes puedes usar la siguiente configuración YAML con `DOZZLE_MODE=k8s`. Incluye un deployment y un service para exponer Dozzle.

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
# Con ReadWriteOnce y la estrategia Recreate, la configuración de cloud y las
# reglas de notificación dejan de estar disponibles un momento durante los
# despliegues (el pod nuevo no puede montar hasta que el viejo suelta el
# volumen). Para que la configuración persista sin cortes, usa una storage
# class ReadWriteMany (NFS, CephFS, etc.) y cambia la estrategia de abajo a
# RollingUpdate.
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

Esta configuración crea una cuenta de servicio, un cluster role y un cluster role binding para que Dozzle pueda acceder a los recursos de Kubernetes que necesita. También crea un deployment para Dozzle y lo expone mediante un service.

> [!WARNING]
> Si despliegas esto con una herramienta de GitOps (como Flux CD o Argo CD) en un namespace distinto de `default`, acuérdate de cambiar el **namespace** en el **Subject del ClusterRoleBinding**

El resto de funciones también están disponibles, incluidas la autenticación, los filtros y demás. Puedes usar las mismas variables de entorno que en Docker para configurar Dozzle en Kubernetes.

> [!NOTE]
> Dozzle en Kubernetes es una función nueva y puede tener algunas limitaciones frente a la versión de Docker. Usa esta [discusión](https://github.com/amir20/dozzle/discussions/3614) para informar de problemas o proponer mejoras.

## <Icon icon="mdi:chart-line" inline /> API de métricas

Dozzle se apoya en la [API de métricas de Kubernetes](https://github.com/kubernetes-sigs/metrics-server) para obtener el uso de recursos. La API se instala con este comando:

```bash
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
```

Para comprobar que la API está funcionando, ejecuta:

```bash
kubectl top pod
```

Por ahora esto es obligatorio para usar Dozzle en Kubernetes.

## <Icon icon="mdi:filter-variant" inline /> Namespaces y filtros

### Namespaces

Por defecto, Dozzle monitoriza todos los namespaces del clúster. Si quieres limitarlo a uno concreto, define la variable de entorno `DOZZLE_NAMESPACE` con el nombre del namespace.

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
> Dozzle admite varios namespaces: puedes poner en `DOZZLE_NAMESPACE` una lista separada por comas. Cuando se indican varios, Dozzle monitoriza cada namespace por separado y combina los resultados.

### Etiquetas y filtros

`DOZZLE_FILTER` funciona igual que los filtros de Docker. Puedes acotar el alcance de Dozzle con la variable de entorno `DOZZLE_FILTER`. Por ejemplo, para limitarlo a `env=prod`:

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
