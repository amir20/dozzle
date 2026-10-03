---
title: Métricas del host
sourceHash: 5d6314385b49
---

# Métricas del host

La tarjeta del host puede mostrar tres lecturas de la máquina en la que corre Docker. Están en un pequeño recuadro a la derecha de la cabecera de la tarjeta, marcado con un icono de pulso, para que no se confundan con los totales de los contenedores en los medidores de CPU y memoria de abajo. En el móvil, el disco pasa a ser un tercer medidor junto a CPU y memoria, y el tiempo activo y la carga bajan a una línea debajo de los medidores:

- **Tiempo activo**, cuánto tiempo lleva encendido el host
- **Carga**, la media de carga de 1 minuto, que se vuelve amarilla cuando supera el número de núcleos y roja por encima del doble (pasa el ratón por encima para ver las de 5 y 15 minutos y el número de núcleos)
- **Disco**, lo lleno que está el sistema de archivos que contiene el directorio de datos de Docker, como una pequeña barra que se vuelve amarilla por encima del 70% y roja por encima del 90% (pasa el ratón por encima para ver el espacio usado y el total)

Se actualizan cada 15 segundos mientras haya una pestaña abierta. Cada una solo aparece cuando Dozzle puede leer un valor real, así que en una instalación por defecto el recuadro puede no aparecer o mostrar solo algunas de ellas.

## Ejecutar Dozzle en un contenedor

Dentro de un contenedor, `/proc` describe el contenedor y no el host. Dozzle no va a hacer pasar las cifras del contenedor por las del host, así que la carga y el tiempo activo permanecen ocultos hasta que montes el `/proc` del host en `/host/proc`.

::: code-group

```sh
docker run -d \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /proc:/host/proc:ro \
  -v dozzle_data:/data \
  -p 8080:8080 amir20/dozzle:latest
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /proc:/host/proc:ro
      - dozzle_data:/data
    ports:
      - 8080:8080
volumes:
  dozzle_data:
```

:::

El disco no necesita ningún montaje adicional. Dozzle mide el sistema de archivos que hay detrás de su propio `/data`, que está en el disco de Docker tanto si `/data` es un volumen con nombre, como arriba, como si no está montado. Si en su lugar montas ahí una carpeta del host (`./data:/data`), la lectura describe el disco en el que está esa carpeta, que normalmente es el mismo.

## Ejecutar Dozzle de forma nativa

Un binario de Dozzle que se ejecuta directamente en el host lee `/proc` tal cual, así que la carga y el tiempo activo no necesitan configuración. El disco se lee del directorio de datos de Docker (`docker info | grep "Docker Root Dir"`, normalmente `/var/lib/docker`), así que funciona siempre que el usuario con el que se ejecuta Dozzle pueda ver ese directorio.

## Agentes

Cada [agente](/es/guide/agent) lee su propia máquina y envía los valores al Dozzle que estás mirando, así que la tarjeta de cada agente muestra su propio tiempo activo, carga y disco. Dale al contenedor del agente los mismos montajes que le darías a Dozzle:

```yaml [docker-compose.yml]
services:
  dozzle-agent:
    image: amir20/dozzle:latest
    command: agent
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /proc:/host/proc:ro
    ports:
      - 7007:7007
```

Las unidades adicionales funcionan igual, montadas en `/host/disks` en el agente. En [modo Swarm](/es/guide/swarm-mode) cada nodo ejecuta Dozzle como su propio agente, así que añade los montajes al servicio y cada nodo informa de sí mismo.

Un agente más antiguo que el Dozzle al que informa no envía métricas, y su tarjeta se queda como estaba. Actualiza el agente para verlas.

## Más unidades

El disco cubre el propio disco de Docker sin configuración adicional. Para vigilar también otras unidades, monta cada una en `/host/disks/<name>`. El nombre de la carpeta se convierte en la etiqueta de la unidad.

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /proc:/host/proc:ro
      - /mnt/media:/host/disks/media:ro
      - /mnt/backup:/host/disks/backup:ro
```

La barra muestra entonces la unidad más llena, ya que es la que se queda sin espacio primero, y al pasar el ratón por encima se listan todas las unidades con su espacio usado y total.

Dozzle solo necesita el punto de montaje para medir una unidad, no sus archivos. Montar la raíz de una unidad sigue permitiendo a Dozzle leer lo que hay en ella, así que si eso te importa, crea una carpeta vacía en la unidad y monta esa en su lugar (`/mnt/media/.dozzle:/host/disks/media:ro`). Está en el mismo sistema de archivos y da las mismas cifras.

Una instalación nativa puede hacer lo mismo con enlaces simbólicos: `ln -s /mnt/media /host/disks/media`.

## Limitaciones

- Un [host remoto](/es/guide/remote-hosts) conectado por TCP no tiene nada en su lado que lea la máquina, así que muestra la CPU y la memoria como antes, sin el recuadro. Ejecuta allí un agente para obtenerlas.
- Si `DOCKER_HOST` apunta a otra máquina (`tcp://` o `ssh://`), Dozzle omite estas lecturas, ya que su propio `/proc` y sus discos no dicen nada sobre ese motor.
- Docker Desktop ejecuta el motor en una máquina virtual. Un binario nativo de Dozzle en macOS o Windows no tiene `/proc` que leer, y Dozzle en un contenedor allí informa de las cifras de la máquina virtual, no de las de tu equipo.
