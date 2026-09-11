---
title: Dozzle Cloud
sourceHash: 8426d3f379ba
---

# Dozzle Cloud

[Dozzle Cloud](https://cloud.dozzle.dev) es un complemento gestionado opcional para Dozzle autoalojado. Dozzle sigue siendo totalmente de código abierto y autoalojado; Cloud se apoya encima y se encarga de la parte que de verdad cuesta montar por tu cuenta: decidir qué merece despertarte y averiguar qué se ha roto realmente.

Tu Dozzle abre una conexión saliente hacia Cloud. No hay puerto entrante, ni IP pública, ni ningún agente que instalar.

**El plan gratuito te deja en paz. Pro sale a buscar.**

## <Icon icon="mdi:bell-ring-outline" inline /> Gratis: una capa de notificaciones inteligente

La mayoría de las alertas de logs son una expresión regular y un webhook, lo que significa que el primer bucle de caídas se convierte en doscientos mensajes idénticos y acabas silenciando el canal. El plan gratuito existe para arreglar esa parte, y es el producto de alertas completo, no una versión de prueba.

- **Alertas inteligentes** — cada alerta que disparan tus reglas de Dozzle se convierte en una frase que nombra la causa, el contenedor y la gravedad, con un enlace de vuelta a la línea de log exacta en tu propio Dozzle.
- **Las repeticiones se agrupan** — 47 caídas llegan como una sola alerta que dice 47. Y recibes un aviso de recuperación cuando vuelve.
- **Silencio por defecto** — supresión, filtros por gravedad y silenciado por patrón en todos los canales. Silencia _este tipo de alerta_ en vez de esta alerta concreta, y lo que sea genuinamente distinto sigue llegando.
- **Todos los canales** — correo, Telegram, Discord, Slack, ntfy, webhooks y notificaciones del navegador, todos en el plan gratuito. Consulta [Canales de notificación](/es/guide/dozzle-cloud/channels).
- **Búsqueda y métricas incluidas** — cada evento se puede consultar en cuanto llega, y CPU, memoria, red y disco se registran como historial. Ninguno cuenta contra tu cupo de eventos.
- **Un hallazgo por semana** — incluso en el plan gratuito, Cloud lee tus logs y saca a la luz lo más grave que ninguna alerta detectó.
- **Una regla por defecto que funciona** — al vincular una instancia se crea una por ti (contenedores que terminan con error), así que una cuenta nueva recibe una alerta útil el primer día sin configurar nada.
- **Agente de chat y MCP** — pregunta «¿ha habido errores hoy?» en Telegram o Discord, y arranca, para o reinicia un contenedor desde la misma conversación en cuanto habilites [Acciones](/es/guide/actions) en tu instancia. El acceso MCP es ilimitado en todos los planes.

> [!TIP]
> Una instancia recién vinculada disfruta 7 días de la experiencia Pro completa: todos los hallazgos, cada mañana. Después el plan gratuito se asienta en un hallazgo por semana.

## <Icon icon="mdi:robot-outline" inline /> Pro: sale a buscar antes de que salte nada

El plan gratuito te dice _que_ ha pasado algo, y se calla cuando no ha pasado nada. Pro es la mitad que no espera a que exista una alerta.

- **Revisión proactiva, cada mañana** — Cloud lee tus logs de error, los agrupa en patrones e informa de lo que merece la pena arreglar. Aquí es donde aparece un disco que se va llenando o un contenedor que se reinicia en silencio, un día en el que no saltó absolutamente nada. No hace falta que exista ninguna regla de alerta.
- **Todos los hallazgos, a diario, con la solución** — no uno por semana y el resto bloqueado. Los hallazgos envejecen día a día mientras dura el problema («sigue pasando, día cuatro, tres veces peor») y se cierran solos cuando para.
- **Un triaje que va y mira** — cuando el texto de la alerta no basta para decidir, inspecciona el contenedor y lee los logs de alrededor antes de pronunciarse, en lugar de adivinar.
- **Investigaciones completas bajo demanda** — un clic lanza más pasadas con un modelo más potente, correlacionando entre tus contenedores, hosts y línea temporal, y te devuelve una causa raíz con pasos concretos.
- **Todos los hosts, un panel** — conecta tantas instancias de Dozzle como tengas. Las preguntas en el chat las cubren todas a la vez.
- **Más memoria** — 30 días de logs y métricas consultables en lugar de 24 horas, que es la diferencia entre «qué pasó anoche» y «¿lleva esto pasando todo el mes?».

Tienes la comparativa completa en [Planes y límites](/es/guide/dozzle-cloud/plans).

## Por dónde seguir

| Página                                                     | Qué cubre                                                                                             |
| ---------------------------------------------------------- | ----------------------------------------------------------------------------------------------------- |
| [Vincular tu instancia](/es/guide/dozzle-cloud/connecting) | Vinculación, por qué no hace falta IP pública ni puerto abierto, cortafuegos, resolución de problemas |
| [En tu Dozzle](/es/guide/dozzle-cloud/in-dozzle)           | El carril de Cloud, las alertas que sobreviven a una recarga y lo que sigue funcionando sin Cloud     |
| [Canales de notificación](/es/guide/dozzle-cloud/channels) | Todos los canales, cómo configurar cada uno y cómo bajar el ruido                                     |
| [Planes y límites](/es/guide/dozzle-cloud/plans)           | Qué incluye cada plan, qué es un evento procesado, qué pasa si te pasas                               |
| [Tus datos](/es/guide/dozzle-cloud/your-data)              | Qué sale de tu host, cómo detenerlo, qué almacena Cloud, claves de API                                |

Las reglas de alerta se configuran en tu propia instancia, no en Cloud. Consulta [Alertas](/es/guide/alerts-and-webhooks).

## Comentarios

Dozzle Cloud lo construye la misma persona que hizo Dozzle, y el listón es el mismo: cosas que la gente quiera usar de verdad. Si lo pruebas y algo te chirría, falta o resulta genuinamente útil, [abre una discusión](https://github.com/amir20/dozzle/discussions). Ese feedback marca lo que se construye después.
