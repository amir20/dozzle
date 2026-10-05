---
title: Uso de disco por contenedor
sourceHash: 36536624d20d
---

# Uso de disco por contenedor

La columna **Disco** de la lista de contenedores muestra cuánto disco ocupa cada contenedor, tanto si está en marcha como si está parado. Suma dos cosas que Docker puede medir: lo que el contenedor ha escrito en su propio sistema de ficheros y los volúmenes de Docker que monta. Pasa el ratón por encima del valor para ver cada parte.

Los tamaños están en unidades binarias, así que el `20.5kB` de Docker aparece aquí como `20 KB`.

## Qué cuenta

- **La capa escribible**: cualquier fichero que un contenedor crea o modifica fuera de sus montajes, como ficheros temporales, cachés o paquetes instalados después de arrancar. Es el primer valor de la columna `SIZE` de `docker ps -s`.
- **Los volúmenes**, con nombre o anónimos, que monta el contenedor. Son los tamaños que lista `docker system df -v`. Un volumen que usan varios contenedores cuenta en cada uno de ellos, y el tooltip indica que es compartido.

No cuenta:

- **Los bind mounts**, como `./data:/var/lib/postgresql/data`
- **La imagen** que ejecuta el contenedor, que comparten todos los contenedores de esa imagen
- **El fichero de log de Docker** para el contenedor
- **Los volúmenes que ningún contenedor usa**, ya que no tienen una fila en la que aparecer. En su lugar cuentan en el [espacio recuperable](#espacio-recuperable) del host.

## Bind mounts

Docker no sabe nada del contenido de una carpeta del host montada como bind mount, y medirla significa recorrer cada fichero que contiene, y Dozzle no lo hace. Una base de datos que guarda sus datos en un bind mount muestra aquí unos pocos KB aunque contenga muchos GB. Para medir uno, ejecuta esto en el host:

```sh
du -sh /data/postgres
```

Para vigilar el disco en el que están esas carpetas, móntalo en la tarjeta del host como se describe en [Métricas del host](/es/guide/host-metrics#mas-unidades).

## Espacio recuperable

La tarjeta del host muestra **Recuperable** junto a su indicador de disco: el espacio que ocupan las cosas que ningún contenedor usa, el mismo total que la columna `RECLAIMABLE` de `docker system df`. Pasa el ratón por encima para ver cuánto hay en imágenes sin usar, volúmenes sin usar, contenedores parados y caché de build. Se actualiza junto con los volúmenes, así que como mucho cada 20 minutos.

## Cuándo se actualiza

Docker no guarda estos valores en ningún sitio. Los calcula recorriendo los ficheros cada vez que se le pregunta, así que Dozzle pregunta poco.

La capa escribible se mide:

- una vez por cada contenedor, poco después de que Dozzle arranque
- una vez más cuando un contenedor se para, ya que su capa no puede cambiar después
- en un contenedor en marcha, después de que haya escrito unos 100 MB, o cada 5 minutos si ha escrito algo

Los volúmenes se miden justo después de esa primera pasada y luego como mucho cada 20 minutos. Docker solo puede medir todos los volúmenes a la vez, así que cada actualización recorre todos los volúmenes del host. Un contenedor creado entre medias muestra sus volúmenes en la siguiente actualización.

Todo lo que viene después de la primera pasada solo ocurre mientras alguien tiene Dozzle abierto. Un contenedor muestra `–` hasta su primera medición.

## Limitaciones

- Kubernetes no tiene capa escribible ni volúmenes de Docker que informar, así que la columna se oculta en [modo k8s](/es/guide/k8s).
- Los volúmenes de un driver de plugin normalmente no pueden informar de su tamaño y se omiten.
- Un [agente](/es/guide/agent) mide sus propios contenedores. Uno más antiguo que el Dozzle al que informa no envía tamaños, y sus contenedores muestran `–`.
