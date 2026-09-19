---
title: Iconos de aplicaciones
sourceHash: 2b128908b958
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

## Usar tu propio icono

Para una imagen que Dozzle nunca va a incluir, como tu propio proyecto, la etiqueta puede llevar el icono en sí como data URI. No se sube ni se monta nada, y nada se descarga de la red.

```yaml [docker-compose.yml]
services:
  app:
    image: my-company/internal-thing
    labels:
      - dev.dozzle.icon=data:image/svg+xml;base64,PHN2ZyB4bWxucz0i...
```

Genera el valor con `base64`:

```sh
echo "data:image/svg+xml;base64,$(base64 < icon.svg | tr -d '\n')"
```

Se aceptan SVG, PNG y WebP, y el valor completo puede ocupar como máximo 16 KB. Cualquier otra cosa se queda sin icono. Los iconos se muestran a unos 20 px, así que un SVG optimizado o un WebP de 64 px es más que suficiente.

Si publicas una imagen, define la etiqueta en su Dockerfile y todos los que la ejecuten tendrán el icono sin configurar nada.

```dockerfile
LABEL dev.dozzle.icon="data:image/svg+xml;base64,PHN2ZyB4bWxucz0i..."
```

## ¿Falta algún icono?

Dozzle incluye una selección cuidada en lugar del set completo de 3.000 iconos, para que la imagen siga siendo pequeña. Si falta algo popular, [abre una incidencia](https://github.com/amir20/dozzle/issues) con el nombre de la imagen y se puede añadir.
