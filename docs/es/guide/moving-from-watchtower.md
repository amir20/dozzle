---
title: Pasar de Watchtower a Dozzle
sourceHash: 3c48dbce256f
---

# Pasar de Watchtower a Dozzle

Dozzle puede hacer lo que hace Watchtower: comprobar de forma programada si tus contenedores tienen imágenes más recientes y actualizarlos. Cada actualización se vigila y, si el contenedor nuevo no se mantiene en marcha, vuelve el anterior. Esta página relaciona los ajustes de Watchtower con los de Dozzle.

## Activarlo

Dozzle necesita las acciones activadas y `/data` en un volumen:

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - dozzle-data:/data
    ports:
      - 8080:8080
    environment:
      DOZZLE_ENABLE_ACTIONS: true
      DOZZLE_AUTO_UPDATE: daily
      DOZZLE_AUTO_UPDATE_TIME: "04:00"
volumes:
  dozzle-data:
```

Puedes quitar las dos líneas `DOZZLE_AUTO_UPDATE` y configurar la programación en **Configuración → Actualizaciones**. Elige también **Qué contenedores** allí. Después detén Watchtower, para que los dos no actualicen los mismos contenedores.

## Ajustes

| Watchtower                                                        | Dozzle                                                                                                                                                                                                          |
| ----------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `--schedule` o `--interval`                                       | `DOZZLE_AUTO_UPDATE` (`daily`, o `weekly` el domingo) y `DOZZLE_AUTO_UPDATE_TIME`. Como mucho una vez al día                                                                                                    |
| Todos los contenedores (por defecto)                              | **Qué contenedores: Todo**                                                                                                                                                                                      |
| `--label-enable` con `com.centurylinklabs.watchtower.enable=true` | **Qué contenedores: Contenedores con etiqueta** (por defecto) con `dev.dozzle.update: auto`                                                                                                                     |
| `com.centurylinklabs.watchtower.enable=false`                     | `dev.dozzle.update: off`                                                                                                                                                                                        |
| `--cleanup`                                                       | Siempre activo. La imagen antigua sin tag se elimina tras una actualización, y se conserva una imagen anterior para revertir                                                                                    |
| `--monitor-only`                                                  | Dejar el contenedor sin etiqueta con **Contenedores con etiqueta**: se comprueba y se muestra como actualización, pero nunca se actualiza solo                                                                  |
| `--rolling-restart`                                               | Siempre: cada host actualiza un contenedor cada vez                                                                                                                                                             |
| `--notification-url`                                              | [Alertas y webhooks](/es/guide/alerts-and-webhooks) sobre eventos de contenedores, o [Dozzle Cloud](/es/guide/dozzle-cloud), que vigila cada actualización y te avisa cuando una versión nueva empieza a fallar |
| `--run-once`                                                      | **Actualizar** en el panel de actualizaciones                                                                                                                                                                   |
| Credenciales de registros privados                                | No se admiten. Los contenedores de un registro privado se omiten                                                                                                                                                |

## Etiquetas

Las etiquetas de Watchtower no se leen. Sustituye `com.centurylinklabs.watchtower.enable` por `dev.dozzle.update`:

| `dev.dozzle.update` | Qué pasa                                                                                                             |
| ------------------- | -------------------------------------------------------------------------------------------------------------------- |
| `auto`              | Se actualiza según la programación, salvo que **Qué contenedores** sea **Solo Dozzle**                               |
| _(sin etiqueta)_    | Se actualiza según la programación con **Todo**. Si no, se comprueba y se muestra como actualización, que aplicas tú |
| `off`               | Nunca se comprueba ni se actualiza                                                                                   |

Las etiquetas antiguas de Dozzle se siguen aceptando: `dev.dozzle.auto-update=true` cuenta como `auto`, y `dev.dozzle.update-check=false` como `off`.

## Qué cambia

- El contenedor anterior se conserva hasta que el nuevo se mantiene en marcha, y sano si tiene healthcheck. Si no, vuelve el anterior.
- Los contenedores que no están sanos se omiten.
- Los contenedores detenidos nunca se actualizan, aunque tengan la etiqueta `auto`.
- Con [Dozzle Cloud](/es/guide/dozzle-cloud), una actualización programada que empieza a fallar se puede [revertir](/es/guide/actions#rolling-back). La programación deja ese contenedor en paz hasta que se publique una imagen más reciente.
- Las imágenes fijadas a un digest y las construidas en local se omiten, porque no hay nada más reciente con qué compararlas.
