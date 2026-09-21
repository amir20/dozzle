---
title: Variables de entorno y subcomandos
sourceHash: 94da032e7db0
---

# Variables de entorno

Cada opción se puede configurar con un flag o con una variable de entorno. Los flags y las variables de entorno siempre tienen prioridad sobre los ajustes que el [asistente de configuración](/es/guide/setup-wizard) guarda en `dozzle.yml`.

Las opciones que aceptan una lista (`DOZZLE_FILTER`, `DOZZLE_REMOTE_HOST`, `DOZZLE_REMOTE_AGENT`, `DOZZLE_NAMESPACE`) admiten un valor separado por comas en la variable de entorno, o el flag repetido una vez por elemento:

```sh
--remote-agent 167.99.1.1:7007 --remote-agent 167.99.1.2:7007
DOZZLE_REMOTE_AGENT=167.99.1.1:7007,167.99.1.2:7007
```

## Servidor

| Variable                                  | Descripción                                                                                         | Valores                                                           | Por defecto |
| ----------------------------------------- | --------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------- | ----------- |
| `DOZZLE_ADDR`<br>`--addr`                 | Dirección y puerto en los que escucha el servidor web. Rara vez hace falta dentro de un contenedor. | `host:port`, p. ej. `:9090`                                       | `:8080`     |
| `DOZZLE_BASE`<br>`--base`                 | Prefijo de ruta bajo el que se sirve Dozzle. Consulta [cambiar la base](/es/guide/changing-base).   | una ruta, p. ej. `/logs`                                          | `/`         |
| `DOZZLE_HOSTNAME`<br>`--hostname`         | Nombre que muestra la interfaz para esta instancia. Consulta [hostname](/es/guide/hostname).        | cualquier texto                                                   | ninguno     |
| `DOZZLE_HOST_ID`<br>`--host-id`           | Sustituye el id que Dozzle deriva para este host. Solo hace falta cuando choca con el de otro host. | letras, dígitos, `_`, `.`, `-`                                    | derivado    |
| `DOZZLE_LEVEL`<br>`--level`               | Nivel de log del propio Dozzle. Consulta [depuración](/es/guide/debugging).                         | `trace`, `debug`, `info`, `warn`, `error`                         | `info`      |
| `DOZZLE_MODE`<br>`--mode`                 | Modo de despliegue.                                                                                 | `server`, [`swarm`](/es/guide/swarm-mode), [`k8s`](/es/guide/k8s) | `server`    |
| `DOZZLE_TIMEOUT`<br>`--timeout`           | Tiempo de espera de las llamadas a la API de Docker o Kubernetes.                                   | una duración, p. ej. `30s`                                        | `10s`       |
| `DOZZLE_NO_ANALYTICS`<br>`--no-analytics` | Desactiva las [analíticas](/es/guide/analytics) anónimas.                                           | `true`, `false`                                                   | `false`     |

## Contenedores y hosts

