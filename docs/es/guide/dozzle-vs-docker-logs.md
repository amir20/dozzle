---
title: Dozzle vs docker logs
description: Qué aporta Dozzle frente a docker logs y las herramientas de terminal, y cuándo te basta con el terminal.
sourceHash: d312eefe8524
---

# Dozzle vs `docker logs`

Dozzle lee los mismos logs que `docker logs`. No se copia ni se guarda nada, así que nunca muestra nada que el comando no pudiera mostrar. Lo que cambia es cómo los lees: en un navegador, a través de contenedores y hosts, sin una shell en el servidor.

## De un vistazo

|                     | Dozzle                                                                                  | `docker logs`                                          |
| ------------------- | --------------------------------------------------------------------------------------- | ------------------------------------------------------ |
| Dónde los lees      | Un navegador, desde cualquier dispositivo                                               | Una shell en el host de Docker                         |
| Varios contenedores | Cualquier conjunto de contenedores en una vista combinada, o un stack o servicio entero | Un contenedor. `docker compose logs` cubre un proyecto |
| Varios hosts        | Una interfaz para todos los hosts, mediante [agentes](/es/guide/agent)                  | Un host por shell, o un `DOCKER_HOST` por comando      |
| Búsqueda            | Búsqueda mientras escribes, con expresiones regulares                                   | Pasarlo por `grep`                                     |
| Logs JSON           | Formateados y coloreados, [SQL](/es/guide/sql-engine) sobre los campos                  | Líneas en bruto, pasarlas por `jq`                     |
| Trazas de error     | Las entradas multilínea se agrupan                                                      | Una línea cada vez                                     |
| CPU y memoria       | Gráficos junto a los logs                                                               | `docker stats` en otra shell                           |
| Compartir           | Un enlace a un contenedor, protegido con [autenticación](/es/guide/authentication)      | Copiar y pegar                                         |
| Instalación         | Un contenedor                                                                           | Ya está ahí                                            |

## Cuándo te basta con el terminal

Si ya estás en el host y quieres las últimas cien líneas de un contenedor, `docker logs --tail 100 -f <name>` es lo más rápido. Para una vista en vivo de todos los contenedores en un terminal, [dtop](/es/guide/dtop) los muestra en una tabla al estilo de `docker ps` y abre Dozzle cuando necesitas los logs completos.

## Cuándo encaja mejor Dozzle

Usa Dozzle cuando más de una persona lee los logs, cuando prefieres no dar acceso SSH para leerlos, o cuando sigues una petición a través de varios contenedores a la vez. También es la forma más sencilla de leer logs desde el móvil.

## Pruébalo

```sh
docker run -d -v /var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle:latest
```

Después abre `http://localhost:8080`. Tienes más opciones en [Primeros pasos](/es/guide/getting-started).
