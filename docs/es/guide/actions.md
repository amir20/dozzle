---
title: Acciones sobre contenedores
sourceHash: 7eab1f511f5f
---

# Acciones sobre contenedores

<Badge type="warning" text="Solo Docker" />

Dozzle permite ejecutar acciones sobre los contenedores: `start`, `stop`, `restart`, `remove` y `update` desde el menú desplegable de la derecha, junto a las estadísticas del contenedor. Esta función está **desactivada** por defecto y se activa poniendo la variable de entorno `DOZZLE_ENABLE_ACTIONS` a `true`.

La acción `update` descarga la última imagen del contenedor y lo recrea con la misma configuración, algo útil para actualizar un contenedor sin tocar su archivo de Compose. `update` solo tiene efecto real cuando la imagen usa una etiqueta móvil (por ejemplo, `latest` o `stable`); con una etiqueta fija se volverá a descargar la misma imagen.

> [!WARNING]
> `remove` y `update` recrean el contenedor. Se perderán los datos escritos en **volúmenes anónimos** o en la capa de escritura del contenedor. Los volúmenes con nombre y los bind mounts se conservan.

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

## Comprobación de actualizaciones

Dozzle comprueba si la imagen que está ejecutando un contenedor sigue siendo la que sirve su registro. Cuando no coinciden, aparece un punto en el menú del contenedor y el menú indica que hay una actualización disponible.

La comprobación pide al registro el digest de la etiqueta con la que se creó el contenedor y lo compara con el digest que el contenedor está ejecutando de verdad. Lo hace con una petición `HEAD` al manifiesto de la imagen, así que no se descarga ninguna capa ni cuenta para los límites de descargas de Docker Hub. Las respuestas se guardan en caché seis horas, y una misma imagen se consulta una sola vez por muchos contenedores o hosts que la usen.

Como la comparación es contra lo que el contenedor está _ejecutando_, un contenedor sigue desactualizado hasta que se recrea, aunque ya se haya descargado una imagen más nueva en el host.

La comprobación es independiente de las acciones. Saber que un contenedor está desactualizado es útil tanto si Dozzle puede hacer algo al respecto como si no, así que el aviso aparece incluso con `DOZZLE_ENABLE_ACTIONS` desactivado. Solo el botón `Update` necesita las acciones.

### Cómo desactivarlo

`DOZZLE_IMAGE_CHECK_MODE` controla si Dozzle contacta con los registros.

| Valor       | Comportamiento                                                                     |
| ----------- | ---------------------------------------------------------------------------------- |
| `automatic` | Comprueba en segundo plano al abrir un contenedor.                                 |
| `manual`    | Nunca comprueba por su cuenta. El menú ofrece la acción «Buscar actualizaciones».  |
| `off`       | La función desaparece. No se registra ningún endpoint ni se hace ninguna petición. |

Por defecto toma el valor de `DOZZLE_RELEASE_CHECK_MODE`, así que si ya le has dicho a Dozzle que no busque versiones automáticamente, tampoco comprobará imágenes de forma automática.

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      DOZZLE_IMAGE_CHECK_MODE: off
```

Para silenciar un contenedor concreto, por ejemplo uno fijado a una versión a propósito, ponle esta etiqueta:

```yaml [docker-compose.yml]
services:
  database:
    image: postgres:18-alpine
    labels:
      dev.dozzle.update-check: false
```

También se puede mostrar una notificación cuando hay una actualización. Viene desactivada y está en Ajustes.

### Lo que no se puede comprobar

Algunos contenedores no tienen nada con lo que comparar, y Dozzle se calla en vez de adivinar:

- Imágenes construidas en local, que no llevan digest de registro
- Referencias fijadas a un digest, que no pueden cambiar
- Registros privados, ya que Dozzle no tiene credenciales propias
- Kubernetes, donde el despliegue de imágenes es cosa del clúster

### Actualizar el propio Dozzle

Dozzle no puede pararse a sí mismo para actualizarse, así que un contenedor de Dozzle independiente muestra el aviso de actualización con un enlace a las notas de la versión en lugar del botón `Update`. Ejecutar Dozzle como servicio de Swarm funciona con normalidad, porque la actualización queda en manos del orquestador. Los agentes de Dozzle en otros hosts son contenedores normales y se actualizan como cualquier otro.
