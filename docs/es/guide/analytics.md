---
title: Analíticas anónimas
sourceHash: 0fedd8c524b8
---

# Recopilación de datos analíticos

Dozzle recopila datos de uso anónimos mediante una baliza ligera para ayudar a priorizar funciones y correcciones. Es un proyecto de código abierto sin financiación, así que estos datos son la señal principal para decidir dónde invertir el esfuerzo.

## Qué se recopila

A grandes rasgos, la baliza incluye cosas como la versión de Dozzle, el modo de despliegue (server, swarm, k8s, agent), qué proveedor de autenticación está activo, algunos indicadores de funciones, la versión del motor de Docker y unos pocos recuentos (número de hosts, contenedores, filtros). También se incluye un identificador aleatorio por instalación para eliminar duplicados.

Nunca se transmite el contenido de los logs, ni nombres de contenedores, ni nombres de imágenes, ni direcciones IP, ni identificadores de usuario. El conjunto exacto de campos va cambiando con el tiempo. La fuente autoritativa es [`types/beacon.go`](https://github.com/amir20/dozzle/blob/master/types/beacon.go), y quien los envía es [`internal/analytics/http_beacon.go`](https://github.com/amir20/dozzle/blob/master/internal/analytics/http_beacon.go).

## Dónde se almacenan los datos

Los eventos se envían a `https://b.dozzle.dev/event`, un pequeño servicio en Go que los escribe en un fichero plano en DigitalOcean para procesarlos más adelante.

## Cómo desactivarlo

Usa `--no-analytics` o define `DOZZLE_NO_ANALYTICS=true`. No se hará ninguna petición de la baliza.

```yaml
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      DOZZLE_NO_ANALYTICS: "true"
```
