---
title: Container-Namen
sourceHash: 31d0ec398f6d
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

## Coolify-Integration

Wenn du [Coolify](https://coolify.io/) verwendest, erkennt Dozzle die Labels von Coolify automatisch als Rückfallwerte:

- `coolify.serviceName` → wird als Container-Name genutzt, wenn `dev.dozzle.name` nicht gesetzt ist
- `coolify.projectName` → wird zur Gruppierung genutzt, wenn `dev.dozzle.group` nicht gesetzt ist

Für Coolify-Deployments ist keine weitere Konfiguration nötig.
