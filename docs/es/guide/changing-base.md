---
title: Proxy inverso y ruta base
sourceHash: f344ea2a42cc
---

# Proxy inverso y ruta base

Es habitual poner Dozzle detrás de un proxy inverso para terminar TLS, autenticar o compartir un nombre de host con otros servicios. Esta página cubre tanto el montaje de Dozzle en una subruta como los ajustes del proxy necesarios para que el streaming funcione bien.

## Cambiar la ruta base

Por defecto Dozzle se monta en `/`. Puedes cambiarlo con el flag `--base` o con la variable de entorno `DOZZLE_BASE`. Por ejemplo, para montarlo en `/foobar`:

::: code-group

```sh
docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --base /foobar
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
      DOZZLE_BASE: /foobar
```

:::

Dozzle estará disponible en `http://localhost:8080/foobar/`. Esta opción reescribe todos los recursos a `/foobar/{file.path}` y redirige automáticamente `/foobar` a `/foobar/`.

## Requisitos del proxy

Dozzle envía los logs por **Server-Sent Events (SSE)** y usa **WebSocket** para las shells de los contenedores. Los proxies inversos deben:

1. **Desactivar el búfer de las respuestas**: SSE entrega los eventos según ocurren. Cualquier búfer hace que los logs lleguen a ráfagas o que no lleguen nunca. Dozzle envía `X-Accel-Buffering: no`, pero algunos proxies lo ignoran.
2. **Reenviar las cabeceras de actualización a WebSocket**: son necesarias para las funciones de shell y attach.
3. **No comprimir `text/event-stream`**: los middlewares de compresión suelen romper SSE.

## Nginx

```nginx
location ^~ /foobar/ {
    proxy_pass http://dozzle:8080;

    chunked_transfer_encoding off;
    proxy_buffering off;
    proxy_cache off;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
}
```

Quita el prefijo `^~ /foobar/` si Dozzle está montado en la raíz. Consulta también la entrada de la FAQ sobre [desactivar el búfer](/es/guide/faq#disabling-buffering-in-nginx).

## Traefik

Traefik gestiona la actualización a WebSocket automáticamente, pero el middleware `compress` por defecto rompe SSE. Excluye `text/event-stream`:

```yaml
http:
  middlewares:
    middlewares-compress:
      compress:
        excludedContentTypes:
          - text/event-stream
```

Y este sería un bloque de etiquetas típico en el servicio de Dozzle:

```yaml
services:
  dozzle:
    image: amir20/dozzle:latest
    labels:
      - traefik.enable=true
      - traefik.http.routers.dozzle.rule=Host(`dozzle.example.com`)
      - traefik.http.routers.dozzle.entrypoints=websecure
      - traefik.http.routers.dozzle.tls.certresolver=letsencrypt
      - traefik.http.services.dozzle.loadbalancer.server.port=8080
```

## Caddy

```caddyfile
dozzle.example.com {
    reverse_proxy dozzle:8080 {
        flush_interval -1
    }
}
```

`flush_interval -1` desactiva el búfer de respuesta en los endpoints de streaming.

## Problemas habituales

- **Página en blanco o recursos con error 404 al usar `--base`**: el proxy está quitando el prefijo de la ruta antes de reenviar la petición. Configúralo para que pase la ruta completa a Dozzle.
- **Los logs se cortan a los pocos segundos**: los tiempos de espera de conexión del proxy son demasiado cortos. Súbelos a varios minutos como mínimo (por ejemplo, `proxy_read_timeout 3600s` en Nginx).
- **La shell se desconecta al instante**: no se están reenviando las cabeceras de actualización a WebSocket. Comprueba las cabeceras `Upgrade` y `Connection`.
