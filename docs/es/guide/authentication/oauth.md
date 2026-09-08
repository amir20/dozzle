---
title: Iniciar sesión con GitHub y OIDC
sourceHash: cb9474f23fcc
---

# <Icon icon="mdi:shield-account" inline /> Iniciar sesión con GitHub y OIDC

Dozzle puede permitir que los usuarios inicien sesión con una cuenta externa en lugar de escribir una contraseña. Esto forma parte del proveedor [`simple`](/es/guide/authentication/simple) y no es un proveedor aparte, así que `users.yml` se sigue leyendo en cada petición y sigue decidiendo quién entra.

Eso tiene una consecuencia que conviene dejar clara desde el principio: **`users.yml` es la lista de permitidos.** Una cuenta externa que no esté vinculada a ninguna entrada no puede iniciar sesión, y nunca se crea ninguna cuenta de forma automática.

El inicio de sesión con contraseña sigue funcionando en paralelo, algo que importa cuando una OAuth App se rompe y necesitas entrar para arreglarla.

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
$ docker run -v /var/run/docker.sock:/var/run/docker.sock -v /path/to/dozzle/data:/data -p 8080:8080 amir20/dozzle --auth-provider simple --auth-github-client-id Iv1.0123456789abcdef --auth-github-client-secret 0123456789abcdef0123456789abcdef01234567
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
      DOZZLE_AUTH_GITHUB_CLIENT_ID: Iv1.0123456789abcdef
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

El valor es el **login** de GitHub (el identificador en `github.com/octocat`), no la dirección de correo. Los logins son estables y siempre visibles, mientras que el correo de una cuenta puede ser privado o cambiar en cualquier momento.

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

### Google

Google es un proveedor OIDC normal. Crea un cliente OAuth en la consola de Google Cloud y usa:

```
DOZZLE_AUTH_OIDC_ISSUER: https://accounts.google.com
DOZZLE_AUTH_OIDC_NAME: Google
```

`--auth-provider google` se acepta como alias de `simple`, así que sirve cualquiera de las dos formas.

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
      DOZZLE_AUTH_GITHUB_CLIENT_ID: Iv1.0123456789abcdef
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

En Swarm el secreto lo gestiona el clúster y no un archivo en disco, así que decláralo como `external` y créalo con `docker secret create`:

```sh
printf '%s' 'your-oidc-client-secret' | docker secret create dozzle_oidc_secret -
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
```

> [!NOTE]
> Un secreto se monta por defecto en `/run/secrets/<name>`, y por eso `_FILE` apunta ahí. Si defines un `target:` explícito, apunta `_FILE` a esa ruta.

### Comprobar que funciona

Dozzle registra en el log los proveedores que ha activado al arrancar. Ejecútalo con `--level debug` y busca la línea que nombra el proveedor:

```sh
$ docker compose logs dozzle | grep -i 'sign in'
DBG Enabling Sign in with GitHub
```

Si el archivo del secreto falta o está vacío, Dozzle termina al arrancar con un mensaje que nombra la variable, así que un montaje mal hecho falla de forma ruidosa en lugar de quitar el botón de inicio de sesión sin más.
