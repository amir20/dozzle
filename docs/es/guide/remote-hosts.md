---
title: Configuración de hosts remotos
sourceHash: 6ae09165c838
---

# Configuración de hosts remotos

<Badge type="warning" text="Solo Docker" />

Dozzle puede conectarse a hosts de Docker remotos. Esto resulta útil cuando ejecutas Dozzle en un contenedor y quieres monitorizar otro host de Docker.

Aun así, con los agentes de Dozzle puedes conectarte a hosts remotos sin exponer el socket de Docker. Consulta la página del [agente](/es/guide/agent) para más información.

Los agentes de Dozzle evitan tener que exponer el socket de Docker en remoto, pero no se pueden usar con un proxy del socket de Docker dentro del stack del propio agente. Si quieres usar un Socket Proxy por su cuenta, sin agente, consulta la sección [conectar con un socket proxy](#connecting-with-a-socket-proxy).

> [!WARNING]
> Los hosts remotos han quedado sustituidos por los agentes. Los agentes son una forma más segura de conectarse a hosts remotos. Aunque los hosts remotos siguen funcionando, se recomienda usar agentes. Consulta la página del [agente](/es/guide/agent) para más información y ejemplos. Para compararlos, mira la sección [comparación entre agentes y conexiones remotas](/es/guide/agent#comparing-agents-with-remote-connection). No voy a poder investigar los problemas de los usuarios con hosts remotos porque lleva muchísimo tiempo.

## Conectar a hosts remotos con TLS

Los hosts remotos se configuran con `--remote-host` o `DOZZLE_REMOTE_HOST`. Todos los certificados deben montarse en el directorio `/certs`. El directorio `/certs` espera encontrar `/certs/{ca,cert,key}.pem`, o `/certs/{host}/{ca,cert,key}.pem` si hay varios hosts.

Ten en cuenta que el valor `{host}` al que se refiere aquí es la IP o el FQDN configurado, no la [etiqueta opcional](#adding-labels-to-hosts).

Se puede repetir el flag `--remote-host` para indicar varios hosts. En cambio, con `DOZZLE_REMOTE_HOST` los valores van separados por comas.

::: code-group

```sh [cli]
$ docker run -v /var/run/docker.sock:/var/run/docker.sock -v /path/to/certs:/certs -p 8080:8080 amir20/dozzle --remote-host tcp://167.99.1.1:2376 --remote-host tcp://167.99.1.2:2376
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /path/to/certs:/certs
    ports:
      - 8080:8080
    environment:
      DOZZLE_REMOTE_HOST: tcp://167.99.1.1:2376,tcp://167.99.1.2:2376
```

:::

## Conectar con un socket proxy

Si estás en una red privada, puedes usar [Docker Socket Proxy](https://github.com/Tecnativa/docker-socket-proxy), que expone el archivo `docker.sock` sin necesidad de TLS. Así te ahorras el agente de Dozzle y Dozzle se conecta directamente al Socket Proxy. Dozzle nunca intenta escribir en Docker, pero necesita acceso a las API de listado. Este comando arranca un proxy con los permisos mínimos:

```sh
$ docker container run --privileged -e CONTAINERS=1 -e INFO=1 -v /var/run/docker.sock:/var/run/docker.sock -p 2375:2375 tecnativa/docker-socket-proxy
```

> [!TIP]
> `CONTAINERS=1` es necesario para listar los contenedores en ejecución. `EVENTS` también hace falta, pero viene activado por defecto. `INFO=1` es necesario para listar la información del sistema.

Ejecutar Dozzle sin certificados debería funcionar. Aquí tienes un ejemplo:

::: code-group

```sh [cli]
$ docker run -p 8080:8080 amir20/dozzle --remote-host tcp://123.1.1.1:2375
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    ports:
      - 8080:8080
    environment:
      DOZZLE_REMOTE_HOST: tcp://123.1.1.1:2375
```

:::

Cuando usas un host remoto, montar `/var/run/docker.sock` es opcional. Necesitas al menos un host remoto al que conectarte.

> [!WARNING]
> Docker Socket Proxy expone la API de Docker a internet. Puede ser un riesgo de seguridad si no se protege bien.

## Poner etiquetas a los hosts

`--remote-host` admite etiquetas de host añadiéndolas a la cadena de conexión con `|`. Por ejemplo, `--remote-host tcp://123.1.1.1:2375|foobar.com` usará foobar.com como etiqueta en la interfaz. Un ejemplo completo con la CLI o con Compose:

::: code-group

```sh
docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --remote-host tcp://123.1.1.1:2375|foobar.com
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /path/to/certs:/certs
    ports:
      - 8080:8080
    environment:
      DOZZLE_REMOTE_HOST: tcp://167.99.1.1:2376|foo.com,tcp://167.99.1.2:2376|bar.com
```

:::

> [!WARNING]
> Dozzle usa la API de Docker para recopilar información sobre los hosts. Cada agente necesita un ID de host único. Para identificar el host se usa el ID de sistema de Docker o el ID de nodo. Si usas Swarm, se usa el ID de nodo. Si no ves todos los hosts, puede que tengas hosts duplicados configurados con el mismo ID. Para arreglarlo, borra el archivo `/var/lib/docker/engine-id`. Consulta la [FAQ](/es/guide/faq#i-am-seeing-duplicate-hosts-error-in-the-logs-how-do-i-fix-it) para más información.

## Cambiar la etiqueta de localhost

`localhost` es una conexión especial y usa una configuración distinta a `--remote-host`. Para cambiar su etiqueta usa el flag `--hostname` o la variable de entorno `DOZZLE_HOSTNAME`. Consulta la página de [hostname](/es/guide/hostname) para ver ejemplos de uso.