| Variable                                  | Descripción                                                                                                              | Valores                             | Por defecto       |
| ----------------------------------------- | ------------------------------------------------------------------------------------------------------------------------ | ----------------------------------- | ----------------- |
| `DOZZLE_FILTER`<br>`--filter`             | Muestra solo los contenedores que cumplen un filtro de Docker. Consulta [filtros](/es/guide/filters).                    | `key=value`, p. ej. `label=app=web` | ninguno           |
| `DOZZLE_REMOTE_AGENT`<br>`--remote-agent` | [Agentes](/es/guide/agent) a los que conectarse, con un nombre visible y un grupo opcionales.                            | `host:port[\|name[\|group]]`        | ninguno           |
| `DOZZLE_REMOTE_HOST`<br>`--remote-host`   | Hosts de Docker a los que conectarse por TCP. Consulta [hosts remotos](/es/guide/remote-hosts).                          | `tcp://host:port[\|label]`          | ninguno           |
| `DOZZLE_NAMESPACE`<br>`--namespace`       | Namespaces de Kubernetes que vigilar. Solo se usa con `DOZZLE_MODE=k8s`.                                                 | nombres de namespace                | todos             |
| `DOZZLE_CERT`<br>`--cert`                 | Certificado TLS para comunicarse con los agentes. Consulta [certificados propios](/es/guide/agent#certificados-propios). | una ruta de archivo                 | `dozzle_cert.pem` |
| `DOZZLE_KEY`<br>`--key`                   | Clave privada TLS para comunicarse con los agentes.                                                                      | una ruta de archivo                 | `dozzle_key.pem`  |

## Funciones

| Variable                                              | Descripción                                                                                                                                                                       | Valores                      | Por defecto                           |
| ----------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------- | ------------------------------------- |
| `DOZZLE_ENABLE_ACTIONS`<br>`--enable-actions`         | Permite iniciar, detener, reiniciar, eliminar y actualizar contenedores desde la interfaz. Consulta [acciones](/es/guide/actions).                                                | `true`, `false`              | `false`                               |
| `DOZZLE_ENABLE_SHELL`<br>`--enable-shell`             | Permite conectarse a los contenedores y abrir una shell en ellos desde la interfaz. Consulta [shell](/es/guide/shell).                                                            | `true`, `false`              | `false`                               |
| `DOZZLE_ENABLE_MCP`<br>`--enable-mcp`                 | Expone el endpoint [MCP](/es/guide/mcp) para clientes LLM.                                                                                                                        | `true`, `false`              | `false`                               |
| `DOZZLE_DISABLE_AVATARS`<br>`--disable-avatars`       | Oculta los avatares de usuario cuando la autenticación está activada.                                                                                                             | `true`, `false`              | `false`                               |
| `DOZZLE_RELEASE_CHECK_MODE`<br>`--release-check-mode` | Si Dozzle comprueba si hay nuevas versiones de sí mismo. `manual` solo comprueba cuando lo pides.                                                                                 | `automatic`, `manual`        | `automatic`                           |
| `DOZZLE_IMAGE_CHECK_MODE`<br>`--image-check-mode`     | Si Dozzle consulta los registros en busca de imágenes de contenedor más recientes. Consulta [comprobación de actualizaciones](/es/guide/actions#comprobacion-de-actualizaciones). | `automatic`, `manual`, `off` | igual que `DOZZLE_RELEASE_CHECK_MODE` |
| `DOZZLE_AUTO_UPDATE`<br>`--auto-update`               | Actualiza el propio contenedor de Dozzle de forma programada. `weekly` se ejecuta el domingo. Requiere `DOZZLE_ENABLE_ACTIONS`.                                                   | `off`, `daily`, `weekly`     | `off`                                 |
| `DOZZLE_AUTO_UPDATE_TIME`<br>`--auto-update-time`     | Hora del día a la que se ejecuta la actualización automática, en la hora local del servidor.                                                                                      | `HH:MM`, p. ej. `04:30`      | `03:00`                               |

## Autenticación

Consulta [autenticación](/es/guide/authentication) para ver cómo funcionan los proveedores.

| Variable                                        | Descripción                                                                                                                       | Valores                                                         | Por defecto |
| ----------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------- | ----------- |
| `DOZZLE_AUTH_PROVIDER`<br>`--auth-provider`     | Proveedor de autenticación que se usa. `github` y `google` son alias de `simple`.                                                 | `none`, `simple`, `oidc`, `forward-proxy`                       | `none`      |
| `DOZZLE_AUTH_TTL`<br>`--auth-ttl`               | Cuánto dura un inicio de sesión. `session` cierra la sesión al cerrar el navegador.                                               | `session` o una duración, p. ej. `48h` (unidades `s`, `m`, `h`) | `session`   |
| `DOZZLE_AUTH_LOGOUT_URL`<br>`--auth-logout-url` | Adónde se envía al usuario al cerrar sesión con `forward-proxy`. Con `oidc` sustituye el endpoint de cierre de sesión del emisor. | una URL                                                         | ninguno     |

### GitHub OAuth

Añade "Iniciar sesión con GitHub" a la autenticación `simple`. Consulta [OAuth](/es/guide/authentication/oauth).

| Variable                                                            | Descripción                              | Valores  | Por defecto |
| ------------------------------------------------------------------- | ---------------------------------------- | -------- | ----------- |
| `DOZZLE_AUTH_GITHUB_CLIENT_ID`<br>`--auth-github-client-id`         | Client id de la app OAuth de GitHub.     | un texto | ninguno     |
| `DOZZLE_AUTH_GITHUB_CLIENT_SECRET`<br>`--auth-github-client-secret` | Client secret de la app OAuth de GitHub. | un texto | ninguno     |

### OpenID Connect

Se usa con `DOZZLE_AUTH_PROVIDER=oidc`. Consulta [OIDC](/es/guide/authentication/oidc).

| Variable                                                        | Descripción                                                                                                                                                                                                      | Valores                               | Por defecto |
| --------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------- | ----------- |
| `DOZZLE_AUTH_OIDC_ISSUER`<br>`--auth-oidc-issuer`               | URL del emisor del proveedor de identidad.                                                                                                                                                                       | una URL                               | ninguno     |
| `DOZZLE_AUTH_OIDC_CLIENT_ID`<br>`--auth-oidc-client-id`         | Client id registrado en el proveedor.                                                                                                                                                                            | un texto                              | ninguno     |
| `DOZZLE_AUTH_OIDC_CLIENT_SECRET`<br>`--auth-oidc-client-secret` | Client secret registrado en el proveedor.                                                                                                                                                                        | un texto                              | ninguno     |
| `DOZZLE_AUTH_OIDC_NAME`<br>`--auth-oidc-name`                   | Etiqueta del botón de inicio de sesión.                                                                                                                                                                          | cualquier texto                       | `SSO`       |
| `DOZZLE_AUTH_OIDC_ROLES_CLAIM`<br>`--auth-oidc-roles-claim`     | Claim del que leer los roles. Solo hace falta cuando la búsqueda por defecto (`dozzle_roles`, `resource_access.<client-id>.roles`, `roles`) no lo encuentra.                                                     | una ruta de claim separada por puntos | ninguno     |
| `DOZZLE_AUTH_OIDC_FILTERS_CLAIM`<br>`--auth-oidc-filters-claim` | Claim del que leer los filtros de contenedores. Solo hace falta cuando la búsqueda por defecto (`dozzle_filters`, `resource_access.<client-id>.filters`, `filters`) no lo encuentra.                             | una ruta de claim separada por puntos | ninguno     |
| `DOZZLE_AUTH_OIDC_SCOPES`<br>`--auth-oidc-scopes`               | Scopes adicionales que solicitar además de `openid`, `profile` y `email`, para proveedores que solo entregan un claim [cuando se solicita su scope](/es/guide/authentication/oidc#solicitar-scopes-adicionales). | scopes separados por comas            | ninguno     |

> [!TIP]
> `DOZZLE_AUTH_GITHUB_CLIENT_SECRET` y `DOZZLE_AUTH_OIDC_CLIENT_SECRET` también aceptan una variante `_FILE` que indica el archivo del que leer el valor, pensada para usarse con [secretos de Docker](/es/guide/authentication/oauth#usar-secretos-de-docker-para-el-client-secret).

### Forward Proxy

Se usa con `DOZZLE_AUTH_PROVIDER=forward-proxy`. Cada variable indica la cabecera HTTP que establece el proxy. Consulta [forward proxy](/es/guide/authentication/forward-proxy).

| Variable                                              | Descripción                                         | Valores               | Por defecto     |
| ----------------------------------------------------- | --------------------------------------------------- | --------------------- | --------------- |
| `DOZZLE_AUTH_HEADER_USER`<br>`--auth-header-user`     | Cabecera con el nombre de usuario.                  | un nombre de cabecera | `Remote-User`   |
| `DOZZLE_AUTH_HEADER_EMAIL`<br>`--auth-header-email`   | Cabecera con el correo.                             | un nombre de cabecera | `Remote-Email`  |
| `DOZZLE_AUTH_HEADER_NAME`<br>`--auth-header-name`     | Cabecera con el nombre visible.                     | un nombre de cabecera | `Remote-Name`   |
| `DOZZLE_AUTH_HEADER_FILTER`<br>`--auth-header-filter` | Cabecera con el filtro de contenedores del usuario. | un nombre de cabecera | `Remote-Filter` |
| `DOZZLE_AUTH_HEADER_ROLES`<br>`--auth-header-roles`   | Cabecera con los roles del usuario.                 | un nombre de cabecera | `Remote-Roles`  |

## Subcomandos

### generate

Genera una entrada de `users.yml` para la [autenticación simple](/es/guide/authentication/simple). El nombre de usuario es el primer argumento.

```sh
docker run -it --rm amir20/dozzle generate admin --password password --email test@email.net --name "John Doe" --user-filter name=foo --user-roles shell > users.yml
```

| Flag               | Descripción                                                                             | Valores                                                                 |
| ------------------ | --------------------------------------------------------------------------------------- | ----------------------------------------------------------------------- |
| `--password`, `-p` | Contraseña del usuario.                                                                 | un texto                                                                |
| `--email`, `-e`    | Correo del usuario, usado para el avatar.                                               | un correo                                                               |
| `--name`, `-n`     | Nombre visible del usuario.                                                             | un texto                                                                |
| `--user-filter`    | Contenedores que el usuario puede ver.                                                  | filtros `key=value` separados por comas                                 |
| `--user-roles`     | Lo que el usuario puede hacer. Antepón `^` a un rol para quitarlo, p. ej. `all,^shell`. | `all`, `none`, `shell`, `actions`, `download`, `notifications`, `cloud` |

### agent

Ejecuta Dozzle como un [agente](/es/guide/agent) al que se conecta otra instancia de Dozzle.

```sh
docker run -v /var/run/docker.sock:/var/run/docker.sock -p 7007:7007 amir20/dozzle agent
```

| Variable                              | Descripción                                      | Valores     | Por defecto |
| ------------------------------------- | ------------------------------------------------ | ----------- | ----------- |
| `DOZZLE_AGENT_ADDR`<br>`--agent-addr` | Dirección y puerto en los que escucha el agente. | `host:port` | `:7007`     |

### generate-certs

Genera un certificado y una clave únicos para las conexiones de agentes, en lugar del compartido que incluye Dozzle. Consulta [certificados propios](/es/guide/agent#certificados-propios).

| Flag         | Descripción                          | Por defecto       |
| ------------ | ------------------------------------ | ----------------- |
| `--cert-out` | Dónde escribir el certificado.       | `dozzle_cert.pem` |
| `--key-out`  | Dónde escribir la clave privada.     | `dozzle_key.pem`  |
| `--force`    | Sobrescribe los archivos existentes. | `false`           |

### healthcheck

Comprueba que el servidor o el agente están activos. No viene configurado en la imagen por defecto porque añade un poco de consumo de CPU. Consulta [healthcheck](/es/guide/healthcheck).
