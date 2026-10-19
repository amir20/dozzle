---
title: Dozzle vs Portainer
description: En qué se diferencian Dozzle y Portainer para leer logs de Docker y cómo usarlos a la vez.
sourceHash: d147e79555a6
---

# Dozzle vs Portainer

Portainer está pensado para gestionar contenedores: stacks, volúmenes, redes, imágenes y usuarios. Dozzle está pensado para leer sus logs. Portainer tiene un visor de logs y Dozzle puede arrancar, detener y actualizar contenedores, pero cada uno se centra en un trabajo distinto. Mucha gente usa los dos.

## De un vistazo

|                             | Dozzle                                                                                                           | Portainer                                                            |
| --------------------------- | ---------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------- |
| Función principal           | Leer logs                                                                                                        | Gestionar contenedores, stacks, volúmenes, redes e imágenes          |
| Logs en vivo                | Se transmiten a medida que llegan las líneas                                                                     | Actualización automática                                             |
| Varios contenedores         | Un stack, un servicio o un grupo personalizado en una vista combinada                                            | Un contenedor cada vez                                               |
| Logs JSON                   | Detectados, formateados y coloreados. [SQL](/es/guide/sql-engine) sobre campos JSON                              | Se muestran como texto                                               |
| Búsqueda                    | Búsqueda y filtrado, incluidas expresiones regulares                                                             | Búsqueda y filtrado por texto                                        |
| CPU y memoria               | Gráficos en vivo con historial reciente                                                                          | Estadísticas en vivo                                                 |
| Alertas                     | Sobre líneas de log, CPU o memoria, y eventos de contenedores. Consulta [Alertas](/es/guide/alerts-and-webhooks) | No en el visor de logs                                               |
| Actualizaciones de imágenes | Comprueba si hay imágenes nuevas y actualiza de forma programada. Consulta [Actualizaciones](/es/guide/updates)  | Descargar y volver a desplegar a mano                                |
| Varios hosts                | [Agentes](/es/guide/agent), [Swarm](/es/guide/swarm-mode), [Kubernetes](/es/guide/k8s)                           | Entornos                                                             |
| Instalación                 | Un contenedor, más un asistente de configuración opcional en el primer arranque                                  | Un contenedor, más una cuenta de administrador en el primer arranque |

## Cuándo usar Portainer

Elige Portainer si quieres crear y editar stacks desde el navegador, gestionar volúmenes, redes e imágenes, o dar a un equipo acceso a unos entornos y no a otros. Dozzle no intenta hacer nada de eso.

## Cuándo encaja mejor Dozzle

Elige Dozzle cuando la pregunta es «¿qué está haciendo este contenedor ahora mismo?». Abre directamente en los logs, sigue varios contenedores a la vez y hace legible el JSON sin ninguna configuración. Además es de solo lectura por defecto: las [acciones](/es/guide/actions) y el [acceso a la shell](/es/guide/shell) están desactivados hasta que los activas.

## Usar los dos

No entran en conflicto. Los dos leen el socket de Docker y funcionan en puertos distintos:

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - dozzle_data:/data
    ports:
      - 8080:8080
  portainer:
    image: portainer/portainer-ce:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - portainer-data:/data
    ports:
      - 9443:9443
volumes:
  dozzle_data:
  portainer-data:
```

Para saltar de Portainer a los logs de un contenedor en Dozzle, consulta [enlazar a un contenedor por nombre](/es/guide/faq#tenemos-herramientas-que-usan-dozzle-cuando-se-crea-un-contenedor-¿como-consigo-un-enlace-directo-a-un-contenedor-por-nombre).
