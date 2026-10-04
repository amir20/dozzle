---
title: Acciones sobre contenedores
sourceHash: 43d27d69a1ca
---

# Acciones sobre contenedores

<Badge type="warning" text="Solo Docker" />

Dozzle permite ejecutar acciones sobre los contenedores: `start`, `stop`, `restart`, `remove` y `update` desde el menú desplegable de la derecha, junto a las estadísticas del contenedor. Esta función está **desactivada** por defecto y se activa poniendo la variable de entorno `DOZZLE_ENABLE_ACTIONS` a `true`.

La acción `update` descarga la última imagen del contenedor y lo recrea con la misma configuración, algo útil para actualizar un contenedor sin tocar su archivo de Compose. `update` solo tiene efecto real cuando la imagen usa una etiqueta móvil (por ejemplo, `latest` o `stable`); con una etiqueta fija se volverá a descargar la misma imagen.

El contenedor antiguo se conserva, renombrado, hasta que el nuevo lleva 10 segundos en marcha sin reiniciarse y, si su imagen tiene healthcheck, se ha declarado sano. Si el nuevo contenedor no arranca, termina, se reinicia o pasa a no sano, Dozzle lo elimina y vuelve a poner el antiguo, y la actualización indica **revertido** con el motivo. El nuevo contenedor lleva la etiqueta `dev.dozzle.previous-image` con el id de la imagen que reemplazó, y `dev.dozzle.previous-ref` con el digest `repo@sha256:…` de esa imagen (no existe para imágenes construidas localmente). Un contenedor que no estaba en marcha, como una tarea puntual que ya terminó, se recrea con la nueva imagen y se deja detenido, así que no se vuelve a ejecutar ni se comprueba.

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

