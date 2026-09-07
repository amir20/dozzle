---
title: Autenticación
sourceHash: c0e3f963afbe
---

# Autenticación

Dozzle admite dos configuraciones de autenticación. En la primera, tú aportas tu propio método de autenticación protegiendo Dozzle detrás de un proxy. Dozzle puede leer las cabeceras correspondientes sin configuración adicional.

Si no tienes una solución de autenticación, Dozzle incluye una gestión de usuarios sencilla basada en archivos. Los proveedores de autenticación se configuran con el flag `--auth-provider`. En ambas configuraciones, Dozzle intentará guardar los ajustes de usuario en disco. Esos datos se escriben en `/data`.

## <Icon icon="mdi:shield-alert-outline" inline /> Consideraciones de seguridad

Dozzle tiene acceso a `docker.sock`, lo que, salvo que lo restrinjas, equivale a **root en el host**. Antes de exponer Dozzle fuera de tu red privada, revisa lo siguiente:

- **Pon siempre Dozzle detrás de autenticación** si es accesible desde internet. Usa `--auth-provider=simple` o un forward proxy como Authelia / Authentik / Cloudflare Access.
- **Mantén desactivadas las [acciones](/es/guide/actions) y el [acceso a la shell](/es/guide/shell)** salvo que los necesites. Permiten arrancar, parar, recrear y ejecutar comandos arbitrarios dentro de los contenedores.
- **Limita a los usuarios con [roles](/es/guide/authentication/simple#asignar-roles-especificos-a-los-usuarios) y [filtros](/es/guide/authentication/simple#asignar-filtros-especificos-a-los-usuarios)** en modo multiusuario. Sin roles explícitos, un usuario puede ver todos los contenedores que ve la instancia de Dozzle.
- **Nunca expongas el puerto de Dozzle directamente en modo forward proxy.** Dozzle confía en `Remote-User` en cada petición, y cuando no llega una cabecera de roles se le conceden todos los roles al usuario. Cualquiera que alcance el contenedor sin pasar por el proxy se autentica como quien quiera con solo poner una cabecera. Publica solo el proxy y deja Dozzle en una red interna usando `expose` en lugar de `ports`.
- **Termina el TLS en el proxy inverso**. Consulta [Proxy inverso y ruta base](/es/guide/changing-base) para ver ejemplos con Nginx / Traefik / Caddy.
- **Restringe el acceso a `docker.sock` con un proxy** si no necesitas las acciones. Ten en cuenta que montarlo en solo lectura (`/var/run/docker.sock:/var/run/docker.sock:ro`) _no_ limita la API: el flag `:ro` solo marca el archivo del socket como de solo lectura en disco, mientras que las llamadas a la API siguen pasando con normalidad, así que crear, borrar y actualizar siguen siendo posibles. Para restringir de verdad las operaciones, pon un proxy de socket como [`tecnativa/docker-socket-proxy`](https://github.com/Tecnativa/docker-socket-proxy) delante del daemon.

## <Icon icon="mdi:key-outline" inline /> Elegir un método

| Método                                                  | Quién gestiona los usuarios | Úsalo cuando                                                                                                                                               |
| ------------------------------------------------------- | --------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [Simple](/es/guide/authentication/simple)               | Dozzle, en `users.yml`      | No tienes ninguna solución de autenticación y quieres que Dozzle se encargue de los inicios de sesión.                                                     |
| [GitHub y OIDC](/es/guide/authentication/oauth)         | Dozzle, en `users.yml`      | Quieres que los mismos usuarios de `users.yml` inicien sesión con GitHub, Google, Keycloak, Pocket ID, Zitadel o Authentik en lugar de con una contraseña. |
| [Forward proxy](/es/guide/authentication/forward-proxy) | Tu proxy                    | Ya usas Authelia, Authentik, Cloudflare Access o algo similar, y quieres que se encargue por completo de la autenticación.                                 |

Simple y OAuth son el mismo proveedor: `users.yml` es la lista de usuarios en los dos casos, y OAuth solo añade una segunda forma de demostrar que eres uno de los usuarios que hay en ella. El forward proxy es el que va aparte, y es la opción correcta cuando necesitas reglas de acceso por organización o por dominio, algo que `users.yml` deliberadamente no hace.

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
