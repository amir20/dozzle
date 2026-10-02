---
title: Nombres de contenedores
sourceHash: cafa1526d194
---

# Nombres de contenedores

Por defecto, Dozzle obtiene los nombres de los contenedores directamente de Docker. Normalmente es suficiente, ya que esos nombres se pueden personalizar con la opción `--name` de `docker run` o mediante el campo `container_name` de los servicios de Docker Compose.

## Nombres personalizados

Cuando no es posible cambiar el nombre del contenedor en sí, puedes sobrescribirlo añadiendo la etiqueta `dev.dozzle.name` al contenedor.

Aquí tienes un ejemplo con Docker Compose o la CLI de Docker:

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

En el modo Kubernetes, los contenedores se llaman `<pod>/<container>` por defecto. Define `dev.dozzle.name` en la plantilla del pod para cambiarlo. Es mejor usar una anotación, porque los valores de las etiquetas no pueden contener espacios y tienen un límite de 63 caracteres. Si defines ambas, gana la anotación. Lo mismo vale para cualquier otro ajuste `dev.dozzle.*`, como `dev.dozzle.url` y `dev.dozzle.icon`.

```yaml [deployment.yaml]
spec:
  template:
    metadata:
      annotations:
        dev.dozzle.name: Public API
```

El nombre se aplica a todo el pod. Los contenedores de inicialización siempre conservan su propio nombre como sufijo, por ejemplo `Public API/migrate`, y lo mismo ocurre con cada contenedor de un pod con más de un contenedor de aplicación, por ejemplo `Public API/proxy`.

Todas las réplicas de un mismo deployment reciben el mismo nombre, y Dozzle trata los contenedores con el mismo nombre como uno solo. Fijar una réplica en la barra lateral las fija todas, y el menú del título del contenedor muestra las demás como ejecuciones anteriores del mismo contenedor.

## Integración con Coolify

Si usas [Coolify](https://coolify.io/), Dozzle reconoce automáticamente sus etiquetas como valores alternativos:

- `coolify.serviceName` → se usa como nombre del contenedor si `dev.dozzle.name` no está definida
- `coolify.projectName` → se usa para agrupar si `dev.dozzle.group` no está definida

Los despliegues con Coolify no necesitan configuración adicional.
