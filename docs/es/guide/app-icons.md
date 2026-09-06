---
title: Iconos de aplicaciones
sourceHash: b11b42409e46
---

# Iconos de aplicaciones

Dozzle reconoce las imágenes de contenedor más conocidas, las asocia al logo de su proyecto y lo muestra junto al nombre del contenedor en la barra lateral, la tabla de contenedores y la paleta de comandos. Si tienes un stack \*arr, Plex o Home Assistant, la lista se lee mucho más rápido.

Los iconos van incluidos en Dozzle. Nunca se descargan de una CDN, así que nada sobre tus contenedores sale de tu red y todo funciona sin conexión a internet.

## Cómo desactivarlo

La opción está en **Configuración → Opciones → Mostrar iconos de aplicaciones**. Es un ajuste por perfil, así que solo afecta a tu navegador.

## Cómo funciona la coincidencia

Dozzle mira el nombre de la imagen, ignorando el registro, la etiqueta y el digest. Gana el último segmento de la ruta, así que todos estos resuelven a Sonarr:

- `sonarr`
- `linuxserver/sonarr:latest`
- `lscr.io/linuxserver/sonarr`
- `ghcr.io/hotio/sonarr@sha256:...`

Cuando el nombre de la imagen es genérico, Dozzle recurre al espacio de nombres. Así es como `ghcr.io/goauthentik/server` resuelve a Authentik.

## Cómo cambiar el icono

Algunas imágenes no coinciden con nada, y un fork puede acabar con el logo equivocado. Usa la etiqueta `dev.dozzle.icon` para elegir tú el icono, o ponla a `none` para ocultarlo en ese contenedor.

::: code-group

```sh
docker run --label dev.dozzle.icon=plex my-custom-media-server
```

```yaml [docker-compose.yml]
services:
  media:
    image: my-custom-media-server
    labels:
      - dev.dozzle.icon=plex

  scratch:
    image: alpine
    labels:
      - dev.dozzle.icon=none
```

:::

El valor es un nombre de icono de [dashboard-icons](https://github.com/homarr-labs/dashboard-icons). Solo están disponibles los iconos que Dozzle incluye. Un nombre desconocido se queda sin icono.

## ¿Falta algún icono?

Dozzle incluye una selección cuidada en lugar del set completo de 3.000 iconos, para que la imagen siga siendo pequeña. Si falta algo popular, [abre una incidencia](https://github.com/amir20/dozzle/issues) con el nombre de la imagen y se puede añadir.
