---
title: Acciones sobre contenedores
sourceHash: 121d4b250806
---

# Acciones sobre contenedores

<Badge type="warning" text="Solo Docker" />

Dozzle permite ejecutar acciones sobre los contenedores: `start`, `stop`, `restart`, `remove` y `update` desde el menú desplegable de la derecha, junto a las estadísticas del contenedor. Esta función está **desactivada** por defecto y se activa poniendo la variable de entorno `DOZZLE_ENABLE_ACTIONS` a `true`.

La acción `update` descarga la última imagen del contenedor y lo recrea con la misma configuración, algo útil para actualizar un contenedor sin tocar su archivo de Compose. `update` solo tiene efecto real cuando la imagen usa una etiqueta móvil (por ejemplo, `latest` o `stable`); con una etiqueta fija se volverá a descargar la misma imagen.

El contenedor antiguo se conserva, renombrado, hasta que el nuevo lleva 10 segundos en marcha sin reiniciarse y, si su imagen tiene healthcheck, se ha declarado sano. Si el nuevo contenedor no arranca, termina, se reinicia o pasa a no sano, Dozzle lo elimina y vuelve a poner el antiguo, y la actualización indica **revertido** con el motivo. El nuevo contenedor lleva la etiqueta `dev.dozzle.previous-image` con el id de la imagen que reemplazó, y `dev.dozzle.previous-ref` con el digest `repo@sha256:…` de esa imagen (no existe para imágenes construidas localmente). Un contenedor detenido nunca se actualiza, porque puede estar detenido a propósito. Se sigue mostrando que hay una imagen más reciente, pero no se ofrece **Actualizar**, la programación lo omite, y una actualización pedida de todos modos, por ejemplo desde Dozzle Cloud, se rechaza con «Inicia primero el contenedor». Cuando se inicia, ya se puede actualizar.

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

