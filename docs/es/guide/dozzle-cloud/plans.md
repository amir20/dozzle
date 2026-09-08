---
title: Planes y límites
sourceHash: f7c6bbdc83ea
---

# Planes y límites

Qué incluye cada plan, qué cuenta contra él y qué pasa cuando te pasas.

El plan gratuito es el producto de alertas completo, no una versión de prueba. Lo que compran los planes de pago es la mitad proactiva (la revisión que lee tus logs y encuentra lo que ninguna alerta detectó) más margen para crecer.

## Planes

|                                                         |    Gratis    |       Pro       |      Team       |
| ------------------------------------------------------- | :----------: | :-------------: | :-------------: |
| Precio                                                  |     0 $      |    5 $ / mes    |   15 $ / mes    |
| Hallazgos (lee tus logs, encuentra lo que nunca alertó) |  1 / semana  | Todos, a diario | Todos, a diario |
| Solución incluida con cada hallazgo                     |      —       |        ✓        |        ✓        |
| El triaje inspecciona contenedores y logs si duda       |      —       |        ✓        |        ✓        |
| Investigación completa bajo demanda                     |      —       |        ✓        |        ✓        |
| Eventos procesados al mes                               |    2.000     |       50K       |      250K       |
| Logs consultables                                       | 10 GB · 24 h |  50 GB · 30 d   |  100 GB · 30 d  |
| Historial de alertas y eventos                          |    1 día     |     14 días     |     30 días     |
| Historial de métricas (CPU, memoria, red, disco)        |     24 h     |     30 días     |     30 días     |
| Instancias conectadas                                   |      1       |   Ilimitadas    |   Ilimitadas    |
| Chats con el asistente al mes                           |      10      |       200       |      1.000      |
| Soporte prioritario                                     |      —       |        ✓        |        ✓        |

Las alertas inteligentes, la agrupación de repeticiones, la supresión y los filtros por gravedad, la indexación para búsqueda, el registro de métricas, todos los canales de notificación, las acciones sobre contenedores y el acceso MCP ilimitado están en **todos** los planes, incluido el gratuito.

Los precios actuales están en [cloud.dozzle.dev](https://cloud.dozzle.dev).

## Qué cuenta como evento procesado

Un **evento procesado** es un evento de contenedor o una línea de log coincidente que ha pasado por la tubería de triaje, que decide si enviar una alerta nueva, agruparla en una existente o quedarse callada. Estás pagando por ese trabajo, no por almacenamiento en bruto.

Las líneas de log normales no son eventos: cuentan para el volumen de logs consultables. Así que un contenedor muy charlatán consume almacenamiento, mientras que un contenedor en bucle de caídas consume eventos.

Si un contenedor termina 47 veces, son 47 eventos contra el límite, pero una sola alerta que dice 47. De eso se trata.

**Los hallazgos no cuestan ninguna de las dos cosas.** La revisión de logs lee tus logs en vez de tu historial de alertas, así que no toca el contador de eventos y funciona sin ninguna regla de alerta configurada. Solo se aplica tu volumen mensual de logs.

**La búsqueda y las métricas son gratis en todos los planes.** La indexación está activa desde el momento en que se conecta una instancia, y las series de CPU, memoria, red y disco llegan desde tus instancias sin coste. El plan solo cambia cuánto y hasta dónde atrás.

## Superar el cupo

No se rompe nada. Pasas a modo de muestreo:

- El triaje se pausa.
- El historial de eventos sigue registrándose, así que no se pierde nada.
- Aproximadamente uno de cada diez eventos llega como alerta **en crudo**, para que sigas viendo qué pasa.
- Las repeticiones ya no se condensan en una alerta con contador.

El efecto práctico es que las alertas se vuelven más ruidosas y menos útiles en lugar de desaparecer, y lo notarás en el correo antes de verlo en una página de uso. Si tus alertas se han vuelto de golpe crudas y repetitivas, revisa primero el uso.

Esto también se aplica a los planes de pago. Los cupos se reinician al principio de cada mes.

> [!TIP]
> La mayoría de las cuentas ni se acercan. Solo una manguera de verdad (un contenedor reiniciándose en bucle durante días) supera el cupo gratuito, y la alerta que nombra ese contenedor llega mucho antes que el límite.

## Retención

La retención marca hasta dónde llega tu historial: alertas, eventos y resultados de búsqueda en logs. En el plan gratuito es un día, así que una búsqueda de algo de la semana pasada no devuelve nada aunque ocurriera. Es el motivo más habitual de que una búsqueda parezca «perder» datos.

La retención de métricas es de 24 horas en el plan gratuito y de 30 días en los de pago. Pedir una ventana más larga de la que permite tu plan devuelve la ventana que realmente tienes, no un error.

## El límite de instancias

El plan gratuito vincula una instancia a la vez. Al vincular una segunda aparece un mensaje de límite. Tienes dos opciones:

- **Mover la plaza.** Borra la clave de API de la instancia existente en la página Instances y vincula después la nueva. Esto es permanente para la instancia antigua: su historial se queda, pero tendrías que vincularla de cero otra vez.
- **Subir de plan** para mantener las dos conectadas a la vez.

El límite va de cuánto reenvía una cuenta gratuita, no es un candado sobre una función. Todo lo demás funciona en el plan gratuito con la instancia que tengas vinculada.

## Consultar tu uso

La página de uso en Cloud muestra los eventos, los bytes de logs y los chats con el asistente consumidos este mes frente a tu cupo. También puedes preguntarlo en el chat: «¿cuánto he usado este mes?».

## Cambiar o cancelar

Sube de plan desde la página de precios o desde los ajustes. La facturación la gestiona Stripe; los métodos de pago, las facturas y los recibos se administran allí a través del enlace de facturación en tus ajustes.

Cancelar detiene los cargos futuros y te pasa al plan gratuito al final del periodo que ya has pagado. Tu cuenta y tu historial se mantienen.
