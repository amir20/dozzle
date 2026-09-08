---
title: Canales de notificación
sourceHash: baae80ce439a
---

# Canales de notificación

Los canales se configuran en [Dozzle Cloud](/es/guide/dozzle-cloud) y controlan _a dónde_ van las alertas. Lo que las _dispara_ se configura en tu instancia autoalojada: consulta [Alertas](/es/guide/alerts-and-webhooks).

Activa los que quieras. Cada canal activado recibe todas las alertas, y cada uno se puede encender o apagar por separado.

## Canales disponibles

| Canal                                                            | Alertas | Resumen diario | Agente bidireccional |
| ---------------------------------------------------------------- | :-----: | :------------: | :------------------: |
| <Icon icon="mdi:email-outline" inline /> Correo                  |    ✓    |       ✓        |                      |
| <Icon icon="mdi:telegram" inline /> Telegram                     |    ✓    |       ✓        |          ✓           |
| <Icon icon="ic:baseline-discord" inline /> Bot de Discord (MD)   |    ✓    |       ✓        |          ✓           |
| <Icon icon="ic:baseline-discord" inline /> Webhook de Discord    |    ✓    |       ✓        |                      |
| <Icon icon="mdi:slack" inline /> Slack                           |    ✓    |                |                      |
| <Icon icon="simple-icons:ntfy" inline /> ntfy                    |    ✓    |                |                      |
| <Icon icon="mdi:webhook" inline /> Webhooks                      |    ✓    |                |                      |
| <Icon icon="mdi:bell-badge-outline" inline /> Push del navegador |    ✓    |                |                      |

Todos los canales están disponibles en todos los planes, incluido el gratuito.

## Correo

Se configura automáticamente con la dirección con la que te registraste. No hay nada que tocar. Para dejar de recibirlo, desactiva el canal de correo. Si las alertas dejan de llegar sin motivo aparente, mira primero en spam: la primera alerta acaba ahí de vez en cuando, y marcarla como «no es spam» lo arregla para siempre.

## Telegram

Elige **Telegram** en la página Channels, sigue el enlace para abrir el bot y pulsa **Start**. El canal se activa en cuanto el bot recibe algo tuyo.

Telegram es bidireccional. Puedes responder en el mismo chat y preguntar por tus contenedores («¿ha habido errores hoy?», «enséñame el uso de CPU», «¿qué alertas tengo?») y obtener respuestas sobre el estado en vivo.

## Discord

Discord tiene **dos tipos de canal distintos**, y tener los dos funcionando a la vez es el motivo habitual de recibir cada alerta por duplicado.

**Bot de Discord (mensaje directo)** — el bot te envía las alertas personalmente por MD. Es bidireccional, así que puedes hacerle preguntas. Se configura autorizando el bot desde la página Channels.

**Webhook de Discord (canal del servidor)** — las alertas se publican en un canal de tu servidor, por ejemplo `#alerts`. Unidireccional. Se configura creando un webhook en los ajustes de tu servidor de Discord y pegando la URL en Cloud.

Si las alertas te llegan tanto por MD como a un canal del servidor, tienes los dos configurados. Desactiva el que no quieras; apagar uno deja el otro funcionando. Una configuración habitual es quedarse con el canal compartido del servidor y apagar el MD.

## Slack

Crea un webhook entrante en tu espacio de Slack y pega la URL en el canal de Slack de la página Channels.

## ntfy

Introduce la URL de tu topic. Funcionan tanto ntfy.sh como un servidor ntfy autoalojado. Muy usado para notificaciones al móvil sin abrir otra cuenta.

## Webhooks

Introduce cualquier URL que acepte un POST. Las alertas se entregan como JSON, así que puedes encaminarlas hacia lo que ya tengas montado: Home Assistant, n8n, un script, otra herramienta de alertas.

> [!NOTE]
> Este es un canal de Cloud, distinto de los webhooks que tu Dozzle autoalojado puede llamar directamente. Esos están en [Alertas](/es/guide/alerts-and-webhooks), junto con las variables de plantilla de Go.

## Push del navegador

Actívalo en la página Channels y permite las notificaciones cuando el navegador te lo pida. Las alertas llegan entonces como notificaciones de escritorio.

