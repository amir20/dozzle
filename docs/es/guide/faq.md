---
title: Preguntas frecuentes
sourceHash: c964c824dcab
---

# Preguntas frecuentes

## Dozzle no arranca y muestra `client version 1.x is too new`. ¿Qué significa?

Dozzle necesita Docker Engine 19.03 o posterior (API 1.40+). Los daemons más antiguos, por ejemplo Docker 18.06 (API 1.38), no son compatibles con el SDK de Docker que usa Dozzle y fallan al arrancar con un error como `failed to create docker client: ... client version 1.54 is too new. Maximum supported API version is 1.38`.

Actualiza Docker Engine a una versión compatible. Como solución temporal, fija Dozzle en `v10.5.2` o anterior, que usaba un SDK de Docker que todavía negociaba versiones antiguas de la API.

## ¿Cómo actualizo Dozzle?

Dozzle sigue las prácticas habituales de las imágenes de Docker. Para actualizar, descarga la nueva imagen y recrea el contenedor:

```sh
docker pull amir20/dozzle:latest
docker compose up -d dozzle
```

Los ajustes de usuario, las reglas de notificación y el resto del estado se guardan en `/data` (más abajo hay detalles), así que mantén ese volumen montado entre actualizaciones. En producción, fija una etiqueta concreta (por ejemplo `amir20/dozzle:v10.9.2`) en lugar de `latest`, para que las actualizaciones sean deliberadas. Las notas de versión se publican en la [página de releases de GitHub](https://github.com/amir20/dozzle/releases). Volver atrás es tan sencillo como desplegar de nuevo una etiqueta anterior.

## Mi plataforma envuelve el entrypoint del contenedor y Dozzle falla con `no such file or directory`

La imagen por defecto se construye `FROM scratch`, así que contiene el binario de Dozzle y nada más. Sin shell, sin intérprete.

Algunas plataformas añaden funciones opcionales montando un wrapper `#!/bin/sh` sobre el entrypoint del contenedor y volviendo a ejecutar el original. El interruptor de Tailscale por contenedor de Unraid funciona así, igual que algunos inyectores de sidecars e init. Sin `/bin/sh` en la imagen, el wrapper no se puede ejecutar y el contenedor termina con un error que nombra al wrapper y no a la shell que falta:

```
exec /opt/unraid/tailscale: no such file or directory
```

Para estos casos usa la variante `alpine`, que es el mismo binario sobre una base Alpine:

```sh
docker run \
  --volume=/var/run/docker.sock:/var/run/docker.sock \
  -p 8080:8080 \
  amir20/dozzle:alpine
```

Las etiquetas con versión siguen el mismo patrón (`amir20/dozzle:v10.9.2-alpine`). La imagen `latest` basada en scratch sigue siendo la recomendada para todo lo demás, porque ocupa mucho menos y no tiene paquetes de distribución que parchear.

## ¿Qué se guarda en `/data` y cómo hago una copia de seguridad?

El directorio `/data` es donde Dozzle guarda todo lo que debe sobrevivir a un reinicio del contenedor:

- `users.yml` / `users.yaml`, el archivo de usuarios de la autenticación simple (si lo has creado)
- Reglas de notificación, destinos y estado de envío
- Ajustes de interfaz por usuario (solo en modo multiusuario; en modo de un solo usuario los ajustes viven en el localStorage del navegador)
- Un pequeño conjunto de archivos internos, como el estado de los anuncios descartados

El directorio es pequeño (normalmente bastante menos de 10 MB) y se puede respaldar con un simple `tar` o `rsync` del volumen montado. Al actualizar o migrar a un host nuevo, mover el volumen `/data` lleva consigo todos los ajustes.

## Instalé Dozzle, pero los logs van lentos o no cargan nunca. ¿Qué hago?

Dozzle usa Server Sent Events (SSE), que se conecta al servidor mediante un stream HTTP sin cerrar la conexión. Si algún proxy intenta almacenar esa conexión en un búfer, Dozzle nunca recibe los datos y se queda esperando eternamente a que el proxy inverso vacíe el búfer. Desde la versión `1.23.0`, Dozzle envía la cabecera `X-Accel-Buffering: no`, que debería evitar que los proxies inversos usen búfer. Aun así, algunos proxies ignoran esa cabecera. En esos casos tienes que desactivar el búfer explícitamente.

### Desactivar el búfer en nginx

Aquí tienes un ejemplo con nginx usando `proxy_pass` para desactivar el búfer:

```
server {
    ...

    location / {
        proxy_pass                  http://<dozzle.container.ip.address>:8080;
    }

    location /api {
        proxy_pass                  http://<dozzle.container.ip.address>:8080;

        proxy_buffering             off;
        proxy_cache                 off;
    }
}
```

### Desactivar la compresión en traefik

El proxy inverso Traefik se puede configurar con middlewares para admitir compresión. Si lo tienes activado, la configuración habitual es esta:

```
http:
  middlewares:
    middlewares-compress:
      compress: {}
```

Con esta configuración puede que ciertos contenedores dejen de mostrar logs en dozzle si abres dozzle a través de traefik (por ejemplo, dozzle.mydomain.com).
También notarás que esa misma instancia de dozzle sí muestra los logs si accedes directamente (por ejemplo, localhost:8080).

Los contenedores donde se ha observado esto (lista no exhaustiva) son: dozzle, homepage, glances, filebrowser.

Para que los logs vuelvan a fluir, excluye `text/event-stream` del middleware de compresión:

```
http:
  middlewares:
    middlewares-compress:
      compress:
        excludedContentTypes:
          - text/event-stream
```

## Tenemos herramientas que usan Dozzle cuando se crea un contenedor. ¿Cómo consigo un enlace directo a un contenedor por nombre?

Dozzle tiene una [ruta](https://github.com/amir20/dozzle/blob/master/assets/pages/show.vue) especial que sirve para buscar contenedores por nombre y redirigir a ese contenedor. Por ejemplo, si tienes un contenedor con nombre `"foo.bar"` e id `abc123`, puedes enviar a tus usuarios a `/show?name=foo.bar`, que redirigirá a `/container/abc123`.

## Instalé Dozzle, pero no aparece el consumo de memoria

_Esto es un problema específico de los dispositivos ARM._

Dozzle usa la API de Docker para obtener información sobre el uso de memoria de los contenedores. Si el uso de memoria no aparece, lo más probable es que la API de Docker no lo esté devolviendo.

Puedes comprobarlo ejecutando docker info, donde deberías ver lo siguiente:

```
WARNING: No memory limit support
WARNING: No swap limit support
```

En ese caso, tendrás que añadir la siguiente línea a tu archivo `/boot/cmdline.txt` y reiniciar el dispositivo:

```
cgroup_enable=cpuset cgroup_enable=memory cgroup_memory=1
```

## Veo un error de hosts duplicados en los logs. ¿Cómo lo soluciono?

Si ves el siguiente error en los logs, puede que tengas hosts duplicados configurados con el mismo ID de host:

```
time="2024-07-10T13:35:53Z" level=warning msg="duplicate host ID: *********, Endpoint: 1.1.1.1:7007 found, skipping"
```

Dozzle usa la API de Docker para obtener información sobre los hosts. Cada host debe tener un ID único. Ese ID identifica al host en la interfaz. En modo swarm, Dozzle usa el ID de nodo de `docker system info` para identificar el host. Si no usas modo swarm, Dozzle usará el ID de sistema de `docker system info` como ID de host.

A veces se restauran máquinas virtuales desde copias de seguridad con el mismo ID de host. Eso hace que Dozzle crea que el host ya está presente y lo omita de la lista de hosts. Para solucionarlo, elimina el archivo `/var/lib/docker/engine-id`. Ese archivo contiene el ID de host y se crea al arrancar el daemon de Docker.

## Veo un error de host no encontrado en los logs. ¿Cómo lo soluciono?

Este error debería darse casi siempre solo con Podman: Podman no crea un engine-id como hace Docker.
Si usas Docker, comprueba que el archivo `engine-id` existe en `/var/lib/docker` con los permisos correctos y contiene el UUID.

Para resolver el error sigue estos pasos:

1. Crea las carpetas: `mkdir -p /var/lib/docker`
2. Instala uuidgen si hace falta
3. Genera un UUID con uuidgen: `uuidgen > engine-id`

El archivo engine-id ya debería contener un UUID.

Puedes encontrar un ejemplo de configuración para Ansible en [Podman](/es/guide/podman)

Puede que necesites limpiar tu despliegue actual de Dozzle bajo Podman: para el contenedor y elimina los datos asociados (contenedor y volúmenes). Después puedes volver a desplegar el contenedor de Dozzle y los logs deberían aparecer.

## ¿Por qué solo veo contenedores en ejecución? ¿Cómo veo los contenedores parados?

Por defecto, Dozzle solo muestra los contenedores en ejecución. Para ver los contenedores parados tienes que activar la opción `Show Stopped Containers` en la configuración. Esta opción está desactivada por defecto para reducir el número de contenedores que se muestran en la interfaz.

## ¿Hay alguna forma de sincronizar mis ajustes entre varias instancias de Dozzle?

En modo de un solo usuario, Dozzle guarda los ajustes en el almacenamiento local del navegador. Eso significa que los ajustes solo están disponibles en el navegador donde se configuraron. Para que Dozzle pueda sincronizar ajustes entre varias instancias, necesita saber quién es el usuario. En modo multiusuario, Dozzle usa el nombre de usuario para guardar los ajustes en disco y sincronizarlos entre varias instancias. Esa información se guarda en el directorio `/data`. Si quieres sincronizar los ajustes entre varias instancias, tienes que [activar](/es/guide/authentication) el modo multiusuario e indicar un nombre de usuario.

## ¿Por qué Dozzle no admite notificaciones directas a Slack, Discord, Telegram, correo, etc.?

Por diseño, a Dozzle le da igual a dónde van tus alertas. En lugar de incluir integraciones para plataformas de notificación concretas, Dozzle ofrece **webhooks** con plantillas de payload personalizables. Así puedes enviar alertas a _cualquier_ servicio que acepte peticiones HTTP: Slack, Discord, Telegram, ntfy, PagerDuty, Opsgenie o tus propias herramientas internas, sin esperar a que Dozzle añada soporte explícito.

Hay varias razones para este enfoque:

- **Universalidad.** Los webhooks funcionan prácticamente con cualquier plataforma de notificaciones. Añadir integraciones específicas cubriría solo una fracción de lo que la gente necesita, mientras que los webhooks las cubren todas.
- **Mantenimiento.** Cada integración trae sus propias rarezas de API, flujos de autenticación, límites de peticiones y cambios incompatibles. Darles soporte convertiría a quienes mantienen Dozzle en responsables de depurar problemas de servicios de terceros, algo fuera del alcance de un visor de logs.
- **Simplicidad.** Dozzle es una herramienta ligera y centrada en ver logs de Docker. Mantener genérica la capa de notificaciones mantiene el código pequeño y el proyecto sostenible.

Si necesitas una experiencia más cerrada con integraciones más ricas (por ejemplo, notificaciones push web o botones de acción de ntfy), [Dozzle Cloud](/es/guide/dozzle-cloud) está pensado para eso.

Para configurar webhooks con el servicio que prefieras, consulta la guía de [Alertas y webhooks](/es/guide/alerts-and-webhooks); incluye plantillas de payload integradas para Slack, Discord y ntfy que puedes usar tal cual o personalizar.

## ¿Por qué Dozzle mantiene dockerd y containerd con la CPU algo elevada aunque no haya ningún navegador conectado?

Dozzle mantiene el streaming de estadísticas de los contenedores hasta 6 horas (2 horas en Kubernetes) después de que se desconecte el último navegador, y luego apaga el recolector de estadísticas por su cuenta. Es intencionado. Las estadísticas se transmiten de forma continua para que, al volver a abrir la interfaz, veas el histórico de CPU y memoria en lugar de un gráfico vacío. Si el streaming se detuviera al cerrar la pestaña, no habría histórico que mostrar.

El coste es una cantidad de CPU pequeña y constante en dockerd y containerd, ya que la API de estadísticas de Docker se basa en sondeo. Reiniciar el contenedor de Dozzle reinicia el temporizador de inmediato, y por eso un reinicio devuelve el host al reposo. Esto no es configurable por diseño. Un tiempo de espera corto rompería otras funciones que asumen que las estadísticas siguen llegando, así que reducirlo anularía el sentido del histórico de estadísticas.

## Mis instancias de Dozzle dan timeout en modo Swarm o no veo todos mis nodos de Swarm detrás de un balanceador de carga. ¿Cómo lo soluciono?

En modo Swarm, las instancias de Dozzle pueden necesitar su propia red overlay. Si ves un comportamiento inconsistente al conectarte a distintos nodos de Dozzle, plantéate añadir una red overlay separada que contenga solo las instancias de Dozzle, como se muestra abajo:

```
services:
  logs:
    ...
    networks: [ traefik, dozzle ]
    ...

networks:
  dozzle:
    driver: overlay
  traefik:
    external: true
```

La red externa `traefik` es la red overlay que usa el balanceador de carga para descubrir servicios, y hemos creado una nueva red overlay `dozzle` para que los nodos de Dozzle se comuniquen entre sí.
