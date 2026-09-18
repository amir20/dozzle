---
title: Dozzle Cloud
sourceHash: ddf1502ea23d
---

# Dozzle Cloud

[Dozzle Cloud](https://cloud.dozzle.dev) es un complemento gestionado opcional para Dozzle autoalojado. Dozzle sigue siendo totalmente de código abierto y autoalojado; Cloud se apoya encima y se encarga de la parte que de verdad cuesta montar por tu cuenta: decidir qué merece despertarte y averiguar qué se ha roto realmente.

Tu Dozzle abre una conexión saliente hacia Cloud. No hay puerto entrante, ni IP pública, ni ningún agente que instalar.

El plan gratuito es toda la capa de alertas: cada alerta que disparan tus reglas de Dozzle se tría en un mensaje legible, las repeticiones se condensan en una alerta con contador y sale por correo, Telegram, Discord, Slack, ntfy, webhook o notificación push del navegador. Los planes de pago añaden la mitad proactiva: Cloud lee tus logs cada mañana e informa de los problemas que nunca dispararon una alerta, además de más historial y más instancias.

El recorrido completo está en [Funciones](https://cloud.dozzle.dev/features), y lo que incluye cada plan en [Precios](https://cloud.dozzle.dev/pricing).

## Por dónde seguir

| Página                                                     | Qué cubre                                                                                             |
| ---------------------------------------------------------- | ----------------------------------------------------------------------------------------------------- |
| [Vincular tu instancia](/es/guide/dozzle-cloud/connecting) | Vinculación, por qué no hace falta IP pública ni puerto abierto, cortafuegos, resolución de problemas |
| [En tu Dozzle](/es/guide/dozzle-cloud/in-dozzle)           | El carril de Cloud, las alertas que sobreviven a una recarga y lo que sigue funcionando sin Cloud     |
| [Canales de notificación](/es/guide/dozzle-cloud/channels) | Todos los canales, cómo configurar cada uno y cómo bajar el ruido                                     |
| [Planes y límites](/es/guide/dozzle-cloud/plans)           | Toparse con un límite, el límite de instancias, uso, cancelación                                      |
| [Tus datos](/es/guide/dozzle-cloud/your-data)              | Qué sale de tu host, cómo detenerlo, qué almacena Cloud, claves de API                                |

Las reglas de alerta se configuran en tu propia instancia, no en Cloud. Consulta [Alertas](/es/guide/alerts-and-webhooks).

## Comentarios

Dozzle Cloud lo construye la misma persona que hizo Dozzle, y el listón es el mismo: cosas que la gente quiera usar de verdad. Si lo pruebas y algo te chirría, falta o resulta genuinamente útil, [abre una discusión](https://github.com/amir20/dozzle/discussions). Ese feedback marca lo que se construye después.
