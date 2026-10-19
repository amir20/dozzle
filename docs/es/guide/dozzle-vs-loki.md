---
title: Dozzle vs Grafana Loki
description: Si necesitas Grafana y Loki para leer logs de Docker o si te basta con Dozzle, y cómo usar los dos.
sourceHash: 95a7cee5f63a
---

# Dozzle vs Grafana Loki

Loki es un almacén de logs. Un recolector envía cada línea de log a Loki, Loki la guarda durante el tiempo que configures y Grafana es donde la consultas. Dozzle no guarda nada: lee los logs directamente de Docker, los mismos que muestra `docker logs`, y los transmite a tu navegador.

Así que la pregunta de verdad es si necesitas guardar los logs en algún sitio que no sea Docker.

## De un vistazo

|                         | Dozzle                                                                  | Grafana + Loki                                            |
| ----------------------- | ----------------------------------------------------------------------- | --------------------------------------------------------- |
| Qué ejecutas            | Un contenedor                                                           | Loki, Grafana y un recolector como Grafana Alloy          |
| Dónde están los logs    | En Docker, tal como ya los guarda                                       | Copiados en el almacenamiento propio de Loki              |
| Hasta cuándo            | Hasta donde llegue la rotación de logs de Docker                        | Tanto como fijes en la retención de Loki                  |
| Seguimiento en vivo     | La vista por defecto                                                    | Disponible en Grafana Explore                             |
| Consultas               | Búsqueda, expresiones regulares, [SQL](/es/guide/sql-engine) sobre JSON | LogQL, incluidas métricas calculadas a partir de los logs |
| Paneles                 | CPU y memoria de los contenedores                                       | Cualquier panel que construyas en Grafana                 |
| Alertas                 | [Alertas](/es/guide/alerts-and-webhooks) de logs, métricas y eventos    | Alertas de Grafana sobre consultas LogQL                  |
| Logs de fuera de Docker | No                                                                      | Sí: archivos, journal de systemd, Kubernetes y más        |

## Cuándo usar Loki

Elige Loki cuando necesitas los logs después de que Docker los haya rotado, cuando se elimina un contenedor y sus logs deben quedarse, o cuando necesitas buscar en semanas de historial. También encaja mejor si ya usas Grafana y quieres los logs junto a tus métricas, o si necesitas logs de cosas que no son contenedores.

Ten en cuenta la instalación. Promtail, el recolector que usan la mayoría de las guías antiguas, llegó al final de su vida útil en marzo de 2026, así que las instalaciones nuevas usan Grafana Alloy o el driver de Docker de Loki.

## Cuándo encaja mejor Dozzle

Elige Dozzle cuando quieres ver qué está haciendo un contenedor ahora, o qué hizo en las últimas horas o días, sin montar un pipeline de logs. No hay almacenamiento que dimensionar, ni retención que ajustar, ni lenguaje de consultas que aprender. Arranca en segundos. Consulta [Primeros pasos](/es/guide/getting-started).

Cuánto historial puede mostrar Dozzle depende de la rotación de logs de Docker. Para conservar más, sube los límites en `/etc/docker/daemon.json`:

```json
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "50m",
    "max-file": "5"
  }
}
```

Los nuevos límites se aplican a los contenedores creados después de reiniciar Docker.

## Usar los dos

Muchas instalaciones usan Loki para el historial y Dozzle para la depuración del día a día. No se estorban: Dozzle lee de Docker, y Alloy lee de Docker. Si en cambio pasas los contenedores al driver de Docker de Loki, mantén activado el [registro dual](https://docs.docker.com/engine/logging/dual-logging/) de Docker (viene activado por defecto), para que `docker logs`, y por tanto Dozzle, puedan seguir leyéndolos. Docker guarda entonces esa copia en una caché cuyo tamaño fijan `cache-max-size` y `cache-max-file` (5 archivos de 20 MB por defecto), no el `max-size` de arriba.
