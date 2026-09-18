---
title: Planes y límites
sourceHash: be1bd7a795e4
---

# Planes y límites

Los planes, precios y cupos están en la [página de precios de Dozzle Cloud](https://cloud.dozzle.dev/pricing), que siempre está al día. Esta página explica cómo se ven los límites desde tu lado cuando te topas con ellos.

## Las alertas de repente llegan en crudo y repetidas

Lo más probable es que hayas superado tu cupo mensual de eventos. No se rompe nada: el historial de eventos sigue registrándose, pero el triaje se pausa, aproximadamente uno de cada diez eventos llega como alerta en crudo y las repeticiones ya no se condensan en una alerta con contador. Revisa primero la página de uso en Cloud. Los cupos se reinician al principio de cada mes, y esto también se aplica a los planes de pago.

## Una búsqueda no devuelve nada de la semana pasada

La búsqueda solo llega hasta donde alcanza la retención de tu plan. Todo lo anterior ya se ha borrado, aunque ocurriera. Pedir una ventana de métricas más larga de la que permite tu plan devuelve la ventana que realmente tienes, no un error.

## El límite de instancias

El plan gratuito vincula una instancia a la vez. Al vincular una segunda aparece un mensaje de límite. Tienes dos opciones:

- **Mover la plaza.** Borra la clave de API de la instancia existente en la página Instances y vincula después la nueva. Esto es permanente para la instancia antigua: su historial se queda, pero tendrías que vincularla de cero otra vez.
- **Subir de plan** para mantener las dos conectadas a la vez.

## Consultar tu uso

La página de uso en Cloud muestra los eventos, los bytes de logs y los chats con el asistente consumidos este mes frente a tu cupo. También puedes preguntarlo en el chat: «¿cuánto he usado este mes?».

## Cambiar o cancelar

Sube de plan desde la página de precios o desde los ajustes. La facturación la gestiona Stripe; los métodos de pago, las facturas y los recibos se administran allí a través del enlace de facturación en tus ajustes.

Cancelar detiene los cargos futuros y te pasa al plan gratuito al final del periodo que ya has pagado. Tu cuenta y tu historial se mantienen.
