---
title: Uso de disco por contenedor
sourceHash: db32da6a38bf
---

# Uso de disco por contenedor

La columna **Disco** de la lista de contenedores muestra cuánto ha escrito cada contenedor en su propio sistema de ficheros, tanto si está en marcha como si está parado. Es el mismo valor que el primero de la columna `SIZE` de `docker ps -s`, mostrado en unidades binarias, así que el `20.5kB` de Docker aparece aquí como `20 KB`.

## Qué cuenta

Docker lo llama la capa escribible del contenedor: cualquier fichero que un contenedor crea o modifica fuera de sus montajes. Un contenedor que escribe ficheros temporales, cachés o logs dentro de su propio sistema de ficheros crece aquí, y también uno que ejecuta `apt install` después de arrancar.

No cuenta:

- **Los volúmenes**, con nombre o anónimos
- **Los bind mounts**, como `./data:/var/lib/postgresql/data`
- **La imagen** que ejecuta el contenedor, que comparten todos los contenedores de esa imagen
- **El fichero de log de Docker** para el contenedor

La mayoría de las bases de datos y almacenes de logs guardan sus datos en un volumen o un bind mount, así que un Postgres con 30 GB puede mostrar aquí solo unos pocos KB. Es correcto: los 30 GB están en el montaje, no en el contenedor.

## Encontrar el resto

Docker puede medir volúmenes, pero no sabe nada del contenido de una carpeta del host montada como bind mount. Medirla significa recorrer cada fichero que contiene, y Dozzle no lo hace. Para ver adónde va el resto del espacio, ejecuta esto en el host:

```sh
# volúmenes, con cuántos contenedores usan cada uno
docker system df -v

# una carpeta montada como bind mount
du -sh /data/postgres
```

Para vigilar el disco en el que están esas carpetas, móntalo en la tarjeta del host como se describe en [Métricas del host](/es/guide/host-metrics#mas-unidades).

## Cuándo se actualiza

Docker no guarda este valor en ningún sitio. Lo calcula recorriendo la capa cada vez que se le pregunta, así que Dozzle pregunta poco:

- una vez por cada contenedor, poco después de que Dozzle arranque
- una vez más cuando un contenedor se para, ya que su capa no puede cambiar después
- en un contenedor en marcha, después de que haya escrito unos 100 MB, o cada 5 minutos si ha escrito algo

Las comprobaciones de contenedores en marcha solo ocurren mientras alguien tiene Dozzle abierto. Un contenedor muestra `–` hasta su primera medición.

## Limitaciones

- Kubernetes no tiene capa escribible que informar, así que la columna se oculta en [modo k8s](/es/guide/k8s).
- Un [agente](/es/guide/agent) mide sus propios contenedores. Uno más antiguo que el Dozzle al que informa no envía tamaño, y sus contenedores muestran `–`.
