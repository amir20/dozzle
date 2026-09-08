---
title: Forward proxy
sourceHash: 37a5c3119fb2
---

# <Icon icon="mdi:swap-horizontal" inline /> Forward proxy

Dozzle se puede configurar para leer cabeceras de proxy poniendo `--auth-provider` en `forward-proxy`.

::: code-group

```sh [cli]
$ docker run -v /var/run/docker.sock:/var/run/docker.sock -v /path/to/dozzle/data:/data -p 8080:8080 amir20/dozzle --auth-provider forward-proxy
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /path/to/dozzle/data:/data
    ports:
      - 8080:8080
    environment:
      DOZZLE_AUTH_PROVIDER: forward-proxy
```

:::

Monta `/data` también aquí. Los ajustes por usuario se escriben en disco también en modo forward proxy, y sin el volumen se pierden cada vez que se recrea el contenedor.

En este modo, Dozzle espera las siguientes cabeceras:

- `Remote-User` para el nombre de usuario, por ejemplo `johndoe`
- `Remote-Email` para la dirección de correo del usuario. Ese correo también se usa para encontrar el [Gravatar](https://gravatar.com/) correcto.
- `Remote-Name` para un nombre visible como `John Doe`
- `Remote-Filter` para una lista de filtros permitidos al usuario, separados por comas.
- `Remote-Roles` para una lista de roles permitidos al usuario, separados por comas.

Además, puedes configurar una URL de cierre de sesión con:

```yaml
DOZZLE_AUTH_LOGOUT_URL: http://oauth2.example.ru/oauth2/sign_out
```

## Configurar Dozzle con Authelia

[Authelia](https://www.authelia.com/) es un servidor y portal de autenticación y autorización de código abierto que cubre la gestión de identidades y accesos. Configurar Authelia queda fuera del alcance de esta sección, pero la configuración se puede compartir como ejemplo para montar Dozzle con Authelia.

<details>
<summary>➡️ Haz clic para desplegar el ejemplo de Authelia</summary>

::: code-group

```yaml [docker-compose.yml]
networks:
  net:
    driver: bridge

services:
  authelia:
    image: authelia/authelia
    container_name: authelia
    volumes:
      - ./authelia:/config
    networks:
      - net
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.authelia.rule=Host(`authelia.example.com`)"
      - "traefik.http.routers.authelia.entrypoints=https"
      - "traefik.http.routers.authelia.tls=true"
      - "traefik.http.routers.authelia.tls.options=default"
      - "traefik.http.middlewares.authelia.forwardAuth.address=http://authelia:9091/api/authz/forward-auth"
      - "traefik.http.middlewares.authelia.forwardAuth.trustForwardHeader=true"
      - "traefik.http.middlewares.authelia.forwardAuth.authResponseHeaders=Remote-User,Remote-Groups,Remote-Name,Remote-Email"
    expose:
      - 9091
    restart: unless-stopped

  traefik:
    image: traefik:v3.5
    container_name: traefik
    volumes:
      - ./traefik:/etc/traefik
      - /var/run/docker.sock:/var/run/docker.sock
    networks:
      - net
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.api.rule=Host(`traefik.example.com`)"
      - "traefik.http.routers.api.entrypoints=https"
      - "traefik.http.routers.api.service=api@internal"
      - "traefik.http.routers.api.tls=true"
      - "traefik.http.routers.api.tls.options=default"
      - "traefik.http.routers.api.middlewares=authelia@docker"
    ports:
      - "80:80"
      - "443:443"
    command:
      - "--api"
      - "--providers.docker=true"
      - "--providers.docker.exposedByDefault=false"
      - "--providers.file.filename=/etc/traefik/certificates.yml"
      - "--entrypoints.http=true"
      - "--entrypoints.http.address=:80"
      - "--entrypoints.http.http.redirections.entrypoint.to=https"
      - "--entrypoints.http.http.redirections.entrypoint.scheme=https"
      - "--entrypoints.https=true"
      - "--entrypoints.https.address=:443"
      - "--log=true"
      - "--log.level=DEBUG"

  dozzle:
    image: amir20/dozzle:latest
    networks:
      - net
    environment:
      DOZZLE_AUTH_PROVIDER: forward-proxy
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - dozzle:/data
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.dozzle.rule=Host(`dozzle.example.com`)"
      - "traefik.http.routers.dozzle.entrypoints=https"
      - "traefik.http.routers.dozzle.tls=true"
      - "traefik.http.routers.dozzle.tls.options=default"
      - "traefik.http.routers.dozzle.middlewares=authelia@docker"
    expose:
      - 8080
    restart: unless-stopped

volumes:
  dozzle:
```

```yaml [configuration.yml]
###############################################################
#                   Authelia configuration                      #
###############################################################

server:
  address: tcp://0.0.0.0:9091

log:
  level: info

totp:
  issuer: authelia.com

identity_validation:
  reset_password:
    jwt_secret: a_very_important_secret

authentication_backend:
  file:
    path: /config/users_database.yml

access_control:
  default_policy: deny
  rules:
    - domain: traefik.example.com
      policy: one_factor
    - domain: dozzle.example.com
      policy: one_factor

session:
  secret: unsecure_session_secret
  cookies:
    - domain: example.com # Debe coincidir con el dominio raíz protegido que uses
      authelia_url: https://authelia.example.com
      default_redirection_url: https://public.example.com

regulation:
  max_retries: 3
  find_time: 120
  ban_time: 300

storage:
  encryption_key: you_must_generate_a_random_string_of_more_than_twenty_chars_and_configure_this
  local:
    path: /config/db.sqlite3

notifier:
  filesystem:
    filename: /config/notification.txt
```

:::

Se necesitan claves SSL válidas porque Authelia solo funciona con SSL.

Authelia envía la pertenencia a grupos en `Remote-Groups`, y Dozzle no lee esa cabecera por defecto. Para asociar los grupos de Authelia con los [roles](/es/guide/authentication/simple#asignar-roles-especificos-a-los-usuarios) de Dozzle, pon `DOZZLE_AUTH_HEADER_ROLES: Remote-Groups` en el servicio de Dozzle y nombra los grupos como los roles. Los alias con prefijo `dozzle_` existen para esto, así que un grupo llamado `dozzle_shell` concede el rol `shell` y los demás nombres de grupo se ignoran. Sin esa asociación, todo usuario autenticado recibe todos los roles.

</details>

## Configurar Dozzle con Cloudflare Zero Trust

Cloudflare Zero Trust es un servicio para dar acceso autenticado a software autoalojado. Esta sección explica cómo configurar Dozzle para usar Cloudflare Zero Trust como autenticación.

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      DOZZLE_AUTH_PROVIDER: forward-proxy
      DOZZLE_AUTH_HEADER_USER: Cf-Access-Authenticated-User-Email
      DOZZLE_AUTH_HEADER_EMAIL: Cf-Access-Authenticated-User-Email
      DOZZLE_AUTH_HEADER_NAME: Cf-Access-Authenticated-User-Email
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - dozzle:/data
    expose:
      - 8080
    restart: unless-stopped

volumes:
  dozzle:
```

`expose` mantiene el puerto 8080 fuera del host, así que la única entrada es a través del túnel. Publicarlo con `ports` permitiría a cualquiera en el host poner la cabecera `Cf-Access-Authenticated-User-Email` por su cuenta y saltarse Cloudflare por completo.

Después de arrancar el contenedor de Dozzle, configura la aplicación en el panel de Cloudflare Zero Trust siguiendo esta [guía](https://developers.cloudflare.com/cloudflare-one/applications/configure-apps/self-hosted-apps/).

## Configurar Dozzle con Pocket ID

> [!TIP]
> Dozzle ya habla OpenID Connect de forma nativa, así que puedes apuntar `--auth-oidc-issuer` directamente a Pocket ID y saltarte oauth2-proxy por completo. Consulta [Iniciar sesión con GitHub y OIDC](/es/guide/authentication/oauth#iniciar-sesion-con-oidc). La receta de abajo sigue aquí para quien ya use oauth2-proxy o quiera que el proxy proteja algo más que Dozzle.

Primero tienes que montar un contenedor que pase la autenticación OpenID Connect a través de tu proxy inverso.

Aquí tienes un ejemplo con [oauth2-proxy](https://github.com/oauth2-proxy/oauth2-proxy).

<details>
<summary>➡️ Haz clic para desplegar el ejemplo de oauth2-proxy</summary>

1. Crea un nuevo cliente OIDC en Pocket ID para Dozzle:
   - **Nombre:** `Dozzle`
   - **URLs de callback:** `https://dozzle.example.com/oauth2/callback`
   - **PKCE:** `Enabled`

   Copia los valores de **Client ID** y **Client Secret** para usarlos después.

2. Añade lo siguiente al compose de Dozzle que ya tengas:

   ```yml
   environment:
     DOZZLE_AUTH_PROVIDER: forward-proxy
     DOZZLE_AUTH_HEADER_USER: X-Forwarded-User
     DOZZLE_AUTH_HEADER_EMAIL: X-Forwarded-Email
     DOZZLE_AUTH_HEADER_NAME: X-Forwarded-Preferred-Username
   ```

   Comenta los puertos de Dozzle, ya que los vamos a redirigir a través del nuevo contenedor de autenticación.

   Este método no debería requerir cambios en la configuración de tu proxy inverso.

   ```yml
   # ports:
   #   - 8080:8080
   ```

3. Añade un nuevo servicio de contenedor oauth2-proxy al compose de Dozzle:

   ```yml
   services:
     # ...
     oauth2-proxy:
       image: quay.io/oauth2-proxy/oauth2-proxy:latest
       restart: unless-stopped
       container_name: dozzle-oidc
       command: --config /oauth2-proxy.cfg
       volumes:
         - "./oauth2-proxy.cfg:/oauth2-proxy.cfg"
       ports:
         - 8080:4180
   ```

4. Crea el archivo de configuración de oauth2-proxy.

   En el directorio donde está tu archivo compose, crea `oauth2-proxy.cfg`:

   ```toml
    client_id = "xxx"                            # de Pocket ID
    client_secret = "xxx"                        # de Pocket ID
    cookie_secret = "xxx"                        # genéralo con openssl rand -base64 32 | tr -- '+/' '-_'
    upstreams = "http://dozzle:8080"             # upstream al puerto interno del contenedor de Dozzle
    code_challenge_method = "S256"               # desafíos PKCE plain o S256
    cookie_expire = "0"                          # segundos, 0 para sesión
    cookie_name = "__Host-oauth2-proxy"          # o __Secure-oauth2-proxy (menos seguro)
    cookie_secure = true                         # usa la cookie segura HTTPS
    email_domains = ["*"]                        # permite autenticarse desde cualquier dominio de correo
    http_address = "0.0.0.0:4180"                # puerto en el que escucha oauth2-proxy
    oidc_issuer_url = "https://id.example.com"   # tu URL base de Pocket
    provider_display_name = "Pocket ID"          # nombre visible para el inicio de sesión OIDC
    provider = "oidc"                            # usa OpenID connect
    reverse_proxy = true                         # pasa el tráfico por el proxy inverso
    scope = "openid email profile groups"        # deja pasar estos scopes OIDC
   ```

   Rellena las variables según los comentarios.

5. Por último, reinicia tu stack de Docker compose.

   Tu proxy inverso ya debería autenticarte en Dozzle a través de oauth2-proxy.

   Revisa los logs para solucionar problemas.

</details>
