---
title: Vincular tu instancia
sourceHash: d6f8f7b6b845
---

# Vincular tu instancia

Cómo vincular un Dozzle autoalojado a [Dozzle Cloud](/es/guide/dozzle-cloud), confirmar que realmente está conectado y arreglarlo cuando no lo está.

## Vincular una instancia

1. Abre tu Dozzle autoalojado y haz clic en el icono de **nube** en la barra superior.
2. Haz clic en **Link instance**. Se te lleva a Cloud para iniciar sesión con GitHub o Google y confirmar.
3. La instancia aparece en el panel de Cloud en unos segundos.

No hay contraseña que crear ni ningún agente que instalar en el host.

## No necesitas IP pública, ni un puerto abierto, ni un dominio

Es la preocupación más frecuente, y la respuesta es no en los tres casos.

Tu instancia de Dozzle abre una conexión **saliente** hacia Cloud y la mantiene abierta. Cloud nunca se conecta de vuelta a ti, nunca busca tu host y nunca necesita alcanzar tu dirección. Eso significa que funciona con normalidad cuando Dozzle está:

- detrás de NAT en una red doméstica, sin redirección de puertos
- en una dirección privada RFC1918 como `192.168.1.50`
- en una red Tailscale, WireGuard o ZeroTier
- detrás de CGNAT, donde no podrías redirigir puertos aunque quisieras
- en un portátil que cambia de red

No hace falta proxy inverso, ni DNS dinámico, ni IP fija.

## Reglas del cortafuegos

Solo se necesita acceso **saliente**. Permite que tu host de Dozzle alcance:

```
agent.doligence.dozzle.dev:443    (TCP, outbound)
```

Ese único destino en el puerto 443 basta. Si tu cortafuegos filtra por nombre de host en vez de por IP, permite el nombre: las direcciones que hay detrás pueden cambiar. La mayoría de los cortafuegos domésticos y de oficina pequeña ya permiten todo el tráfico saliente, así que normalmente no hay nada que configurar.

## Dónde viven las reglas y los canales

Con esto tropieza casi todo el mundo, así que conviene decirlo claro.

| Lo que quieres cambiar                                           | Dónde se hace                                                 |
| ---------------------------------------------------------------- | ------------------------------------------------------------- |
| **Qué dispara una alerta** — contenedores, patrones, umbrales    | Dozzle autoalojado → [Alertas](/es/guide/alerts-and-webhooks) |
| **Dónde se entregan las alertas** — correo, Telegram, Slack, ... | Dozzle Cloud → [Canales](/es/guide/dozzle-cloud/channels)     |
| Revisar alertas pasadas, silenciar, cambiar de plan              | Dozzle Cloud                                                  |

La regla se define en tu propia instancia porque ahí están tus logs. La entrega se configura en Cloud porque ahí es donde se mantiene la conexión con tu móvil. Si buscas en Cloud un sitio para decir «avísame cuando este contenedor dé errores» y no lo encuentras, ese es el motivo: abre tu Dozzle autoalojado.

## Vincular más de una instancia

Cada instancia se vincula por separado, con los mismos pasos y la misma cuenta de Cloud. Una vez vinculadas, todas aparecen juntas en el panel, y las preguntas en el chat cubren todas las instancias conectadas a la vez.

Esa vista combinada vive en Cloud, no dentro de ningún Dozzle autoalojado concreto. Un Dozzle autoalojado muestra los hosts que configuraste directamente en él; no muestra otras instancias vinculadas.

Vincular varias instancias de Dozzle es distinto de las funciones propias de Dozzle [Modo agente](/es/guide/agent) y [Hosts remotos](/es/guide/remote-hosts), que conectan hosts de Docker adicionales a un único Dozzle. Ambas cosas están soportadas y se pueden combinar.

El plan gratuito vincula una instancia a la vez. Consulta [Planes y límites](/es/guide/dozzle-cloud/plans).

## No aparece nada

Repasa esto en orden.

**1. ¿Está el contenedor de Dozzle en marcha?**
Si Dozzle está parado o reiniciándose, no llega nada a Cloud.

**2. ¿Se llegó a completar la vinculación?**
Empezar el proceso y no aprobarlo no deja nada. Repite los pasos anteriores y confirma que la instancia aparece en la página Instances.

**3. ¿Se borró la clave de API?**
Borrar la clave de API de una instancia la desvincula de forma permanente. La clave antigua no se puede volver a asociar: vincula otra vez para obtener una nueva.

**4. ¿Se está bloqueando el tráfico saliente?**
Las redes restrictivas (empresas, universidades, algunos proveedores de VPS) pueden bloquear el 443 saliente hacia destinos que no estén en una lista blanca. Consulta _Reglas del cortafuegos_.

**5. ¿Has llegado al límite de instancias del plan gratuito?**
El plan gratuito vincula una instancia a la vez. Al intentar vincular una segunda aparece un mensaje de límite en lugar de conectarse.

**6. ¿Está el contenedor excluido del reenvío?**
Un contenedor con la etiqueta `dev.dozzle.cloud.min_level=disabled` no envía nada, por diseño. Si falta un contenedor concreto mientras los demás funcionan, revisa sus etiquetas. Consulta [Tus datos](/es/guide/dozzle-cloud/your-data).

## Dejar que el agente controle contenedores

Leer logs y estado de contenedores funciona en cuanto la instancia está vinculada. Arrancar, parar y reiniciar los rechaza tu instancia salvo que lo habilites tú:

::: code-group

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle
    environment:
      DOZZLE_ENABLE_ACTIONS: true
```

```sh
docker run ... amir20/dozzle --enable-actions
```

:::

Es un ajuste de **tu** Dozzle, no de Cloud, porque decide qué está dispuesto a hacer tu Dozzle con tus contenedores. Reinicia Dozzle tras cambiarlo. Consulta [Acciones](/es/guide/actions).

## Desvincular

Borra la clave de API de la instancia en la página Instances de Cloud. La conexión cae, no se reenvían más datos y tu Dozzle autoalojado sigue funcionando exactamente igual que antes. Vincular nunca cambia la visualización local de logs.
