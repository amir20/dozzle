---
title: Tus datos
sourceHash: f11dd8e45cb5
---

# Tus datos

Qué sale de tu host, cómo detenerlo y qué almacena [Dozzle Cloud](/es/guide/dozzle-cloud) una vez que llega.

## Vincular no expone tu Dozzle

Tu Dozzle abre una conexión **saliente** hacia Cloud. No se abre nada entrante, no se redirige ningún puerto y Cloud no puede alcanzar tu instancia salvo por la conexión que ella misma inició. Si desvinculas, ese acceso termina al instante.

Vincular tampoco añade autenticación a tu Dozzle autoalojado. Esa es una cuestión aparte, y además importante: **por defecto Dozzle no tiene inicio de sesión.** Cualquiera que lo alcance en tu red puede ver tus logs. Si has expuesto Dozzle a internet o compartes red, configura la [Autenticación](/es/guide/authentication) en la propia instancia. Esto aplica vincules o no.

Las acciones sobre contenedores (arrancar, parar, reiniciar) las sigue rechazando tu instancia salvo que las habilites con `DOZZLE_ENABLE_ACTIONS`. Consulta [Acciones](/es/guide/actions).

## Controlar qué se reenvía

El control de privacidad más eficaz es no enviar algo en primer lugar. Por defecto, cada contenedor en ejecución envía sus logs a Cloud mientras esté vinculado. Para los contenedores cuyo parloteo a nivel info no tiene valor diagnóstico, o que manejan material que prefieres que no salga del host, filtra o desactiva con una etiqueta.

### `dev.dozzle.cloud.min_level`

| Valor                                         | Efecto                                                                                             |
| --------------------------------------------- | -------------------------------------------------------------------------------------------------- |
| _(sin definir)_                               | Se reenvían todas las líneas de log. Valor por defecto.                                            |
| `disabled`                                    | El contenedor se omite por completo. No se reenvía ningún log a Cloud.                             |
| `trace`                                       | Igual que sin definir, ya que trace es el nivel más bajo. Se reenvía todo.                         |
| `debug` / `info` / `warn` / `error` / `fatal` | Solo se reenvían las líneas de ese nivel o superior. Las líneas sin nivel detectado siempre pasan. |

Un valor no reconocido (una errata como `warning` o `wran`) se registra como error y se ignora, así que el contenedor envía todo como si la etiqueta no existiera.

La etiqueta se lee cuando arranca el lector de logs. Cambiarla en un contenedor en marcha surte efecto tras reiniciarlo.

```yaml
services:
  zigbee2mqtt:
    image: koenkk/zigbee2mqtt
    labels:
      # Reenviar solo warn/error/fatal a Dozzle Cloud
      - dev.dozzle.cloud.min_level=warn

  noisy-debug-tool:
    image: example/debug
    labels:
      # No enviar nada desde este contenedor
      - dev.dozzle.cloud.min_level=disabled
```

El filtro se ejecuta en tu instancia de Dozzle **antes de que los logs salgan del host**, así que las líneas descartadas nunca tocan la red ni cuentan para tu plan. La visualización local de logs en Dozzle no se ve afectada.

## Qué almacena Cloud

- **Líneas de log** reenviadas desde tus instancias vinculadas, para la búsqueda de texto completo.
- **Eventos y alertas** que coincidieron con tus reglas, con sus investigaciones y hallazgos.
- **Metadatos de contenedores y hosts** — nombres, imágenes, estados, uso de recursos.
- **Tu cuenta** — dirección de correo, plan, configuración de los canales de notificación.
- **Historial de chat** con el agente.

Todo está limitado a tu cuenta; otros usuarios no pueden ver tus datos. Los datos almacenados se conservan durante la ventana de retención de tu plan y luego se borran automáticamente. Consulta [Planes y límites](/es/guide/dozzle-cloud/plans).

## Claves de API

Cada instancia vinculada se autentica con su propia clave de API. Las claves se almacenan con hash BLAKE2b, admiten caducidad y nunca se guardan en texto plano.

Borrar una clave en la página Instances desconecta esa instancia de forma inmediata y permanente. La clave no se puede recuperar ni volver a asociar: vincula la instancia otra vez para obtener una nueva. Si crees que una clave se ha expuesto, bórrala y vuelve a vincular. Ese es el remedio completo: la clave antigua deja de funcionar en el momento en que se borra.

## Iniciar sesión

Cloud usa inicio de sesión con GitHub o Google. No hay contraseña aparte que crear, y Cloud nunca ve tu contraseña de GitHub o Google. Si te registras con un proveedor y luego inicias sesión con el otro usando la misma dirección de correo, llegas a la misma cuenta.

## Detener la recogida sin cerrar la cuenta

1. Borra las claves de API de tus instancias en la página Instances. El reenvío se detiene al momento.
2. Desactiva tus canales en la página Channels, para que no se entregue nada.

El historial existente caduca solo dentro de la ventana de retención de tu plan.

## Cerrar tu cuenta

Todavía no hay un botón de borrado autoservicio. Escribe a **amir@dozzle.dev** desde la dirección de la cuenta y pide el borrado. Si estás en un plan de pago, cancélalo primero desde los ajustes para que no te vuelvan a cobrar.

Antes de eso, o en lugar de eso, con los pasos anteriores eliminas prácticamente todo por tu cuenta: borrar tus claves de API detiene toda recogida, y los datos almacenados caducan con la retención.
