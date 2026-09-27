---
title: Analíticas anónimas
sourceHash: 8421075e674a
---

# Recopilación de datos analíticos

Dozzle recopila datos de uso anónimos mediante una baliza ligera para ayudar a priorizar funciones y correcciones. Es un proyecto de código abierto sin financiación, así que estos datos son la señal principal para decidir dónde invertir el esfuerzo.

## Qué se recopila

Dozzle envía una baliza al arrancar y otra cada vez que alguien abre la interfaz. Entre las dos incluyen:

- la versión de Dozzle, el modo de despliegue (server, swarm, k8s, agent) y la versión de Docker Engine
- qué proveedor de autenticación está activo y si las acciones y la shell están activadas
- pequeños recuentos: hosts, agentes, contenedores en ejecución y filtros
- la cadena user agent del navegador, solo en la baliza de la interfaz
- el ID de Docker Engine, para no contar dos veces la misma instalación

Nunca se transmiten contenidos de logs, nombres de contenedores, nombres de imágenes, nombres de host ni identificadores de usuario. El conjunto exacto de campos cambia con el tiempo. La fuente de referencia es [`types/beacon.go`](https://github.com/amir20/dozzle/blob/master/types/beacon.go), y el envío está en [`internal/analytics/http_beacon.go`](https://github.com/amir20/dozzle/blob/master/internal/analytics/http_beacon.go).

## Dónde se almacenan los datos

Las balizas se envían a `https://b.dozzle.dev/event` y las recibe [drain](https://github.com/amir20/drain), un servicio en Go de código abierto que las guarda en una base de datos y en archivos Parquet para su análisis. drain no guarda la dirección IP de la que llegó una baliza, y los datos no se comparten con terceros.

## Cómo desactivarlo

Usa `--no-analytics` o define `DOZZLE_NO_ANALYTICS=true`. No se hará ninguna petición de la baliza.

```yaml
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      DOZZLE_NO_ANALYTICS: "true"
```
