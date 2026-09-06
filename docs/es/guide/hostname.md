---
title: Nombre de host
sourceHash: 8769ba2c0e47
---

# Cambiar el nombre de host de Dozzle

La conexión por defecto de Dozzle se llama localhost. Con la opción `--hostname` puedes cambiar ese nombre por el que quieras. El valor aparece en el título de la página y debajo del logo de Dozzle.

Esto también cambia la etiqueta de la conexión `localhost` que se muestra en el menú multihost. El siguiente ejemplo usa `--hostname` para cambiar el subtítulo a `mywebsite.xyz`.

::: code-group

```sh
docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --hostname mywebsite.xyz
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
      DOZZLE_HOSTNAME: mywebsite.xyz
```

:::

## Multihost y agentes

`--hostname` solo renombra el host donde se ejecuta **este** proceso de Dozzle. Los [agentes](/es/guide/agent) remotos anuncian su propio nombre: define `DOZZLE_HOSTNAME` (o `--hostname`) en cada agente para controlar cómo aparece en el menú multihost. En [modo Swarm](/es/guide/swarm-mode) cada nodo ejecuta su propio agente, así que dale a cada nodo un nombre distinto para poder distinguirlos.
