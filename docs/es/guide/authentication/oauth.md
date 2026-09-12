---
title: Iniciar sesión con GitHub y OIDC
sourceHash: cfb7126acb65
---

# <Icon icon="mdi:shield-account" inline /> Iniciar sesión con GitHub y OIDC

Dozzle puede permitir que los usuarios inicien sesión con una cuenta externa en lugar de escribir una contraseña. Esto forma parte del proveedor [`simple`](/es/guide/authentication/simple) y no es un proveedor aparte, así que `users.yml` se sigue leyendo en cada petición y sigue decidiendo quién entra.

Eso tiene una consecuencia que conviene dejar clara desde el principio: **`users.yml` es la lista de permitidos.** Una cuenta externa que no esté vinculada a ninguna entrada no puede iniciar sesión, y nunca se crea ninguna cuenta de forma automática.

El inicio de sesión con contraseña sigue funcionando en paralelo, algo que importa cuando una OAuth App se rompe y necesitas entrar para arreglarla.

> [!TIP]
> Si prefieres que el proveedor de identidad sea el dueño de la lista de usuarios, con roles y filtros leídos del token y sin ningún `users.yml`, eso es un proveedor aparte: consulta [OpenID Connect](/es/guide/authentication/oidc).

## Iniciar sesión con GitHub

Dozzle puede permitir que los usuarios inicien sesión con su cuenta de GitHub en lugar de escribir una contraseña. Esto forma parte del proveedor `simple` y no es un proveedor de autenticación aparte, así que `users.yml` se sigue leyendo en cada petición y sigue decidiendo quién entra. Continúa usando `--auth-provider simple`. También se acepta `github` como alias si prefieres dejar claro qué usa la instancia.

