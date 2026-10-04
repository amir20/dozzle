---
title: Pasar de Watchtower a Dozzle
sourceHash: acec3dd8eb61
---

# Pasar de Watchtower a Dozzle

Dozzle puede hacer lo que hace Watchtower: comprobar de forma programada si tus contenedores tienen imágenes más nuevas y actualizarlos. Cada actualización se vigila y se revierte si el nuevo contenedor no se mantiene en marcha. Esta página relaciona los ajustes de Watchtower con los de Dozzle.

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

También puedes quitar las dos líneas `DOZZLE_AUTO_UPDATE` y fijar la programación en **Ajustes → Actualizaciones**. Después detén Watchtower, para que los dos no actualicen los mismos contenedores.

## Ajustes

| Watchtower                                                        | Dozzle                                                                                                                                                                                                          |
| ----------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `--schedule` o `--interval`                                       | `DOZZLE_AUTO_UPDATE` (`daily`, o `weekly` el domingo) y `DOZZLE_AUTO_UPDATE_TIME`. Como mucho una vez al día                                                                                                    |
| Todos los contenedores (por defecto)                              | **Qué contenedores: Todo**                                                                                                                                                                                      |
| `--label-enable` con `com.centurylinklabs.watchtower.enable=true` | **Qué contenedores: Dozzle y los contenedores que elija** (por defecto) con `dev.dozzle.update: auto`                                                                                                           |
| `com.centurylinklabs.watchtower.enable=false`                     | `dev.dozzle.update: off`, o **Manual** para mantener la comprobación                                                                                                                                            |
| `--cleanup`                                                       | Siempre activo. La imagen antigua sin etiqueta se elimina tras una actualización, y se conserva una imagen anterior para revertir                                                                               |
| `--monitor-only`                                                  | **Manual**: el contenedor se comprueba y se muestra como actualización, pero nunca se actualiza solo                                                                                                            |
| `--rolling-restart`                                               | Siempre: cada host actualiza un contenedor cada vez                                                                                                                                                             |
| `--notification-url`                                              | [Alertas y webhooks](/es/guide/alerts-and-webhooks) sobre eventos de contenedores, o [Dozzle Cloud](/es/guide/dozzle-cloud), que vigila cada actualización y te avisa cuando una versión nueva empieza a fallar |
| `--run-once`                                                      | **Actualizar** en el panel de actualizaciones                                                                                                                                                                   |
| Credenciales de registros privados                                | No soportado. Los contenedores de un registro privado se omiten                                                                                                                                                 |

## Etiquetas

Las etiquetas de Watchtower no se leen. Cambia `com.centurylinklabs.watchtower.enable` por `dev.dozzle.update`:

| `dev.dozzle.update` | Qué pasa                                                             |
| ------------------- | -------------------------------------------------------------------- |
| `auto`              | Se actualiza según la programación                                   |
| _(sin etiqueta)_    | Manual: se comprueba y se muestra como actualización, que aplicas tú |
| `off`               | Nunca se comprueba ni se actualiza                                   |

Una etiqueta manda sobre lo elegido en la interfaz. Las etiquetas antiguas de Dozzle se siguen aceptando: `dev.dozzle.auto-update=true` cuenta como `auto`, y `dev.dozzle.update-check=false` como `off`.

## Qué cambia

- El contenedor antiguo se conserva hasta que el nuevo se mantiene en marcha, y sano si tiene healthcheck. Si no, se restaura el antiguo.
- Los contenedores que no están sanos se omiten.
- Un contenedor que alguien [revirtió](/es/guide/actions#rolling-back) no vuelve a la misma imagen hasta que se publica una más nueva.
- Las imágenes fijadas a un digest y las construidas localmente se omiten, porque no hay nada más nuevo con lo que comparar.
