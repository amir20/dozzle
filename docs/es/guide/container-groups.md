---
title: Grupos de contenedores
sourceHash: 87c26dbd0b16
---

# Grupos de contenedores

Dozzle agrupa automáticamente los contenedores según su nombre de stack o de servicio. También puedes crear grupos propios mediante etiquetas.

## Grupos por defecto

En modo host, los contenedores se agrupan por defecto según su nombre de stack. Si existe la etiqueta `com.docker.swarm.service.name`, Dozzle activa automáticamente un «modo Swarm» en el que todos los contenedores con el mismo nombre de servicio se unen.

## Grupos personalizados

Además, puedes crear grupos personalizados añadiendo una etiqueta a tu contenedor. La etiqueta es `dev.dozzle.group` y su valor es el nombre del grupo. Todos los contenedores con el mismo nombre de grupo aparecerán juntos en la interfaz. Por ejemplo, con un grupo llamado `myapp`, todos los contenedores con la etiqueta `dev.dozzle.group=myapp` se agruparán.

Aquí tienes un ejemplo con Docker Compose o la CLI de Docker:

::: code-group

```sh
docker run --label dev.dozzle.group=myapp hello-world
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: hello-world
    labels:
      - dev.dozzle.group=myapp
```

:::
