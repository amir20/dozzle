---
title: Autenticación
sourceHash: 4ce1bd2c3816
---

# Autenticación

Dozzle admite dos configuraciones de autenticación. En la primera, tú aportas tu propio método de autenticación protegiendo Dozzle detrás de un proxy. Dozzle puede leer las cabeceras correspondientes sin configuración adicional.

Si no tienes una solución de autenticación, Dozzle incluye una gestión de usuarios sencilla basada en archivos. Los proveedores de autenticación se configuran con el flag `--auth-provider`. En ambas configuraciones, Dozzle intentará guardar los ajustes de usuario en disco. Esos datos se escriben en `/data`.

## <Icon icon="mdi:shield-alert-outline" inline /> Consideraciones de seguridad

Dozzle tiene acceso a `docker.sock`, lo que, salvo que lo restrinjas, equivale a **root en el host**. Antes de exponer Dozzle fuera de tu red privada, revisa lo siguiente:

- **Pon siempre Dozzle detrás de autenticación** si es accesible desde internet. Usa `--auth-provider=simple` o un forward proxy como Authelia / Authentik / Cloudflare Access.
- **Mantén desactivadas las [acciones](/es/guide/actions) y el [acceso a la shell](/es/guide/shell)** salvo que los necesites. Permiten arrancar, parar, recrear y ejecutar comandos arbitrarios dentro de los contenedores.
- **Limita a los usuarios con [roles](#asignar-roles-especificos-a-los-usuarios) y [filtros](#asignar-filtros-especificos-a-los-usuarios)** en modo multiusuario. Sin roles explícitos, un usuario puede ver todos los contenedores que ve la instancia de Dozzle.
- **Nunca expongas el puerto de Dozzle directamente en modo forward proxy.** Dozzle confía en `Remote-User` en cada petición, y cuando no llega una cabecera de roles se le conceden todos los roles al usuario. Cualquiera que alcance el contenedor sin pasar por el proxy se autentica como quien quiera con solo poner una cabecera. Publica solo el proxy y deja Dozzle en una red interna usando `expose` en lugar de `ports`.
- **Termina el TLS en el proxy inverso**. Consulta [Proxy inverso y ruta base](/es/guide/changing-base) para ver ejemplos con Nginx / Traefik / Caddy.
- **Restringe el acceso a `docker.sock` con un proxy** si no necesitas las acciones. Ten en cuenta que montarlo en solo lectura (`/var/run/docker.sock:/var/run/docker.sock:ro`) _no_ limita la API: el flag `:ro` solo marca el archivo del socket como de solo lectura en disco, mientras que las llamadas a la API siguen pasando con normalidad, así que crear, borrar y actualizar siguen siendo posibles. Para restringir de verdad las operaciones, pon un proxy de socket como [`tecnativa/docker-socket-proxy`](https://github.com/Tecnativa/docker-socket-proxy) delante del daemon.

## <Icon icon="mdi:account-cog-outline" inline /> Gestión de usuarios basada en archivos

Dozzle admite autenticación multiusuario poniendo `--auth-provider` en `simple`. En este modo, Dozzle intentará leer el archivo de usuarios desde `/data/`, dando prioridad a `users.yml` sobre `users.yaml` si ambos existen. Si solo hay uno, se usará ese. El log indicará qué archivo se está leyendo (por ejemplo, `Reading users.yml file`).

### Rutas de ejemplo:

- `/data/users.yml`
- `/data/users.yaml`

El contenido del archivo tiene este aspecto:

```yaml
users:
  # "admin" aquí es el nombre de usuario
  admin:
    email: me@email.net
    name: Admin
    # Genéralo con docker run -it --rm amir20/dozzle generate admin --password password --email me@email.net --name "Admin"
    password: $2a$11$9ho4vY2LdJ/WBopFcsAS0uORC0x2vuFHQgT/yBqZyzclhHsoaIkzK
    filter:
    roles:
```

Dozzle usa `email` para generar avatares mediante [Gravatar](https://gravatar.com/). Es opcional. La contraseña se cifra con `bcrypt`, que se puede generar con `docker run amir20/dozzle generate`.

> [!WARNING]
> Los hashes de contraseña SHA-256 ya no son compatibles. Las versiones antiguas de Dozzle cifraban las contraseñas con SHA-256, y un `users.yml` que todavía contenga uno se cargará sin quejarse, pero el proceso terminará en cuanto ese usuario intente iniciar sesión. Regenera todas las contraseñas con `generate` antes de actualizar. Para más detalles, consulta [este aviso de seguridad](https://github.com/amir20/dozzle/security/advisories/GHSA-w7qr-q9fh-fj35).

Tendrás que montar este archivo para que Dozzle lo encuentre. Aquí tienes un ejemplo:

::: code-group

```sh [cli]
$ docker run -v /var/run/docker.sock:/var/run/docker.sock -v /path/to/dozzle/data:/data -p 8080:8080 amir20/dozzle --auth-provider simple
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
      DOZZLE_AUTH_PROVIDER: simple
```

```yaml [users.yml]
users:
  admin:
    email: me@email.net
    name: Admin
    password: $2a$11$9ho4vY2LdJ/WBopFcsAS0uORC0x2vuFHQgT/yBqZyzclhHsoaIkzK
```

:::

O usando secretos de Docker:

```yaml
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      - DOZZLE_AUTH_PROVIDER=simple
    secrets:
      - source: users
        target: /data/users.yml
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - dozzle:/data
secrets:
  users:
    file: users.yml
volumes:
  dozzle:
```

### Ampliar la duración de la cookie de autenticación

Por defecto, Dozzle usa cookies de sesión que caducan al cerrar el navegador. Puedes ampliar la duración de la cookie poniendo `--auth-ttl` a una duración. Aquí tienes un ejemplo:

::: code-group

```sh [cli]
$ docker run -v /var/run/docker.sock:/var/run/docker.sock -v /path/to/dozzle/data:/data -p 8080:8080 amir20/dozzle --auth-provider simple --auth-ttl 48h
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
      DOZZLE_AUTH_PROVIDER: simple
      DOZZLE_AUTH_TTL: 48h
```

:::

Ten en cuenta que solo se admiten duraciones. Solo puedes usar `s`, `m`, `h` para segundos, minutos y horas respectivamente.

### Asignar filtros específicos a los usuarios

Dozzle permite asignar filtros a los usuarios. Los filtros sirven para limitar los contenedores que un usuario puede ver. Se definen en el archivo `users.yml`. Aquí tienes un ejemplo:

```yaml
users:
  admin:
    email:
    name: Admin
    password: $2a$11$9ho4vY2LdJ/WBopFcsAS0uORC0x2vuFHQgT/yBqZyzclhHsoaIkzK
    filter:

  guest:
    email:
    name: Guest
    password: $2a$11$9ho4vY2LdJ/WBopFcsAS0uORC0x2vuFHQgT/yBqZyzclhHsoaIkzK
    filter: "label=com.example.app"
```

En este ejemplo, el usuario `admin` no tiene filtro, así que ve todos los contenedores. El usuario `guest` solo ve los contenedores con la etiqueta `com.example.app`. Esto es útil para restringir el acceso a contenedores concretos.

> [!NOTE]
> Los filtros también se pueden definir [globalmente](/es/guide/filters) con el flag `--filter`. Ese flag se aplica a todos los usuarios. Si un usuario tiene un filtro definido, este tiene prioridad sobre el global.

### Asignar roles específicos a los usuarios

Dozzle permite asignar roles a los usuarios. Los roles definen qué acciones puede realizar un usuario sobre los contenedores. Los roles se configuran en el archivo users.yml.

```yaml
users:
  admin:
    email:
    name: Admin
    password: $2a$11$9ho4vY2LdJ/WBopFcsAS0uORC0x2vuFHQgT/yBqZyzclhHsoaIkzK
    roles:

  guest:
    email:
    name: Guest
    password: $2a$11$9ho4vY2LdJ/WBopFcsAS0uORC0x2vuFHQgT/yBqZyzclhHsoaIkzK
    roles: shell
```

En este ejemplo, el usuario `admin` no tiene roles indicados, así que tiene acceso completo a todas las acciones sobre contenedores. El usuario `guest` tiene el rol shell, es decir, solo puede abrir una shell en los contenedores. Los roles facilitan controlar y limitar lo que los usuarios pueden hacer en Dozzle.

Dozzle admite los siguientes roles:

| Rol             | También aceptado       | Permite                                                                                             |
| --------------- | ---------------------- | --------------------------------------------------------------------------------------------------- |
| `shell`         | `dozzle_shell`         | Conectarse a un contenedor y abrir una sesión exec. La instancia también necesita `--enable-shell`. |
| `actions`       | `dozzle_actions`       | Arrancar, parar y reiniciar contenedores. La instancia también necesita `--enable-actions`.         |
| `download`      | `dozzle_download`      | Descargar los logs de un contenedor como archivo.                                                   |
| `notifications` | `dozzle_notifications` | Crear y editar reglas y destinos de notificación.                                                   |
| `cloud`         | `dozzle_cloud`         | Vincular, desvincular y configurar Dozzle Cloud.                                                    |
| `all`           | `dozzle_all`           | Todos los roles anteriores. Es el valor por defecto cuando `roles` está vacío.                      |
| `none`          | `dozzle_none`          | Ningún rol. Los logs siguen siendo visibles, según el filtro del usuario. Anula todo lo demás.      |

Los roles se separan con comas o barras verticales (`shell,actions` o `shell|actions`), y también sirve un array JSON (`["shell", "actions"]`). Los nombres no distinguen mayúsculas de minúsculas. Los alias con prefijo `dozzle_` existen para que los nombres de grupo de un proveedor de identidad se puedan pasar tal cual en modo forward proxy.

> [!WARNING]
> Las reglas de notificación son de toda la instancia. Una regla selecciona contenedores por expresión, no por el filtro del usuario, así que un usuario con el rol `notifications` puede crear una regla para contenedores que su filtro oculta y recibir esas líneas de log en un destino que él controla. Concédelo solo a usuarios en los que confíes con todos los contenedores de la instancia.

> [!WARNING]
> Dozzle Cloud también es de toda la instancia. Vincularlo guarda una única clave de API que redirige el envío de alertas, el streaming de logs y la ejecución de herramientas a una sola cuenta de la nube, y las herramientas de la nube se ejecutan con el filtro de la instancia, no con el del usuario que la vinculó. Un usuario con el rol `cloud` puede vincular la instancia a su propia cuenta de la nube y ver todos los contenedores a través de ella, o desvincular una conexión existente. Concédelo solo a usuarios en los que confíes con todos los contenedores de la instancia.

Cualquier rol puede llevar el prefijo `^` para excluirlo. Las exclusiones se aplican al final, así que el orden no importa:

```yaml
roles: all,^shell # todo excepto shell
```

`none` es el único rol que no se puede negar. `^none` se ignora, y un `none` a secas en cualquier punto de la lista descarta todos los demás roles.

## <Icon icon="mdi:file-document-edit-outline" inline /> Generar users.yml

Dozzle incluye un comando `generate` para generar `users.yml`. Aquí tienes un ejemplo:

```sh
docker run -it --rm amir20/dozzle generate admin --password password --email test@email.net --name "John Doe" --user-filter name=foo --user-roles shell > users.yml
```

En este ejemplo, `admin` es el nombre de usuario. El correo y el nombre son opcionales, pero se recomiendan para mostrar avatares correctos. `docker run -it --rm amir20/dozzle generate --help` muestra todas las opciones. El flag `--user-filter` es una lista de filtros separados por comas. El flag `--user-roles` es una lista de roles separados por comas.

Si omites `--password`, Dozzle la pide por stdin para que la contraseña no acabe en el historial de tu shell. Esto requiere un terminal interactivo, así que mantén los flags `-it`:

```sh
docker run -it --rm amir20/dozzle generate admin --email test@email.net --name "John Doe" > users.yml
```

La petición se escribe en stderr, así que redirigir stdout a `users.yml` sigue funcionando. También puedes pasar la contraseña por tubería, por ejemplo `echo "$PASSWORD" | docker run -i --rm amir20/dozzle generate admin > users.yml`.

## <Icon icon="mdi:swap-horizontal" inline /> Forward proxy

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

### Configurar Dozzle con Authelia

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

Authelia envía la pertenencia a grupos en `Remote-Groups`, y Dozzle no lee esa cabecera por defecto. Para asociar los grupos de Authelia con los [roles](#asignar-roles-especificos-a-los-usuarios) de Dozzle, pon `DOZZLE_AUTH_HEADER_ROLES: Remote-Groups` en el servicio de Dozzle y nombra los grupos como los roles. Los alias con prefijo `dozzle_` existen para esto, así que un grupo llamado `dozzle_shell` concede el rol `shell` y los demás nombres de grupo se ignoran. Sin esa asociación, todo usuario autenticado recibe todos los roles.

</details>

### Configurar Dozzle con Cloudflare Zero Trust

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

### Configurar Dozzle con Pocket ID

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
