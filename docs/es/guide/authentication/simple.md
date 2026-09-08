---
title: Autenticación simple
sourceHash: deea96688436
---

# <Icon icon="mdi:account-cog-outline" inline /> Autenticación simple

La gestión de usuarios propia de Dozzle. Los usuarios viven en un archivo `users.yml` que gestiona el propio Dozzle, que además sirve su propia página de inicio de sesión. Pon `--auth-provider` en `simple` para activarla.

Las contraseñas son una forma de demostrar que eres uno de esos usuarios. [Iniciar sesión con GitHub o OIDC](/es/guide/authentication/oauth) es la otra, y las dos leen el mismo `users.yml`.

> [!TIP]
> Usa el [comando `generate`](/es/guide/authentication#generar-users-yml) integrado para crear `users.yml` en lugar de escribir hashes bcrypt a mano.

Dozzle admite autenticación multiusuario poniendo `--auth-provider` en `simple`. En este modo, Dozzle intentará leer el archivo de usuarios desde `/data/`, dando prioridad a `users.yml` sobre `users.yaml` si ambos existen. Si solo hay uno, se usará ese. El log indicará qué archivo se está leyendo (por ejemplo, `Reading users.yml file`).

## Rutas de ejemplo:

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

## Ampliar la duración de la cookie de autenticación

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

## Asignar filtros específicos a los usuarios

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

## Asignar roles específicos a los usuarios

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
