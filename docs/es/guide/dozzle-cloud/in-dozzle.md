---
title: En tu Dozzle
description: Lo que cambia en tu propio Dozzle al enlazarlo con Dozzle Cloud, incluidos Ask Dozzle, el historial de métricas y las alertas que sobreviven a una recarga.
sourceHash: 6f212d9d6aeb
---

# En tu Dozzle

Lo que cambia en tu propio Dozzle una vez que una instancia está [enlazada](/es/guide/dozzle-cloud/connecting). Todo lo de esta página vive en la interfaz de Dozzle que ya usas, junto a los logs de los que habla, y no en un sitio aparte al que haya que ir.

## Nada local queda bloqueado

Dozzle no pierde funciones cuando Cloud no está configurado. Las alertas siguen disparándose, siguen apareciendo en el flujo de logs y siguen llegando a tus webhooks. Lo que añade el enlace es **memoria**: la misma alerta sigue ahí tras una recarga, tras un reinicio y una semana después.

Esa es toda la línea entre ambos. Dozzle es dueño del contenedor que tienes delante y puede actuar sobre él. Cloud es dueño de lo que es historia, de lo que abarca varias instancias y de lo que pertenece a la cuenta.

Por eso una instalación sin Cloud muestra una sección de historial vacía con una línea que dice qué contendría, y no una tarjeta bloqueada:

> Las alertas aparecerán aquí cuando esta instancia esté enlazada con Dozzle Cloud. Hasta entonces salen en el flujo de logs y se olvidan al recargar.

## El panel de Dozzle Cloud

Preguntar a Dozzle, Métricas y Alertas son pestañas de un mismo panel que se abre desde la derecha. Solo existe cuando Cloud está enlazado, y está en todas las páginas.

Una franja de iconos en el borde derecho es la forma de encontrarlo: cada icono abre su pestaña en el panel. Que la franja esté oculta se recuerda, y una pequeña pestaña en el borde la trae de vuelta, igual que la barra lateral se pliega en el otro lado. En un teléfono no hay franja, y el panel se abre a pantalla completa desde la barra de herramientas o la paleta de comandos.

El panel tapa los logs mientras está abierto, así que todo lo que en él apunta de vuelta al flujo (mostrar las líneas, un momento en un gráfico) primero lo cierra y luego desplaza el flujo hasta allí. <kbd>Esc</kbd> lo cierra.

Alertas trata de los contenedores que hay en pantalla, así que solo aparece en una página que muestre alguno. En la página de inicio o en Configuración, Métricas muestra en su lugar todos los hosts.

### <Icon icon="mdi:message-outline" inline /> Preguntar a Dozzle

Una pregunta sobre lo que estás mirando. En una vista de logs se responde con los logs que contiene; en cualquier otra página, con toda la instancia. Encima del campo de texto está lo que se envía con la pregunta: qué contenedores o qué host hay en la vista, cuántas líneas, y la línea de log que señalaste si llegaste desde el menú de una fila. Nada se envía en silencio.

Las respuestas terminan en Dozzle en lugar de en un enlace hacia fuera. Cuando la respuesta trata de un momento concreto, la acción lleva tu propio flujo hasta él.

Ábrelo con el botón **Preguntar a Dozzle** junto a la barra de búsqueda, con <kbd>⇧</kbd><kbd>⌘</kbd><kbd>K</kbd> en cualquier página, pulsando <kbd>.</kbd> en los logs de un contenedor, desde el menú del contenedor o desde una línea de log. En la paleta de <kbd>⌘</kbd><kbd>K</kbd>, una consulta que suena a pregunta pone **Preguntar** en primer lugar para que <kbd>Enter</kbd> la envíe, y <kbd>⇧</kbd><kbd>⌘</kbd><kbd>↵</kbd> pregunta desde cualquier fila. En Windows y Linux, <kbd>Ctrl</kbd> sustituye a <kbd>⌘</kbd>.

### <Icon icon="mdi:chart-line" inline /> Métricas

Dozzle guarda 300 muestras en el navegador y nada por detrás, así que «¿estaba así hace una hora?» no tiene respuesta local. Este panel relee las muestras que tu instancia lleva enviando todo el tiempo: CPU, memoria, red y disco en la última 1 h, 6 h o 24 h. Cada barra es una media con la muestra más alta dibujada tenuemente detrás, así que un pico breve sigue visible por larga que sea la ventana, y al pasar el cursor sobre cualquier gráfico se leen todos en el mismo instante. Red y disco suman lo que salió y lo que entró a lo largo de la ventana. Pro añade una ventana de 7 días.

En una página sin contenedor, la pestaña muestra en su lugar todos los hosts: CPU y memoria con los contenedores de cada host sumados, en las mismas ventanas. La memoria se dibuja frente al total del host cuando Dozzle lo conoce. La suma de CPU se lee como la carga del host para los contenedores sin límite de CPU; un contenedor con límite informa respecto a ese límite, así que en un host así la suma puede salir alta.

### <Icon icon="mdi:bell-outline" inline /> Alertas

Lo que se ha disparado en los contenedores que hay ahora en la vista. La página de [notificaciones](/es/guide/alerts-and-webhooks) responde a lo mismo para toda la instancia; esta pestaña se limita a lo que está en pantalla, y esa es la única razón por la que se gana un sitio en Dozzle. Abrirla apaga el punto de no leídas en la campana.

## Alertas que sobreviven a una recarga

Cuando las alertas se recuerdan, aparecen en tres sitios más:

- **En la regla que las disparó**, de modo que una regla escrita hace meses se juzga por lo que de verdad ha atrapado.
- **Como un punto en la fila del contenedor** de la tabla, con el color de la gravedad. Al hacer clic se abre un panel pequeño con el titular, el nivel, cuándo se disparó, cuántos eventos se agruparon y un resumen de una línea.
- **En la lista de actividad** de la página de notificaciones, filtrable y legible mucho después de que el aviso emergente haya desaparecido.

## «Muéstrame las líneas»

Todas esas superficies terminan en la misma acción, y siempre es la acción principal.

**Muéstrame las líneas** abre la vista histórica de ese contenedor, desplazada hasta el momento que describe la alerta o el hallazgo, con la línea exacta resaltada y el término de búsqueda ya escrito. Las alertas de métricas y de eventos no llevan ninguna línea de log, así que caen sobre el momento en lugar de sobre una fila.

Es el único movimiento que Dozzle puede hacer y Cloud no, y por eso estos paneles viven en Dozzle. Los enlaces hacia fuera se reservan para lo que Cloud hace de verdad mejor: facturación y claves de API, el archivo de informes, resúmenes entre instancias y las transcripciones de investigaciones completas.

## Hallazgos

Los hallazgos no son alertas. Una alerta es una regla que escribiste tú; un hallazgo es algo que Cloud notó y que ninguna regla cubre. Nunca se listan juntos, y la superficie de hallazgos existe solo cuando Cloud está configurado, en lugar de quedarse en la navegación como un anuncio permanente.

Consulta [Dozzle Cloud](/es/guide/dozzle-cloud) para ver qué cubren los hallazgos en cada plan.

## Cómo desactivarlo

Desenlazar la instancia elimina todas las superficies de esta página y deja Dozzle tal como estaba: las alertas siguen disparándose, siguen apareciendo en el flujo y se olvidan al recargar. Consulta [Tus datos](/es/guide/dozzle-cloud/your-data) para ver qué deja de salir de tu host.
