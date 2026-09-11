---
title: Novedades de la v11
sourceHash: 4b0ece212354
---

# <Icon icon="mdi:party-popper" inline /> Novedades de la v11

La v11 es el mayor cambio visual que ha tenido Dozzle. Casi todas las superficies se redibujaron con un único lenguaje de diseño: plano, tranquilo, paneles neutros y el color reservado para lo que de verdad pide atención. También llegó el inicio de sesión con GitHub y OIDC, y el flujo de logs aprendió algunos formatos nuevos.

## Un aspecto nuevo

- **Barra lateral** reconstruida en torno a grupos plegables con un contador, iconos de aplicación de cada contenedor que llevan el estado como distintivo en la esquina y una fila activa teñida. Combinar un grupo entero es ahora un botón del propio grupo.
- **Flujo de logs** rediseñado. Las marcas de tiempo son discretas y sin recuadro, una línea simple recibe un punto de nivel y una entrada agrupada recibe una barra, y las filas `warn` y `error` llevan un tinte suave para localizarlas al desplazarse.
- **Barra de título del contenedor** simplificada. El nombre va primero, la imagen es texto atenuado que se copia al hacer clic, y anclar pasó a una chincheta que combina con la sección «Anclados» de la barra lateral.
- **Panel de inicio**, **paleta de comandos**, **avisos emergentes**, **paneles de attach y shell** y el **menú del contenedor** se redibujaron todos. El menú está dividido en secciones con nombre en lugar de una lista larga.
- El **indicador de flujo en vivo** y el **indicador de posición de desplazamiento** son nuevos. Este último flota sobre el flujo, dice en qué punto de la vida del contenedor estás y desaparece cuando dejas de desplazarte.
- Todos los menús usan ya la API nativa de popover, así que dejaron de recortarse o quedar atrapados dentro de un panel con desplazamiento.

## Inicio de sesión con GitHub y OIDC

Los usuarios pueden entrar con una cuenta de GitHub o con cualquier proveedor OIDC (Authentik, Keycloak, Pocket ID, Google). Forma parte del proveedor `simple`, así que `users.yml` sigue siendo la lista de permitidos y sigue decidiendo quién entra. Nunca se crea una cuenta automáticamente, y el acceso con contraseña sigue funcionando junto a esto.

```yaml
environment:
  DOZZLE_AUTH_PROVIDER: simple
  DOZZLE_AUTH_GITHUB_CLIENT_ID: Ov23liABCDEFGHIJKLMN
  DOZZLE_AUTH_GITHUB_CLIENT_SECRET: 0123456789abcdef0123456789abcdef01234567
```

La configuración completa está en [Inicio de sesión con GitHub y OIDC](/es/guide/authentication/oauth).

## Logs

- Los campos `severityText` y `severityNumber` de OpenTelemetry se reconocen como niveles de log, y se usa `severityNumber` cuando el texto no es un nombre de nivel.
- Se interpretan los niveles numéricos de Pino (`30`, `40`, `50`).
- Los interruptores de campos se aplican también en las vistas de servicio y stack, no solo en contenedores sueltos.
- Las columnas ancladas viven en la URL, de modo que una vista en paralelo es un enlace que puedes enviar.

## Alertas y Dozzle Cloud

- Las alertas se recuerdan. Aparecen en la regla que las disparó y como un punto en la fila del contenedor, y sobreviven a una recarga.
- Junto al flujo hay un nuevo carril de Cloud con tres paneles: preguntar sobre lo que estás viendo, las métricas que hay detrás del gráfico en vivo y las alertas que se dispararon en los contenedores de la vista. Solo se monta cuando Cloud está enlazado y se pliega en una pestaña en el borde.
- Las herramientas de Cloud se limitan a quien pregunta, así que el asistente ve exactamente lo que esa cuenta puede ver.
- `min_level=disabled` ya no detiene también las métricas.

## Rendimiento y correcciones

- Las primeras líneas de logs se pintan bastante antes.
- Los flujos de logs que el navegador abandonó en silencio vuelven a conectarse.
- Los puntos de estadísticas en la carga de cambios de contenedores son mucho más pequeños.
- Los flujos de logs combinados ya no bloquean el store de contenedores.

## Actualización

Los tokens de sesión se firman ahora con un secreto aleatorio guardado en `session_secret`, dentro del directorio de datos, junto a `users.yml`. **Al actualizar, todo el mundo sale de su sesión una vez.** Si `/data` no admite escritura, Dozzle arranca igualmente con un secreto en memoria y avisa de ello, lo que significa que las sesiones se pierden en cada reinicio.

La clave anterior se derivaba solo de `users.yml`, y eso dejó de ser entropía suficiente en cuanto una cuenta puede demostrarse por OAuth y no llevar ningún hash de contraseña.
