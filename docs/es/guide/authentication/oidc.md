---
title: OpenID Connect
sourceHash: 9dcc3df4a715
---

# <Icon icon="mdi:shield-account" inline /> OpenID Connect

Con `--auth-provider oidc`, tu proveedor de identidad es la base de datos de usuarios. En este modo Dozzle nunca lee `users.yml`: quién es un usuario, qué roles tiene y qué contenedores puede ver salen todos del token de OpenID Connect. Añade un usuario en Keycloak, Authentik, Zitadel o Pocket ID y podrá iniciar sesión; quítale el rol y ya no podrá.

Esto es distinto de [iniciar sesión con OIDC con los usuarios de `users.yml`](/es/guide/authentication/oauth#iniciar-sesion-con-oidc) bajo el proveedor `simple`. Ahí el proveedor solo demuestra quién eres y `users.yml` sigue decidiendo qué obtienes. Elige `simple` cuando quieras listar cada usuario a mano, y `oidc` cuando el proveedor deba ser el dueño de la lista.

## Configuración mínima

Registra Dozzle como cliente confidencial en tu proveedor y pon la redirect URI en:

```
https://your-dozzle-host/api/auth/callback
```

Incluye la ruta base si Dozzle se ejecuta bajo una, por ejemplo `https://example.com/dozzle/api/auth/callback`. Después apunta Dozzle al issuer:

::: code-group

```sh [cli]
$ docker run -v /var/run/docker.sock:/var/run/docker.sock -v /path/to/dozzle/data:/data -p 8080:8080 amir20/dozzle --auth-provider oidc --auth-oidc-issuer https://keycloak.example.com/realms/main --auth-oidc-client-id dozzle --auth-oidc-client-secret secret
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
      DOZZLE_AUTH_PROVIDER: oidc
      DOZZLE_AUTH_OIDC_ISSUER: https://keycloak.example.com/realms/main
      DOZZLE_AUTH_OIDC_CLIENT_ID: dozzle
      DOZZLE_AUTH_OIDC_CLIENT_SECRET: secret
```

:::

Eso es todo. Para las disposiciones más habituales no hace falta configurar ninguna ruta de claim, y `DOZZLE_AUTH_OIDC_NAME` solo cambia la etiqueta del botón de inicio de sesión. El client secret también acepta una variante `_FILE`, consulta [Usar secretos de Docker](/es/guide/authentication/oauth#usar-secretos-de-docker-para-el-client-secret).

La URL del issuer es la que sirve `/.well-known/openid-configuration`. Dozzle descarga ese documento para localizar los endpoints de autorización, token y userinfo, y se niega a iniciar el flujo si el documento indica un issuer distinto del que has configurado.

## Roles

Los roles de un usuario se leen del token. Dozzle prueba estos claims en orden y se queda con el primero que exista:

1. `dozzle_roles`
2. `resource_access.<client-id>.roles`
3. `roles`

El client id ya está configurado, así que con `DOZZLE_AUTH_OIDC_CLIENT_ID=dozzle` la segunda ruta es `resource_access.dozzle.roles`, que es donde Keycloak coloca los roles de cliente. Para esa disposición no hace falta configurar nada más.

Si tus roles viven en otro sitio, `DOZZLE_AUTH_OIDC_ROLES_CLAIM` sustituye la búsqueda por la única ruta separada por puntos que le indiques:

```yaml
DOZZLE_AUTH_OIDC_ROLES_CLAIM: realm_access.roles
```

Cada claim se busca primero en el ID token y después en la respuesta de userinfo, así que da igual en cuál de los dos lo ponga tu proveedor. Se aceptan tres formas: un array de cadenas, una única cadena separada por comas o espacios, y un objeto cuyas claves son los roles, que es como Zitadel codifica los roles de proyecto.

Los nombres de rol son los mismos que en `users.yml`: `shell`, `actions`, `download`, `notifications`, `cloud` y `all`, con `^` para excluir, así que `all,^shell` concede todo excepto el acceso por shell. También se aceptan los nombres con prefijo `dozzle_`, lo que ayuda cuando el proveedor comparte un mismo claim de roles entre varias aplicaciones. Consulta [roles](/es/guide/authentication/simple#asignar-roles-especificos-a-los-usuarios) para ver qué desbloquea cada uno.

> [!WARNING]
> `groups` queda fuera de la lista a propósito. En Authentik o Google todos los usuarios pertenecen al menos a un grupo, así que buscar ahí convertiría "denegado" en "con sesión iniciada y puede leer todos los contenedores". Si lo que tienes son grupos, mapéalos a un claim `dozzle_roles` en el proveedor, consulta los ejemplos más abajo.

### Cuando se rechaza un inicio de sesión

El inicio de sesión se rechaza cuando no existe ninguno de los claims, o cuando el primero que existe está vacío. El log nombra las rutas que se han probado:

```
WRN OIDC login rejected: no roles claim found in the ID token or userinfo, or it was empty sub=... tried="dozzle_roles, resource_access.dozzle.roles, roles"
```

Un claim que está presente pero no contiene nada que Dozzle reconozca como rol es otra cosa. Ese usuario inicia sesión sin privilegios, igual que con `roles: none` en `users.yml`: puede leer los logs de los contenedores que permita su filtro, y nada más. Los roles de realm de Keycloak se comportan así, porque ahí todos los usuarios llevan `offline_access` y `uma_authorization`, y por eso los ejemplos de más abajo usan roles de cliente en su lugar.

## Filtros

Los filtros de contenedores funcionan de la misma manera: se leen del primero de `dozzle_filters`, `resource_access.<client-id>.filters` y `filters` que exista, o de la única ruta indicada en `DOZZLE_AUTH_OIDC_FILTERS_CLAIM`. Cada valor es un filtro con la [misma sintaxis que `users.yml`](/es/guide/authentication/simple#asignar-filtros-especificos-a-los-usuarios), por ejemplo `label=com.example.app` o `name=web`:

```json
"resource_access": {
  "dozzle": {
    "roles": ["shell", "actions"],
    "filters": ["label=com.example.app"]
  }
}
```

Un usuario sin claim de filtros ve todos los contenedores que ve la instancia de Dozzle. Un filtro que no se puede interpretar hace fallar el inicio de sesión en lugar de descartarse, así que una errata en el proveedor no puede ampliar en silencio lo que alguien ve.

## Identidad

El claim `sub` es el identificador estable del usuario. Es la clave del directorio de perfil bajo `/data`, así que los ajustes siguen a la persona aunque su nombre de usuario o su correo cambien en el proveedor. El nombre que se muestra en el menú es `name`, con `preferred_username` como alternativa, después `email` y por último `sub`. `email` y `picture` alimentan el avatar, y cuando el proveedor envía una URL en `picture` se usa directamente.

A diferencia del proveedor `simple`, aquí el correo no tiene que estar verificado. Solo se muestra, nunca se compara con nada, así que un issuer que no conceda el scope de correo funciona sin problema.

## Sesiones

Tras iniciar sesión, Dozzle emite su propia cookie de sesión con los roles y los filtros que leyó del token. Se aplican en cada petición, pero no se vuelven a consultar: un cambio de rol en el proveedor tiene efecto en el siguiente inicio de sesión del usuario. Si ese margen importa, pon [`--auth-ttl`](/es/guide/supported-env-vars) en algo como `8h` para que las sesiones caduquen y se vuelvan a establecer a partir de un token nuevo.

## Cerrar sesión

Cerrar sesión borra la sesión de Dozzle. Si `--auth-logout-url` está definido, el navegador se envía después a esa URL, así que apúntala a la URL de fin de sesión de tu proveedor para cerrar también la sesión en el proveedor:

```yaml
DOZZLE_AUTH_LOGOUT_URL: https://keycloak.example.com/realms/main/protocol/openid-connect/logout
```

## Qué cambia respecto a `simple`

Los dos proveedores comparten los flags `--auth-oidc-*`, así que la diferencia se nota en el comportamiento:

- `users.yml` nunca se lee. Si existe uno bajo `/data`, Dozzle registra en el log que lo está ignorando.
- No hay formulario de contraseña ni endpoint `/api/token`. El proveedor de identidad es la única forma de entrar, así que una URL de callback equivocada o un client secret caducado dejan a todo el mundo fuera hasta que se arregle.
- `--auth-github-*` es un error de arranque. GitHub no es un issuer de OpenID Connect y no publica ningún claim del que leer roles.
- Al arrancar se registran en el log el issuer y las rutas de claim de las que se leerán los roles.

## Ejemplos por proveedor

### Keycloak

Crea un cliente `dozzle` en tu realm con la autenticación de cliente activada y añade la redirect URI de arriba. Después, en la pestaña **Roles** del cliente, crea los roles de cliente que quieras repartir: `shell`, `actions`, `download`, `notifications`, `cloud` o `all`. Asígnalos a usuarios o grupos en **Role mapping**.

Keycloak emite los roles de cliente como `resource_access.<client-id>.roles`, que Dozzle ya busca. Revisa los **Client scopes** del cliente, abre el scope dedicado y confirma que el mapper **client roles** añade el claim al ID token o a userinfo; Dozzle lee los dos pero no el access token.

Para los filtros, añade un atributo de usuario `dozzle_filters` y un mapper **User Attribute** en el scope dedicado con el mismo nombre de claim en el token y **Multivalued** activado. Cada valor es un filtro, como `label=com.example.app`.

### Authentik

Añade un scope mapping en **Customization** → **Property Mappings** que devuelva los roles a partir de los grupos del usuario, y asócialo al proveedor de Dozzle:

```python
roles = []
if request.user.ak_groups.filter(name="dozzle-admins").exists():
    roles.append("all")
elif request.user.ak_groups.filter(name="dozzle-users").exists():
    roles.append("download")
return {"dozzle_roles": roles}
```

Un usuario que no esté en ninguno de los dos grupos recibe una lista vacía y se le rechaza.

### Zitadel

Concede a los usuarios roles de proyecto con los nombres de los roles de Dozzle y activa **Assert Roles on Authentication** en la aplicación. El claim de roles de Zitadel es un objeto cuyas claves son los nombres de rol, algo que Dozzle acepta, así que configura:

```yaml
DOZZLE_AUTH_OIDC_ROLES_CLAIM: urn:zitadel:iam:org:project:roles
```

### Google

Los tokens de Google no llevan ningún claim de roles, así que `oidc` no se puede usar con él. Usa en su lugar el [proveedor `simple` con inicio de sesión de Google](/es/guide/authentication/oauth#google), donde `users.yml` decide quién entra.