Si después no llega nada, lo más probable es que el navegador denegara el permiso. Los navegadores no vuelven a preguntar una vez denegado: borra el permiso de notificaciones del sitio en los ajustes del navegador y actívalo de nuevo. El push del navegador no funciona en una ventana privada o de incógnito.

## <Icon icon="mdi:bell-sleep-outline" inline /> Bajar el ruido

Solo deberían interrumpirte cuando importa. Si Cloud hace ruido, es un problema de ajuste, y estas son las herramientas.

| Situación                                          | Haz esto                                |
| -------------------------------------------------- | --------------------------------------- |
| Un error recurrente que ya conoces                 | **Silencia el patrón**                  |
| Las alertas son útiles pero demasiado frecuentes   | **Vota con el pulgar hacia abajo**      |
| Mantenimiento planificado, copias, actualizaciones | **Silencia el patrón antes de empezar** |
| Alerta correcta, aplicación equivocada             | **Desactiva ese canal**                 |
| No quieres nada, de ningún sitio                   | **Desactiva todos los canales**         |

Borrar la regla de alerta casi nunca es la respuesta correcta: elimina toda una categoría de vigilancia para resolver una línea molesta.

### Silenciar una alerta recurrente

El silenciado es por patrón: calla _este tipo de alerta_, no solo la que tienes delante. Las apariciones posteriores se quedan calladas y lo que sea genuinamente distinto sigue llegando.

- **Desde una alerta** — ábrela en Cloud y elige silenciarla.
- **En el chat** — di «silencia esto» o «deja de avisarme de X». El agente enuncia el patrón exacto que va a silenciar y espera tu confirmación, porque un silenciado es duradero y podría ocultar un fallo real más adelante.

El silenciado dura hasta que lo deshagas. Pregunta «¿qué he silenciado?» para listar tus reglas, y quítalo del mismo modo. Las alertas silenciadas se siguen registrando: silenciar cambia lo que te interrumpe, no lo que se vigila.

### Menos, no ninguna

Si una alerta es genuinamente útil pero llega demasiado a menudo, vótala con el **pulgar hacia abajo** en lugar de silenciarla. Esa es la señal de «sigue vigilando esto, interrúmpeme menos». El pulgar hacia arriba en las alertas que acertaron ayuda igual.

### Las repeticiones ya se agrupan

Antes de silenciar, comprueba si el problema es repetición. Las apariciones repetidas del mismo fallo se condensan en una sola alerta con un contador. Si recibes muchas alertas, suelen ser muchos problemas _distintos_, o te has pasado del cupo de eventos de tu plan y las alertas han bajado a crudas y sin agrupar. Consulta [Planes y límites](/es/guide/dozzle-cloud/plans).

### Filtrar en el origen

Para un contenedor ruidoso durante su funcionamiento normal, el mejor arreglo está más arriba: la etiqueta `dev.dozzle.cloud.min_level` impide que las líneas de baja gravedad salgan siquiera de tu host. Consulta [Tus datos](/es/guide/dozzle-cloud/your-data).

## ¿Por qué no me llegó una alerta?

**1. ¿Existe una regla para eso?** Un error en tus logs no produce por sí solo una alerta; algo tiene que estar vigilándolo. La regla por defecto solo cubre contenedores que terminan con error: un contenedor que registra errores mientras sigue en pie necesita una regla de log.

**2. ¿Hay algún canal activado?** Una regla sin canal activado no tiene a dónde entregar.

**3. ¿Está conectada la instancia?** Si estaba desconectada cuando ocurrió el problema, no se reenvió nada. Consulta [Vincular tu instancia](/es/guide/dozzle-cloud/connecting).

**4. ¿Se agrupó en una alerta que ya recibiste?** Cuarenta fallos producen una alerta que dice cuarenta. Es lo previsto, no un fallo.

**5. ¿La silenciaste?** Revisa tus reglas de silenciado.

**6. ¿Está el contenedor excluido del reenvío?** Consulta [Tus datos](/es/guide/dozzle-cloud/your-data).

**7. ¿Estás por encima de los límites de tu plan?** Pasado el cupo, la entrega cambia y las alertas se muestrean.

**8. Mira la carpeta de spam**, en el caso del correo.
