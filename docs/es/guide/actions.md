---
title: Acciones sobre contenedores
sourceHash: 0b048a644851
---

# Acciones sobre contenedores

<Badge type="warning" text="Solo Docker" />

Dozzle permite ejecutar acciones sobre los contenedores: `start`, `stop`, `restart`, `remove` y `update` desde el menú desplegable de la derecha, junto a las estadísticas del contenedor. Esta función está **desactivada** por defecto y se activa poniendo la variable de entorno `DOZZLE_ENABLE_ACTIONS` a `true`.

La acción `update` descarga la última imagen del contenedor y lo recrea con la misma configuración, algo útil para actualizar un contenedor sin tocar su archivo de Compose. `update` solo tiene efecto real cuando la imagen usa una etiqueta móvil (por ejemplo, `latest` o `stable`); con una etiqueta fija se volverá a descargar la misma imagen.

> [!WARNING]
> `remove` elimina el contenedor: se pierden los datos de su capa de escritura y sus volúmenes anónimos quedan sueltos, sin contenedor. `update` recrea el contenedor y conserva todos los volúmenes, también los anónimos, y todos los bind mounts. Solo se pierden los datos escritos en la capa de escritura del contenedor.

::: code-group

```sh
docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --enable-actions
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    ports:
      - 8080:8080
    environment:
      DOZZLE_ENABLE_ACTIONS: true
```

:::

## Actualizaciones {#auto-updating-containers}

Cada actualización conserva el contenedor antiguo hasta que el nuevo se mantiene en marcha, y lo restaura si no. La comprobación de imágenes nuevas, la actualización de varios contenedores a la vez, la programación de actualizaciones automáticas, la limpieza y la reversión se explican en [Actualizaciones de contenedores](/es/guide/updates).