Para dejar de comprobar un contenedor concreto, por ejemplo uno fijado a una versión a propósito, ponle esta etiqueta. También lo saca de la [actualización automática programada](#auto-updating-containers).

```yaml [docker-compose.yml]
services:
  database:
    image: postgres:18-alpine
    labels:
      dev.dozzle.update: off
```

También se puede mostrar una notificación cuando hay una actualización. Viene desactivada y está en Configuración.

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

Dozzle puede actualizar contenedores de forma programada. Se configura en **Configuración → Actualizaciones** o en el [asistente de configuración](/es/guide/setup-wizard#auto-update):

- **Cuándo:** desactivada, a diario o cada semana el domingo, a una hora del día. Igual que `DOZZLE_AUTO_UPDATE` y `DOZZLE_AUTO_UPDATE_TIME`.
- **Qué contenedores:** **Solo Dozzle**, **Contenedores con etiqueta** (por defecto) o **Todo**. Dozzle sigue la programación en los tres casos. Igual que `DOZZLE_UPDATE_CONTAINERS` (`off`, `labelled` o `all`).

Una etiqueta en el contenedor decide el resto:

| `dev.dozzle.update` | Qué pasa                                                                                                             |
| ------------------- | -------------------------------------------------------------------------------------------------------------------- |
| `auto`              | Se actualiza según la programación, salvo que **Qué contenedores** sea **Solo Dozzle**                               |
| _(sin etiqueta)_    | Se actualiza según la programación con **Todo**. Si no, se comprueba y se muestra como actualización, que aplicas tú |
| `off`               | Nunca se comprueba ni se actualiza                                                                                   |

```yaml
services:
  app:
    image: ghcr.io/example/app:latest
    labels:
      dev.dozzle.update: auto
```

Las etiquetas antiguas siguen funcionando: `dev.dozzle.auto-update=true` cuenta como `auto`, y `dev.dozzle.update-check=false` como `off`.

A la hora programada, Dozzle compara cada contenedor de la programación con su registro y actualiza solo los que tienen una imagen más reciente, el propio Dozzle el último. Cada actualización es el intercambio seguro descrito arriba: si el contenedor nuevo no se mantiene en marcha, vuelve el anterior. Se omiten los contenedores detenidos o que no están sanos, los que Dozzle [no puede comprobar](#lo-que-no-se-puede-comprobar) y los que se [revirtieron](#rolling-back) desde la imagen que se ofrece. **Configuración → Actualizaciones** muestra los contenedores que actualizará la próxima ejecución.

Con **Todo**, una base de datos con un tag flotante como `postgres:latest` puede saltar a una versión mayor cuyos archivos de datos no sabe leer. Al elegir **Todo** se muestran los contenedores que guardan datos en volúmenes con nombre. Ponles la etiqueta `dev.dozzle.update: off` para dejarlos fuera.

**Qué contenedores** se guarda en [`dozzle.yml`](/es/guide/setup-wizard#dozzle-yml) como `updateContainers`, así que cambiarlo desde la interfaz requiere `/data` en un volumen. La actualización automática funciona en modo servidor, también para los contenedores de [agentes remotos](/es/guide/agent), y requiere las acciones activadas. ¿Vienes de Watchtower? Consulta [Pasar de Watchtower a Dozzle](/es/guide/moving-from-watchtower).

## Limpiar imágenes antiguas {#cleaning-up-old-images}

Cada actualización deja en el host la imagen que reemplazó, así que Dozzle elimina las imágenes antiguas después de una actualización, como el `--cleanup` de Watchtower. Siempre se hace, en todas las actualizaciones: programadas, desde la acción `Update` de un contenedor o desde el panel de actualizaciones. No hay nada que activar.

Dozzle conserva la imagen con la que funcionaba el contenedor hasta ahora, para que aún pueda volver a ella, y elimina la anterior. Una actualización de 1.4.1 a 1.4.2 elimina 1.4.0 y conserva 1.4.1, así que cada contenedor guarda como mucho una imagen de reserva. Dozzle lee qué imagen eliminar de la etiqueta `dev.dozzle.previous-image` del contenedor antiguo, por lo que la primera actualización de un contenedor no elimina nada.

La limpieza solo se hace cuando la actualización se ha completado y el contenedor antiguo ya no existe. Una actualización revertida no elimina nada. Dozzle solo elimina una imagen sin tag que ningún contenedor use: una imagen que aún tiene un tag, como una que descargaste o construiste tú, se conserva, y la eliminación no se fuerza, así que Docker se niega mientras otro contenedor, en marcha o detenido, la siga usando. Una negativa nunca hace fallar la actualización.

Los contenedores en [agentes remotos](/es/guide/agent) se limpian de la misma forma, y también el propio contenedor de Dozzle: el contenedor auxiliar de la [autoactualización](/es/guide/setup-wizard#self-update) elimina la imagen anterior a la previa en cuanto el nuevo Dozzle sigue en marcha. Los servicios de Swarm, incluido Dozzle cuando se ejecuta como uno, no se limpian, porque cada nodo guarda sus propias imágenes y Swarm poda su propio historial de tareas.

## Revertir una actualización {#rolling-back}

Revertir una actualización es una función de [Dozzle Cloud](/es/guide/dozzle-cloud). Dozzle Cloud vigila cada actualización que hace la programación y, cuando la nueva versión empieza a fallar, ofrece revertirla. Dozzle vuelve entonces a poner el contenedor en la imagen que ejecutaba antes, la que indica su etiqueta `dev.dozzle.previous-image`, igual que en una actualización: el contenedor actual se conserva hasta que la imagen anterior se mantiene en marcha, y vuelve a ponerse si no. La configuración y los volúmenes no cambian. No se descarga nada: si la imagen anterior ya no está en el host, la reversión falla y el contenedor queda como estaba.

El contenedor revertido lleva la etiqueta `dev.dozzle.rolled-back-from` con la imagen que dejó, y la actualización automática programada lo deja en paz hasta que su tag apunte a una imagen más reciente. Cuando la reversión se mantiene estable, la imagen que se dejó se [limpia](#cleaning-up-old-images) como cualquier imagen antigua, así que solo se elimina cuando ya ningún tag apunta a ella.

La reversión funciona con contenedores independientes, también los de [agentes remotos](/es/guide/agent). No está disponible para servicios de Swarm, Kubernetes ni el propio contenedor de Dozzle.

## Actualizaciones en la vista de logs

Cuando Dozzle actualiza un contenedor, según la programación, desde la interfaz de Dozzle o desde Dozzle Cloud, los logs del contenedor nuevo empiezan con una marca que indica la imagen de origen y de destino y qué inició la actualización. También se marcan una reversión y una actualización deshecha. Con Dozzle Cloud vinculado, la marca muestra además lo que Dozzle Cloud opina de la actualización, con un enlace a ella. Dozzle guarda sus actualizaciones recientes en memoria, así que la marca desaparece cuando Dozzle se reinicia.
