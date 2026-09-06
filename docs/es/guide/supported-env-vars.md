---
title: Variables de entorno y subcomandos
sourceHash: 3930d18cbbb4
---

# Variables de entorno globales

La configuración se puede hacer con flags o con variables de entorno. La tabla siguiente recoge todas las opciones admitidas y sus variables de entorno correspondientes.

| Flag                   | Variable de entorno         | Por defecto       |
| ---------------------- | --------------------------- | ----------------- |
| `--addr`               | `DOZZLE_ADDR`               | `:8080`           |
| `--base`               | `DOZZLE_BASE`               | `/`               |
| `--hostname`           | `DOZZLE_HOSTNAME`           | `""`              |
| `--level`              | `DOZZLE_LEVEL`              | `info`            |
| `--auth-provider`      | `DOZZLE_AUTH_PROVIDER`      | `none`            |
| `--auth-header-user`   | `DOZZLE_AUTH_HEADER_USER`   | `Remote-User`     |
| `--auth-header-email`  | `DOZZLE_AUTH_HEADER_EMAIL`  | `Remote-Email`    |
| `--auth-header-name`   | `DOZZLE_AUTH_HEADER_NAME`   | `Remote-Name`     |
| `--auth-header-filter` | `DOZZLE_AUTH_HEADER_FILTER` | `Remote-Filter`   |
| `--auth-header-roles`  | `DOZZLE_AUTH_HEADER_ROLES`  | `Remote-Roles`    |
| `--auth-logout-url`    | `DOZZLE_AUTH_LOGOUT_URL`    | `""`              |
| `--auth-ttl`           | `DOZZLE_AUTH_TTL`           | `session`         |
| `--enable-actions`     | `DOZZLE_ENABLE_ACTIONS`     | `false`           |
| `--enable-shell`       | `DOZZLE_ENABLE_SHELL`       | `false`           |
| `--enable-mcp`         | `DOZZLE_ENABLE_MCP`         | `false`           |
| `--disable-avatars`    | `DOZZLE_DISABLE_AVATARS`    | `false`           |
| `--filter`             | `DOZZLE_FILTER`             | `""`              |
| `--no-analytics`       | `DOZZLE_NO_ANALYTICS`       | `false`           |
| `--mode`               | `DOZZLE_MODE`               | `server`          |
| `--release-check-mode` | `DOZZLE_RELEASE_CHECK_MODE` | `automatic`       |
| `--image-check-mode`   | `DOZZLE_IMAGE_CHECK_MODE`   | heredado          |
| `--remote-host`        | `DOZZLE_REMOTE_HOST`        |                   |
| `--remote-agent`       | `DOZZLE_REMOTE_AGENT`       |                   |
| `--timeout`            | `DOZZLE_TIMEOUT`            | `10s`             |
| `--namespace`          | `DOZZLE_NAMESPACE`          | `""`              |
| `--cert`               | `DOZZLE_CERT`               | `dozzle_cert.pem` |
| `--key`                | `DOZZLE_KEY`                | `dozzle_key.pem`  |

> [!TIP]
> Algunos flags como `--remote-host` o `--remote-agent` se pueden repetir. Por ejemplo, `--remote-agent 167.99.1.1:7007 --remote-agent 167.99.1.2:7007` o separados por comas con `DOZZLE_REMOTE_AGENT=167.99.1.1:7007,167.99.1.2:7007`.

## Generar users.yml

Dozzle puede generar el archivo `users.yml`. Ese archivo sirve para autenticar usuarios. Aquí tienes un ejemplo:

```sh
docker run -it --rm amir20/dozzle generate admin --password password --email test@email.net --name "John Doe" --user-filter name=foo --user-roles shell > users.yml
```

En este ejemplo, `admin` es el nombre de usuario. El correo y el nombre son opcionales, pero conviene ponerlos para que los avatares se muestren bien. Con `docker run amir20/dozzle generate --help` puedes ver todas las opciones.

| Flag            | Descripción            | Por defecto |
| --------------- | ---------------------- | ----------- |
| `--password`    | Contraseña del usuario |             |
| `--email`       | Correo del usuario     |             |
| `--name`        | Nombre completo        |             |
| `--user-filter` | Filtros del usuario    |             |
| `--user-roles`  | Roles del usuario      |             |

Consulta [autenticación](/es/guide/authentication) para más información.

## Modo agente

Dozzle puede ejecutarse en modo agente. El modo agente es útil cuando ejecutas Dozzle en un host remoto y quieres monitorizar otro host de Docker. Se activa con el flag `--remote-agent`. Aquí tienes un ejemplo:

```sh
docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --remote-agent remote-ip:7007
```

| Flag     | Variable de entorno | Por defecto |
| -------- | ------------------- | ----------- |
| `--addr` | `DOZZLE_AGENT_ADDR` | `:7007`     |

Consulta [agente](/es/guide/agent) para más información.

## Healthcheck

Dozzle admite comprobaciones de salud con el comando `dozzle healthcheck`. No está activado por defecto porque añade consumo de CPU. Para usar `healthcheck` tienes que configurarlo.

Consulta [healthcheck](/es/guide/healthcheck) para más información.