Primero crea una OAuth App en [Developer settings](https://github.com/settings/developers) de GitHub y pon la **Authorization callback URL** en:

```
https://your-dozzle-host/api/auth/callback
```

Si sirves Dozzle bajo una [ruta base](/es/guide/changing-base), inclúyela, por ejemplo `https://example.com/dozzle/api/auth/callback`. Dozzle no envía ningún `redirect_uri` al iniciar el flujo, así que GitHub siempre redirige a la URL de callback registrada en la OAuth App. Que no coincidan es el motivo más habitual de que falle el inicio de sesión.

Después copia el client ID, genera un client secret y pásale los dos a Dozzle:

::: code-group

```sh [cli]
$ docker run -v /var/run/docker.sock:/var/run/docker.sock -v /path/to/dozzle/data:/data -p 8080:8080 amir20/dozzle --auth-provider simple --auth-github-client-id Ov23liABCDEFGHIJKLMN --auth-github-client-secret 0123456789abcdef0123456789abcdef01234567
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
      DOZZLE_AUTH_GITHUB_CLIENT_ID: Ov23liABCDEFGHIJKLMN
      DOZZLE_AUTH_GITHUB_CLIENT_SECRET: 0123456789abcdef0123456789abcdef01234567
```

:::

Vincula un usuario con su cuenta de GitHub añadiendo una clave `github` en `users.yml`:

```yaml
users:
  admin:
    email: me@email.net
    name: Admin
    password: $2a$11$9ho4vY2LdJ/WBopFcsAS0uORC0x2vuFHQgT/yBqZyzclhHsoaIkzK
    github: octocat

  guest:
    email: guest@email.net
    name: Guest
    github: hubot
    filter: "label=com.example.app"
    roles: none
```

`password` es opcional cuando hay un `github` definido, como muestra `guest` arriba. `admin` tiene los dos, así que puede entrar de cualquiera de las dos formas. El inicio de sesión con contraseña sigue disponible como alternativa para quien todavía tenga una, y la página de inicio de sesión muestra ambas opciones.

> [!WARNING]
> Deja una contraseña en al menos una cuenta. Cuando ningún usuario de `users.yml` tiene `password`, el formulario de inicio de sesión desaparece por completo y el proveedor externo se convierte en la única forma de entrar, así que una URL de callback equivocada, una OAuth App revocada o un client secret caducado dejan a todo el mundo fuera de la interfaz web. Recuperar el acceso implica editar `users.yml` en el host para volver a poner una contraseña, lo que exige acceso por shell a donde viva el `/data` de Dozzle.

El valor es el **login** de GitHub (el identificador en `github.com/octocat`), no la dirección de correo. Un login siempre está presente y es visible, mientras que el correo de una cuenta puede ser privado o cambiar en cualquier momento.

> [!WARNING]
> Un login de GitHub no es permanente. Si alguien renombra su cuenta de GitHub, el identificador antiguo queda liberado y cualquiera puede registrarlo, y quien lo haga hereda esa entrada de tu `users.yml` en su siguiente inicio de sesión. Trata un cambio de nombre como un cambio de acceso: actualiza `users.yml` a la vez y elimina las entradas de quienes ya se han ido en lugar de dejar listado un identificador obsoleto.

`users.yml` es la lista de permitidos. Una cuenta de GitHub que no esté en `users.yml` no puede iniciar sesión, pertenezca a la organización que pertenezca. No hay aprovisionamiento automático: dar acceso a alguien significa añadirlo al archivo. Los filtros y los roles se resuelven desde `users.yml` en cada petición, igual que con los usuarios con contraseña, así que un usuario de GitHub con `roles: none` queda igual de limitado.

> [!NOTE]
> Dozzle no admite a propósito dar acceso a toda una organización de GitHub ni a un dominio de correo entero. Cada usuario se indica de forma individual. Si necesitas acceso por grupos o por dominio, usa `forward-proxy` con [Authelia](/es/guide/authentication/forward-proxy#configurar-dozzle-con-authelia) o Authentik, que están hechos para eso.

> [!WARNING]
> Editar `users.yml` rota la clave de firma de los JWT y cierra la sesión de todos los usuarios. Esto ya pasa hoy al añadir o quitar un usuario, y también se aplica cuando añades una clave `github`.

## Iniciar sesión con OIDC

Cualquier proveedor que publique un documento de descubrimiento de OpenID Connect funciona con este mismo callback: Google, Keycloak, Pocket ID, Zitadel, Authentik y otros. Apunta Dozzle a la URL del issuer y dale un client id y un client secret.

Registra Dozzle como cliente confidencial en tu proveedor y pon la redirect URI en:

```
https://your-dozzle-host/api/auth/callback
```

Incluye la ruta base si Dozzle se ejecuta bajo una, por ejemplo `https://example.com/dozzle/api/auth/callback`. A diferencia de GitHub, con OIDC Dozzle sí envía `redirect_uri`, así que este valor tiene que coincidir exactamente con el que hayas registrado.

::: code-group

```sh [cli]
$ docker run -v /var/run/docker.sock:/var/run/docker.sock -v /path/to/dozzle/data:/data -p 8080:8080 amir20/dozzle --auth-provider simple --auth-oidc-issuer https://id.example.com --auth-oidc-client-id dozzle --auth-oidc-client-secret secret --auth-oidc-name "Pocket ID"
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
      DOZZLE_AUTH_OIDC_ISSUER: https://id.example.com
      DOZZLE_AUTH_OIDC_CLIENT_ID: dozzle
      DOZZLE_AUTH_OIDC_CLIENT_SECRET: secret
      DOZZLE_AUTH_OIDC_NAME: Pocket ID
```

:::

`DOZZLE_AUTH_OIDC_NAME` es solo la etiqueta del botón de inicio de sesión. Por defecto es `SSO`.

La URL del issuer es la que sirve `/.well-known/openid-configuration`. Dozzle descarga ese documento para localizar los endpoints de autorización, token y userinfo, y se niega a iniciar el flujo si el documento indica un issuer distinto del que has configurado.

### Vincular usuarios

OIDC identifica al usuario por el **correo verificado**, no por el login. Define `email` en el usuario dentro de `users.yml`:

```yaml
users:
  admin:
    email: me@email.net
    name: Admin
    # la contraseña es opcional una vez vinculada la cuenta
```

Tu proveedor tiene que marcar ese correo como verificado. Dozzle rechaza el inicio de sesión cuando `email_verified` es falso, porque aceptar una dirección sin verificar permitiría que cualquiera capaz de registrarse en un proveedor permisivo se apropiara de una cuenta con solo escribir el correo de otra persona.

> [!NOTE]
> GitHub identifica por el login y OIDC por el correo, y esa diferencia es intencionada. Un login de GitHub es estable y siempre está presente, mientras que el correo de GitHub puede ser privado o cambiar. OIDC no tiene un equivalente estable y legible por personas, así que el correo verificado es el dato que los administradores conocen de verdad.

En este modo el proveedor solo demuestra quién eres. Los roles y los filtros siguen saliendo de `users.yml`, y un usuario que no esté en el archivo no puede iniciar sesión por muchos roles que lleve el token. Para que el proveedor decida las dos cosas, usa [`--auth-provider oidc`](/es/guide/authentication/oidc) en su lugar.

### Google

Google es un proveedor OIDC normal. Crea un cliente OAuth en la consola de Google Cloud y usa:

```
DOZZLE_AUTH_OIDC_ISSUER: https://accounts.google.com
DOZZLE_AUTH_OIDC_NAME: Google
```

`--auth-provider google` se acepta como alias de `simple`, así que sirve cualquiera de las dos formas.

## Detrás de un proxy inverso

Dozzle decide si la petición original usó HTTPS a partir de la cabecera `X-Forwarded-Proto`, y toma el nombre de host de `X-Forwarded-Host` cuando está presente. La mayoría de los proxies inversos envían las dos por defecto, pero si el tuyo no lo hace, el inicio de sesión se rompe de dos maneras.

Con OIDC el inicio de sesión se rompe directamente. El `redirect_uri` que envía Dozzle se construye a partir de esas cabeceras, así que un proxy que no envía `X-Forwarded-Proto: https` hace que Dozzle mande `http://your-host/api/auth/callback`. Eso no coincide con la URI `https://` registrada en tu proveedor, y el proveedor rechaza la petición en lugar de redirigir a ningún sitio útil.

Con GitHub la URL no se ve afectada, porque Dozzle omite `redirect_uri` y GitHub recurre al callback registrado en la OAuth App. La cabecera sigue decidiendo si la cookie de sesión se marca como `Secure`, así que conviene tenerla bien puesta en cualquier caso.

Puedes comprobar qué envía tu proxy mirando la cookie que Dozzle define cuando arranca un inicio de sesión:

```sh
$ curl -sI 'https://your-dozzle-host/api/auth/login?provider=github' | grep -i set-cookie
set-cookie: dozzle_oauth_state=...; Path=/; Max-Age=600; HttpOnly; Secure; SameSite=Lax
```

Que aparezca `Secure` en esa respuesta significa que la cabecera está llegando. Si falta, arregla el proxy antes de seguir. Consulta [Proxy inverso y ruta base](/es/guide/changing-base) para ver ejemplos con Nginx, Traefik y Caddy.

## Usar secretos de Docker para el client secret

Poner un client secret directamente en `environment:` significa que aparece en `docker inspect`, en tu archivo de compose y en el historial de la shell de cualquiera que haya arrancado el contenedor a mano. Por eso los dos client secrets aceptan una variante `_FILE` que indica el archivo del que leer el valor, siguiendo la convención que usan las propias imágenes de Docker:

| En lugar de                        | Usa                                     |
| ---------------------------------- | --------------------------------------- |
| `DOZZLE_AUTH_GITHUB_CLIENT_SECRET` | `DOZZLE_AUTH_GITHUB_CLIENT_SECRET_FILE` |
| `DOZZLE_AUTH_OIDC_CLIENT_SECRET`   | `DOZZLE_AUTH_OIDC_CLIENT_SECRET_FILE`   |

Dozzle lee el archivo al arrancar y recorta los espacios de alrededor, así que un salto de línea final de `echo secret > file` no da problemas. Definir a la vez una variable y su variante `_FILE` es un error, en lugar de dar preferencia a una en silencio, y apuntar `_FILE` a un archivo que no existe o está vacío detiene Dozzle al arrancar en vez de desactivar el botón de inicio de sesión sin avisar.

### Docker Compose

Fuera de Swarm no existe `docker secret create`, así que un secreto de Compose es o un archivo en disco o una variable de entorno. La forma con variable de entorno suele ser la que quieres: encaja con un `.env` ignorado por git y no deja ningún archivo en texto plano junto a tu archivo de compose esperando a que alguien lo suba al repositorio.

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    ports:
      - 8080:8080
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./data:/data
    environment:
      DOZZLE_AUTH_PROVIDER: simple
      DOZZLE_AUTH_GITHUB_CLIENT_ID: Ov23liABCDEFGHIJKLMN
      DOZZLE_AUTH_GITHUB_CLIENT_SECRET_FILE: /run/secrets/dozzle_github_secret
    secrets:
      - dozzle_github_secret

secrets:
  dozzle_github_secret:
    environment: GITHUB_CLIENT_SECRET
```

```ini [.env]
GITHUB_CLIENT_SECRET=your-github-client-secret
```

Compose lee la variable por su cuenta y monta el valor en `/run/secrets/dozzle_github_secret`. Nunca llega a formar parte del entorno del contenedor, así que se queda fuera de `docker inspect` igual que un secreto respaldado por un archivo.

Usa la forma con archivo cuando el secreto ya existe como archivo, por ejemplo uno escrito por un gestor de secretos:

```yaml [docker-compose.yml]
secrets:
  dozzle_github_secret:
    file: /run/secrets/github_client_secret
```

En ambos casos Dozzle recorta los espacios de alrededor, así que un salto de línea final en el archivo no importa.

### Docker Swarm

En Swarm el secreto lo gestiona el clúster y no un archivo en disco, así que créalo con `docker secret create` y decláralo como `external`:

```sh
printf '%s' 'your-oidc-client-secret' | docker secret create dozzle_oidc_secret_v1 -
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      DOZZLE_MODE: swarm
      DOZZLE_AUTH_PROVIDER: simple
      DOZZLE_AUTH_OIDC_ISSUER: https://id.example.com
      DOZZLE_AUTH_OIDC_CLIENT_ID: dozzle
      DOZZLE_AUTH_OIDC_CLIENT_SECRET_FILE: /run/secrets/dozzle_oidc_secret
      DOZZLE_AUTH_OIDC_NAME: Pocket ID
    secrets:
      - dozzle_oidc_secret
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    deploy:
      mode: global

secrets:
  dozzle_oidc_secret:
    external: true
    name: dozzle_oidc_secret_v1
```

Fíjate en la separación entre los dos nombres. `dozzle_oidc_secret` es el alias que usa este archivo de compose, y es el que decide la ruta de montaje: el secreto acaba en `/run/secrets/dozzle_oidc_secret`, que coincide con `_FILE`. `name:` es el objeto real en el swarm, y es el único sitio donde aparece la versión.

Esa separación existe porque los secretos de Swarm son inmutables. No hay forma de cambiar el valor de uno que ya existe, así que rotar un client secret filtrado o caducado significa crear la siguiente versión y apuntar el stack a ella. Mantener la versión fuera del alias convierte eso en un cambio de una sola línea, en lugar de tres ediciones que hay que mantener sincronizadas entre `_FILE`, la lista `secrets:` del servicio y la declaración de nivel superior:

```sh
printf '%s' 'your-new-client-secret' | docker secret create dozzle_oidc_secret_v2 -
```

```yaml [docker-compose.yml]
secrets:
  dozzle_oidc_secret:
    external: true
    name: dozzle_oidc_secret_v2 # was _v1
```

Vuelve a desplegar el stack y luego elimina el antiguo con `docker secret rm dozzle_oidc_secret_v1`. La variable de entorno y la ruta de montaje no se han movido.

> [!NOTE]
> Sin `name:`, un secreto se monta en `/run/secrets/<alias>` y el alias debe coincidir con el objeto real en el swarm. Con `name:` ambos quedan desacoplados, que es lo que hace que la rotación de arriba sea una única edición. En ambos casos `_FILE` apunta al alias, nunca a `name:`.

### Comprobar que funciona

Dozzle registra en el log los proveedores que ha activado al arrancar. Ejecútalo con `--level debug` y busca la línea que nombra el proveedor:

```sh
$ docker compose logs dozzle | grep -i 'sign in'
DBG Enabling Sign in with GitHub
```

Si el archivo del secreto falta o está vacío, Dozzle termina al arrancar con un mensaje que nombra la variable, así que un montaje mal hecho falla de forma ruidosa en lugar de quitar el botón de inicio de sesión sin más.
