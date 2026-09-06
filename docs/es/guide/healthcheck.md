---
title: Healthcheck
sourceHash: 4ab4dd21a26a
---

# Activar el healthcheck

Dozzle incluye un subcomando `dozzle healthcheck`. No viene conectado a la imagen por defecto porque añade algo de consumo de CPU. Actívalo desde tu fichero de compose:

```yaml
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    ports:
      - 8080:8080
    healthcheck:
      test: ["CMD", "/dozzle", "healthcheck"]
      interval: 3s
      timeout: 30s
      retries: 5
      start_period: 30s
```

## Qué comprueba

Cuando se ejecuta como servidor, `dozzle healthcheck` hace una petición HTTP `GET` a su propio endpoint `/healthcheck`. El endpoint hace ping a todos los clientes de Docker **locales** (hasta 3s por cliente) y devuelve:

- `200 OK`: al menos un cliente de Docker local ha respondido, **o** no hay clientes locales configurados pero se conoce al menos un host de agente remoto.
- `500 Internal Server Error`: el ping ha fallado en todos los clientes locales y no se conoce ningún host de agente.

Los agentes remotos quedan fuera del healthcheck del servidor a propósito: un agente inalcanzable no debería marcar como enfermo al proceso principal de Dozzle. Cada agente puede exponer su propio healthcheck; consulta [Healthcheck del agente](/es/guide/agent#setting-up-healthcheck).

## Códigos de salida

- `0`: sano (HTTP 200)
- distinto de cero: enfermo, error de red o respuesta distinta de 200. La URL que ha fallado y el estado se escriben en stdout.

El comando respeta `--addr` y `--base`, así que funciona con puertos y rutas base personalizados sin configuración adicional.

> [!WARNING]
> El comando `healthcheck` no funciona con la opción `--health-cmd` por un fallo de Docker. Usa el bloque `healthcheck` en `docker-compose.yml` como se muestra arriba. Consulta [docker/cli#3719](https://github.com/docker/cli/issues/3719) para más detalles.
