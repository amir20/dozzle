---
title: Enlaces de contenedores
sourceHash: d39d7f2119c5
---

# Enlaces de contenedores

La mayoría de los contenedores que merece la pena vigilar también sirven una interfaz web. Añade la etiqueta `dev.dozzle.url` y Dozzle mostrará un enlace a ella junto al nombre del contenedor, para que puedas saltar de los logs a la propia aplicación.

::: code-group

```sh
docker run --label dev.dozzle.url=https://grafana.example.com grafana/grafana
```

```yaml [docker-compose.yml]
services:
  grafana:
    image: grafana/grafana
    labels:
      - dev.dozzle.url=https://grafana.example.com
```

:::

El enlace aparece en tres sitios: la barra lateral, la tabla de contenedores y la barra de título de la página del contenedor. Siempre se abre en una pestaña nueva, y al pulsarlo nunca sales de los logs.

## Qué acepta la etiqueta

El valor tiene que ser una URL absoluta `http` o `https`. Cualquier otra cosa, una ruta relativa, un nombre de host suelto u otro esquema, se ignora y no se muestra ningún enlace.

Dozzle no comprueba que la URL resuelva, y tampoco la reescribe según el host. El enlace abre exactamente lo que escribas, así que usa una dirección que funcione desde el navegador en el que ves Dozzle.

## Por qué no hay detección automática

Dozzle sabe qué puertos publica un contenedor, pero un puerto publicado no equivale a una URL accesible. Los proxies inversos, las rutas personalizadas, TLS y las redes separadas hacen que la conjetura falle lo bastante a menudo como para resultar molesta. La etiqueta lo deja explícito: Dozzle solo muestra el enlace que tú has escrito.

En los contenedores que aún no tienen etiqueta, Dozzle muestra un icono de enlace tenue junto al nombre, tanto en el panel principal como en la página del contenedor. Al pulsarlo se abre un fragmento que puedes copiar a tu fichero de compose, ya relleno con una conjetura. Si lo descartas, la sugerencia desaparece en todas partes.

La conjetura sale de dos sitios. Primero se leen las etiquetas de router de Traefik, porque la regla de un router nombra una dirección que sí llega al contenedor desde un navegador:

```yaml
labels:
  - traefik.http.routers.grafana.rule=Host(`grafana.example.com`)
  - traefik.http.routers.grafana.tls=true
```

Los prefijos de ruta se añaden al final, el esquema sale de la configuración de TLS y de los entrypoints del router, y `traefik.enable=false` lo desactiva todo. Si eso falla, Dozzle recurre a un puerto publicado en el host combinado con el nombre de host desde el que estás viendo Dozzle. Ambos se limitan a rellenar el fragmento. Ninguno se convierte en un enlace hasta que tú lo escribas en `dev.dozzle.url`.

Los contenedores detrás de un proxy inverso no publican ningún puerto en el host, así que las etiquetas de Traefik suelen ser la única señal disponible. Si usas otro proxy, la sugerencia no aparece y tendrás que añadir la etiqueta a mano.

## Swarm

En Swarm, `deploy.labels` define etiquetas en el servicio y la clave `labels` de nivel superior las define en el contenedor. El proveedor de swarm de Traefik lee las etiquetas del servicio, así que es ahí donde todo el mundo las pone:

```yaml
services:
  ui:
    image: my/ui
    deploy:
      labels:
        - traefik.http.routers.ui.rule=Host(`app.example.com`)
        - dev.dozzle.url=https://app.example.com
```

Dozzle vuelca las etiquetas del servicio en cada contenedor de tarea, así que tanto `dev.dozzle.url` como la sugerencia de Traefik funcionan desde `deploy.labels`. Lo mismo vale para `dev.dozzle.name`, `dev.dozzle.group` y `dev.dozzle.icon`. Una etiqueta puesta en el propio contenedor tiene prioridad sobre la del servicio.

Listar servicios es una API exclusiva de los managers. En un swarm de varios nodos, los agentes de los nodos worker no pueden leer las etiquetas del servicio, así que los contenedores programados ahí solo ven las suyas.

## Etiquetas relacionadas

- [`dev.dozzle.name`](/es/guide/container-names) define un nombre personalizado
- [`dev.dozzle.group`](/es/guide/container-groups) agrupa contenedores
- [`dev.dozzle.icon`](/es/guide/app-icons) elige el icono de la aplicación
