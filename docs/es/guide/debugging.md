---
title: Depuración
sourceHash: 2cb9b9a633e2
---

# Depurar con logs

Por defecto Dozzle registra al nivel `info`, que es deliberadamente escueto. Cuando algo no funciona, sube el nivel de detalle con la opción `--level` o la variable de entorno `DOZZLE_LEVEL`.

| Nivel   | Cuándo usarlo                                                                                              |
| ------- | ---------------------------------------------------------------------------------------------------------- |
| `info`  | Por defecto. Detalles de arranque, errores y avisos.                                                       |
| `debug` | Diagnóstico por petición, decisiones de autenticación, conexiones de agentes, volcado de la configuración. |
| `trace` | Todo. Eventos de log individuales, cargas de la baliza, tramas gRPC. Muy ruidoso.                          |

Dozzle escribe todos los logs en `stdout`, así que `docker logs dozzle` es el sitio donde leerlos.

```yaml
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    ports:
      - 8080:8080
    environment:
      DOZZLE_LEVEL: debug
```

## Informar de un fallo

Si crees que has encontrado un fallo, abre una incidencia en [github.com/amir20/dozzle/issues](https://github.com/amir20/dozzle/issues). Incluye:

- La versión de Dozzle (visible en el pie de la interfaz o con `dozzle --version`)
- El modo de despliegue: server, swarm, k8s o agent
- La versión de Docker o Kubernetes
- La salida de log relevante a nivel `debug` o `trace`
- Los pasos para reproducirlo, a ser posible con un `docker-compose.yml` mínimo

Cuanto más contexto tenga el informe inicial, antes se podrá clasificar.
