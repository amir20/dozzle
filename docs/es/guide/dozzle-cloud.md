---
title: Dozzle Cloud
sourceHash: 34c0056128a5
---

# Dozzle Cloud

[Dozzle Cloud](https://cloud.dozzle.dev) es un complemento gestionado opcional para Dozzle autoalojado. Conecta tus instancias entre sí, resume los eventos de los contenedores, reparte las alertas entre varios canales y te deja preguntar cosas sobre tu infraestructura desde el chat. Dozzle sigue siendo totalmente de código abierto y autoalojado; Cloud se monta encima.

La idea es que Dozzle Cloud funcione como ese asistente de SRE personal que no sabías que querías: vigila tus contenedores, te avisa cuando algo importa y no molesta cuando no pasa nada.

## Funciones

### <Icon icon="mdi:text-box-outline" inline /> Resúmenes de logs

Los eventos de los contenedores se agrupan y se resumen con un LLM. Cada resumen recoge la gravedad, el contenedor de origen y un enlace de vuelta a la línea de log completa en tu instancia de Dozzle.

### <Icon icon="mdi:group" inline /> Agrupación de patrones

Los errores repetidos se agrupan y se cuentan en vez de entregarse uno a uno. Un bucle que lanza la misma excepción 200 veces genera una sola notificación con su frecuencia, no 200.

### <Icon icon="mdi:robot-outline" inline /> Agente de IA

Un agente conversacional responde preguntas sobre el estado de los contenedores y la actividad reciente en los logs. Está disponible en Telegram y Discord.

En los planes Pro y Team, el agente también puede actuar sobre los contenedores (iniciar, parar, reiniciar) directamente desde la conversación, sin necesidad de acceso por shell al host.

### <Icon icon="mdi:calendar-clock" inline /> Resúmenes diarios

Un resumen programado de la actividad reciente en todas tus instancias enlazadas: patrones de error más frecuentes, número de eventos y estado general. Se envía por correo a la hora y en la zona horaria que configures.

### <Icon icon="mdi:bell-ring-outline" inline /> Canales de notificación

Las alertas se pueden enviar a varios canales en paralelo. Cada canal se activa o desactiva de forma independiente y se puede limitar a instancias concretas de Dozzle.

| Canal                                                            | Alertas | Resumen diario | Agente bidireccional |
| ---------------------------------------------------------------- | :-----: | :------------: | :------------------: |
| <Icon icon="mdi:telegram" inline /> Telegram                     |    ✓    |       ✓        |          ✓           |
| <Icon icon="ic:baseline-discord" inline /> Discord               |    ✓    |       ✓        |          ✓           |
| <Icon icon="mdi:email-outline" inline /> Correo                  |    ✓    |       ✓        |                      |
| <Icon icon="mdi:slack" inline /> Slack                           |    ✓    |                |                      |
| <Icon icon="simple-icons:ntfy" inline /> ntfy                    |    ✓    |                |                      |
| <Icon icon="mdi:webhook" inline /> Webhooks                      |    ✓    |                |                      |
| <Icon icon="mdi:bell-badge-outline" inline /> Push del navegador |    ✓    |                |                      |

### <Icon icon="mdi:bell-sleep-outline" inline /> Silenciar notificaciones

Las notificaciones se pueden silenciar durante una hora, ocho horas, hasta la mañana siguiente o hasta la semana siguiente. Viene bien durante una incidencia o un mantenimiento programado.

### <Icon icon="mdi:view-dashboard-outline" inline /> Panel multiinstancia

Las instancias de Dozzle enlazadas aparecen en un único panel. Cada instancia se autentica con una clave de API, sin necesidad de instalar ningún agente adicional en el host. El panel muestra el estado de conexión, el inventario de contenedores y los logs en directo.

### <Icon icon="mdi:database-search-outline" inline /> Búsqueda de texto completo en los logs

Cada línea de log que envían tus instancias enlazadas se escribe en un índice de búsqueda de texto completo. Puedes consultar todas las instancias a la vez o filtrar por contenedor, gravedad o intervalo de tiempo. Las búsquedas devuelven resultados en milisegundos incluso con semanas de historial, y cada coincidencia enlaza de vuelta a su contexto en la instancia de origen. La retención depende del plan y va de 24 horas a 30 días.

### <Icon icon="mdi:shield-lock-outline" inline /> Seguridad

- Las claves de API se guardan como hash BLAKE2b y admiten caducidad.
- El inicio de sesión usa OAuth de GitHub o Google.
- Los logs y el contenido de los eventos se guardan solo durante la ventana de retención de tu plan.

## Conectar una instancia

Para enlazar un Dozzle autoalojado con Dozzle Cloud:

1. Abre tu instancia de Dozzle y pulsa el icono de **nube** en la barra superior.
2. Pulsa **Enlazar instancia**. Se te redirigirá para autenticarte y confirmar la conexión.
3. Una vez enlazada, configura las suscripciones de alertas dentro de Dozzle para elegir qué eventos se envían.

## Controlar qué se envía

Por defecto, mientras la instancia está enlazada, todos los contenedores en ejecución envían sus logs a Dozzle Cloud. En contenedores ruidosos, donde el ruido de nivel info no aporta nada al diagnóstico, puedes filtrar o desactivar el envío por contenedor con una sola etiqueta.

### `dev.dozzle.cloud.min_level`

| Valor                                         | Efecto                                                                                           |
| --------------------------------------------- | ------------------------------------------------------------------------------------------------ |
| _(sin definir)_                               | Se envían todas las líneas de log. Es el valor por defecto.                                      |
| `disabled`                                    | El contenedor se omite por completo. No se envía ningún log a Cloud.                             |
| `trace`                                       | Igual que sin definir, ya que trace es el nivel más bajo. Se envía todo.                         |
| `debug` / `info` / `warn` / `error` / `fatal` | Solo se envían las líneas de ese nivel o superior. Las líneas sin nivel detectado siempre pasan. |

Un valor no reconocido (una errata como `warning` o `wran`) se registra como error y se ignora, así que el contenedor envía todo como si la etiqueta no estuviera.

La etiqueta se lee cuando arranca el lector de logs. Cambiarla en un contenedor en marcha no surte efecto hasta que el contenedor se reinicia.

```yaml
services:
  zigbee2mqtt:
    image: koenkk/zigbee2mqtt
    labels:
      # Enviar solo warn/error/fatal a Dozzle Cloud
      - dev.dozzle.cloud.min_level=warn

  noisy-debug-tool:
    image: example/debug
    labels:
      # No enviar nada desde este contenedor
      - dev.dozzle.cloud.min_level=disabled
```

El filtro se aplica en tu instancia de Dozzle antes de que los logs salgan del host, así que las líneas descartadas nunca pasan por la red ni cuentan para tu plan. La visualización local de logs en Dozzle no se ve afectada.

## Precios

El plan gratuito es generoso a propósito; deberías poder usar Dozzle Cloud de verdad en un homelab o en un equipo pequeño sin toparte con un muro. Hay planes de pago para volúmenes de eventos mayores, más retención y las acciones sobre contenedores del agente. Consulta [cloud.dozzle.dev](https://cloud.dozzle.dev) para ver los límites y los detalles de cada plan.

## Comentarios

Dozzle Cloud lo desarrolla la misma persona que hizo Dozzle, y el listón es el mismo: cosas que la gente quiera usar de verdad. Si lo pruebas y algo te chirría, te falta o te resulta especialmente útil, [abre una discusión](https://github.com/amir20/dozzle/discussions). Esos comentarios marcan lo que se construye después.
