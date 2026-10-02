---
title: Container-Namen
sourceHash: 67aa41179aae
---

# Container-Namen

Standardmäßig übernimmt Dozzle die Container-Namen direkt von Docker. Das reicht meistens aus, da sich diese Namen mit der Option `--name` in `docker run` oder über das Feld `container_name` in Docker-Compose-Services anpassen lassen.

## Eigene Namen

Wenn sich der Container-Name selbst nicht ändern lässt, kannst du ihn mit dem Label `dev.dozzle.name` am Container überschreiben.

Hier ein Beispiel mit Docker Compose oder der Docker CLI:

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

Im Kubernetes-Modus heißen Container standardmäßig `<pod>/<container>`. Setze `dev.dozzle.name` am Pod-Template, um den Namen zu überschreiben. Eine Annotation eignet sich dafür besser, da Label-Werte keine Leerzeichen enthalten dürfen und auf 63 Zeichen begrenzt sind. Sind beide gesetzt, gewinnt die Annotation.

```yaml [deployment.yaml]
spec:
  template:
    metadata:
      annotations:
        dev.dozzle.name: Public API
```

Der Name gilt für den ganzen Pod. In einem Pod mit mehreren Containern (Init-Container und Sidecars eingeschlossen) behält deshalb jeder seinen eigenen Namen als Suffix, zum Beispiel `Public API/proxy`. Alle Replikas desselben Deployments bekommen denselben Namen.

## Coolify-Integration

Wenn du [Coolify](https://coolify.io/) verwendest, erkennt Dozzle die Labels von Coolify automatisch als Rückfallwerte:

- `coolify.serviceName` → wird als Container-Name genutzt, wenn `dev.dozzle.name` nicht gesetzt ist
- `coolify.projectName` → wird zur Gruppierung genutzt, wenn `dev.dozzle.group` nicht gesetzt ist

Für Coolify-Deployments ist keine weitere Konfiguration nötig.
