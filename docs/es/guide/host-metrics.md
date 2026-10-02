---
title: Métricas del host
sourceHash: e10c0661efbb
---

# Métricas del host

La tarjeta del host puede mostrar tres lecturas de la máquina en la que corre Docker. Están en un pequeño recuadro a la derecha de la cabecera de la tarjeta, marcado con el icono del host, para que no se confundan con los totales de los contenedores en los medidores de CPU y memoria de abajo:

- **Tiempo activo**, cuánto tiempo lleva encendido el host
- **Carga**, la media de carga de 1 minuto (pasa el ratón por encima para ver las de 5 y 15 minutos)
- **Disco**, lo lleno que está el sistema de archivos que contiene el directorio de datos de Docker, como una pequeña barra que se vuelve amarilla por encima del 70% y roja por encima del 90% (pasa el ratón por encima para ver el espacio usado y el total)

Se actualizan cada 15 segundos mientras haya una pestaña abierta. Cada una solo aparece cuando Dozzle puede leer un valor real, así que en una instalación por defecto el recuadro puede no aparecer o mostrar solo algunas de ellas.

## Ejecutar Dozzle en un contenedor

Dentro de un contenedor, `/proc` describe el contenedor y no el host. Dozzle no va a hacer pasar las cifras del contenedor por las del host, así que la carga y el tiempo activo permanecen ocultos hasta que montes el `/proc` del host en `/host/proc`.

El disco se lee del directorio de datos de Docker (`docker info --format '{{.DockerRootDir}}'`, normalmente `/var/lib/docker`). Móntalo en la misma ruta para obtener la lectura del disco.

::: code-group

```sh
docker run -d \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /proc:/host/proc:ro \
  -v /var/lib/docker:/var/lib/docker:ro \
  -p 8080:8080 amir20/dozzle:latest
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /proc:/host/proc:ro
      - /var/lib/docker:/var/lib/docker:ro
    ports:
      - 8080:8080
```

:::

Los dos montajes son opcionales e independientes. Montar el directorio de datos da a Dozzle acceso de lectura al sistema de archivos de todos los contenedores, así que omítelo si solo quieres la carga y el tiempo activo.

## Ejecutar Dozzle de forma nativa

Un binario de Dozzle que se ejecuta directamente en el host lee `/proc` tal cual, así que la carga y el tiempo activo no necesitan configuración. El disco funciona siempre que el usuario con el que se ejecuta Dozzle pueda ver el directorio de datos.

## Limitaciones

- Por ahora solo el host local informa de métricas. Los hosts conectados mediante un [agente](/es/guide/agent) o como [host remoto](/es/guide/remote-hosts) muestran la CPU y la memoria como antes, sin esta línea.
- Si `DOCKER_HOST` apunta a otra máquina (`tcp://` o `ssh://`), Dozzle omite estas lecturas, ya que su propio `/proc` y sus discos no dicen nada sobre ese motor.
- Docker Desktop ejecuta el motor en una máquina virtual. Un binario nativo de Dozzle en macOS o Windows no tiene `/proc` que leer, y Dozzle en un contenedor allí informa de las cifras de la máquina virtual, no de las de tu equipo.