Para dejar de comprobar un contenedor concreto, por ejemplo uno fijado a una versión a propósito, ponle esta etiqueta. También lo saca de la [programación de actualizaciones automáticas](/es/guide/actions#auto-updating-containers).

```yaml [docker-compose.yml]
services:
  database:
    image: postgres:18-alpine
    labels:
      dev.dozzle.update: off
```

También se puede mostrar una notificación cuando hay una actualización. Viene desactivada y está en Ajustes.

### Lo que no se puede comprobar

Algunos contenedores no tienen nada con lo que comparar, y Dozzle se calla en vez de adivinar:

- Imágenes construidas en local, que no llevan digest de registro
- Referencias fijadas a un digest, que no pueden cambiar
- Registros privados, ya que Dozzle no tiene credenciales propias. En Kubernetes esto incluye las imágenes descargadas con `imagePullSecrets`.

### Kubernetes

En el modo Kubernetes, la comprobación compara el digest que registra el estado del pod con el del registro, así que funciona en clústeres con containerd como k3s, EKS y GKE sin permisos adicionales. El aviso es solo informativo y nunca trae un botón `Update`, porque la imagen de un pod pertenece a su workload. Para una etiqueta que cambia como `:latest` con `imagePullPolicy: Always`, un [reinicio progresivo](/es/guide/k8s#rollout-restart) obtiene la imagen nueva. Cualquier otra cosa es un cambio en la spec del workload. El botón y el panel de actualizaciones del dashboard no se muestran en el modo Kubernetes.

### Actualizar el propio Dozzle

La acción `Update` sobre el propio contenedor de Dozzle actualiza Dozzle en el sitio. Descarga la nueva imagen y deja el cambio en manos de un contenedor auxiliar de corta duración, así que Dozzle desaparece unos segundos y vuelve con la nueva versión, la misma configuración y los mismos volúmenes. También puede ejecutarse de forma programada. Consulta [Cómo se actualiza Dozzle a sí mismo](/es/guide/setup-wizard#self-update) para ver qué se conserva y qué instalaciones no están soportadas. Ejecutar Dozzle como servicio de Swarm lo actualiza a través del orquestador. Los agentes de Dozzle en otros hosts son contenedores normales y se actualizan como cualquier otro.

## Actualizar varios contenedores a la vez

Con las acciones activadas, el panel comprueba todos los contenedores de una sola pasada. Los contenedores desactualizados muestran un pequeño anillo junto a su nombre, y aparece un botón **N actualizaciones** encima de la lista de contenedores. Ambos abren el panel de actualizaciones, que lista todos los contenedores con una imagen más reciente, todos seleccionados. Desmarca lo que quieras dejar como está y pulsa **Actualizar**.

Los hosts se actualizan en paralelo, y cada host actualiza un contenedor cada vez, así que ningún daemon tiene que descargar una docena de imágenes a la vez. El panel muestra cómo cada contenedor pasa por descargando, recreando y actualizado, y un fallo en un contenedor no detiene al resto. La actualización se ejecuta en el servidor, así que cerrar la pestaña no la interrumpe. Si vuelves a abrir el panel, retoma el punto en el que está.

Si el propio contenedor de Dozzle está en la lista, siempre va el último, porque actualizarlo reinicia Dozzle.

Con `DOZZLE_IMAGE_CHECK_MODE=manual`, el botón dice **Buscar actualizaciones** hasta que lo pulsas. Con las acciones desactivadas, el panel se ve exactamente igual que hasta ahora, y el menú de cada contenedor sigue indicando cuándo hay una actualización disponible.

## Actualizar contenedores automáticamente {#auto-updating-containers}

Dozzle puede actualizar contenedores de forma programada, como Watchtower. Se configura en **Ajustes → Actualizaciones** o en el [asistente de configuración](/es/guide/setup-wizard#auto-update):

- **Cuándo:** desactivado, cada día o cada domingo, a una hora del día. Lo mismo que `DOZZLE_AUTO_UPDATE` y `DOZZLE_AUTO_UPDATE_TIME`.
- **Qué contenedores:** **Solo Dozzle**, **Dozzle y los contenedores que elija** (por defecto) o **Todo**. Así se trata un contenedor para el que nadie ha elegido.

Cada contenedor está en **Automático**, **Manual** o **Desactivado**. Elígelo en la página Actualizaciones, con **Actualizar automáticamente** en el panel de actualizaciones o desde el menú del contenedor. O fíjalo con una etiqueta, que manda sobre la interfaz:

| `dev.dozzle.update` | Qué pasa                                                             |
| ------------------- | -------------------------------------------------------------------- |
| `auto`              | Se actualiza según la programación                                   |
| _(sin etiqueta)_    | Manual: se comprueba y se muestra como actualización, que aplicas tú |
| `off`               | Nunca se comprueba ni se actualiza                                   |

```yaml [docker-compose.yml]
services:
  whoami:
    image: traefik/whoami:latest
    labels:
      dev.dozzle.update: auto
```

A la hora programada, Dozzle compara cada contenedor automático con su registro y actualiza solo los que tienen una imagen más reciente, Dozzle el último. Cada actualización se vigila y se revierte si el nuevo contenedor falla. Se omiten los contenedores que no están sanos, los que Dozzle [no puede comprobar](#lo-que-no-se-puede-comprobar) y los que alguien [revirtió](#rolling-back) desde la imagen ofrecida.

Con **Todo**, una base de datos con una etiqueta móvil como `postgres:latest` puede pasar a una versión mayor que no lea sus archivos de datos. La página Actualizaciones lista los contenedores que guardan datos en volúmenes con nombre y los deja en manual con un clic. Con **Todo**, los contenedores detenidos no se tocan salvo que una etiqueta o una elección los ponga en Automático.

Lo que eliges en la interfaz se guarda en [`dozzle.yml`](/es/guide/setup-wizard), así que necesita `/data` en un volumen. La actualización automática funciona en modo servidor, incluidos los contenedores en [agentes remotos](/es/guide/agent), y necesita las acciones activadas. ¿Vienes de Watchtower? Consulta [Pasar de Watchtower a Dozzle](/es/guide/moving-from-watchtower).

## Limpiar imágenes antiguas {#cleaning-up-old-images}

Cada actualización deja en el host la imagen que reemplazó, así que Dozzle elimina las imágenes antiguas después de una actualización, como el `--cleanup` de Watchtower. Siempre se hace, en todas las actualizaciones: programadas, desde la acción `Update` de un contenedor o desde el panel de actualizaciones. No hay nada que activar.

Dozzle conserva la imagen con la que funcionaba el contenedor hasta ahora, para que aún pueda volver a ella, y elimina la anterior. Una actualización de 1.4.1 a 1.4.2 elimina 1.4.0 y conserva 1.4.1, así que cada contenedor guarda como mucho una imagen de reserva. Dozzle lee qué imagen eliminar de la etiqueta `dev.dozzle.previous-image` del contenedor antiguo, por lo que la primera actualización de un contenedor no elimina nada.

La limpieza solo se hace cuando la actualización se ha completado y el contenedor antiguo ya no existe. Una actualización revertida no elimina nada. Dozzle solo elimina una imagen sin tag que ningún contenedor use: una imagen que aún tiene un tag, como una que descargaste o construiste tú, se conserva, y la eliminación no se fuerza, así que Docker se niega mientras otro contenedor, en marcha o detenido, la siga usando. Una negativa nunca hace fallar la actualización.

Los contenedores en [agentes remotos](/es/guide/agent) se limpian de la misma forma. Los servicios de Swarm no se limpian, porque cada nodo guarda sus propias imágenes y Swarm poda su propio historial de tareas.

## Revertir {#rolling-back}

Después de actualizar un contenedor, el menú del contenedor ofrece **Revertir a** la imagen que ejecutaba antes, y cada contenedor actualizado en el panel de actualizaciones recibe un enlace **Revertir**. Dozzle conoce esa imagen por su propia actualización, mediante la etiqueta `dev.dozzle.previous-image` que la actualización dejó en el contenedor, o por las actualizaciones que vio en el host desde que arrancó, lo que cubre una actualización hecha por Watchtower o `docker compose`. La opción solo aparece cuando Dozzle conoce la imagen anterior.

Revertir reemplaza el contenedor igual que una actualización: el contenedor actual se conserva hasta que la imagen anterior se mantiene en marcha, y se restaura si no lo hace. La configuración y los volúmenes no cambian. Si la imagen anterior se eliminó del host, Dozzle la descarga de nuevo por su digest. Nunca descarga un tag, porque el tag ya apunta a la imagen más nueva, así que una imagen construida localmente que ya no está no se puede recuperar.

Dozzle pide confirmación antes de revertir y avisa de dos cosas. La versión más nueva puede haber migrado los datos de los volúmenes del contenedor a un formato que la anterior no puede leer. Y en un proyecto compose, el siguiente `docker compose pull` vuelve a traer la imagen más nueva, salvo que el archivo compose fije la anterior.

Después, la programación de actualizaciones automáticas omite la imagen más nueva para ese contenedor hasta que su tag apunte otra vez a una más reciente. Cuando la reversión se ha mantenido en marcha, la imagen desde la que se revirtió se [limpia](#cleaning-up-old-images) como cualquier imagen antigua, así que solo se elimina cuando ya no la nombra ningún tag.

Revertir funciona con contenedores independientes, incluidos los de [agentes remotos](/es/guide/agent). No está disponible para servicios de Swarm, Kubernetes ni para el propio contenedor de Dozzle.
