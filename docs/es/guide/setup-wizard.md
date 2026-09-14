---
title: Asistente de configuración
sourceHash: 6c58a8385357
---

# Asistente de configuración

<Badge type="warning" text="Docker Only" />

Una instalación nueva de Dozzle se abre con un breve asistente de configuración. Te guía por las pocas cosas que casi todo el mundo cambia justo después de instalar: activar el inicio de sesión, permitir acciones sobre contenedores y acceso a la shell, y conectar Dozzle Cloud. Todo lo que guarda también se puede definir con flags o variables de entorno, así que el asistente es opcional.

El asistente solo aparece en una instalación nueva en modo servidor. Los despliegues de Swarm y Kubernetes nunca lo muestran. Puedes volver a abrirlo más tarde desde Ajustes.

## <Icon icon="mdi:format-list-numbered" inline /> Pasos

### 1. Inicio de sesión

El inicio de sesión va primero, para que no se pueda cambiar nada más en una instancia a la que cualquiera puede acceder.

Primero, el asistente comprueba que `/data` está montado en un volumen. Los ajustes y los usuarios se guardan ahí, y sin un volumen desaparecerían la próxima vez que se recree el contenedor. Si `/data` no es persistente, el asistente muestra cómo montarlo y espera a que pulses **Comprobar de nuevo**.

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - dozzle_data:/data
    ports:
      - 8080:8080
volumes:
  dozzle_data:
```

Cuando `/data` es persistente, elige una de estas tres opciones:

- **Cuenta de Dozzle** crea un único usuario con nombre de usuario, email opcional y contraseña. Dozzle escribe `/data/users.yml` y define `authProvider: simple`. Consulta [Autenticación simple](/es/guide/authentication/simple) para añadir más usuarios o roles más adelante.
- **Mi proxy** es para Authelia, Authentik, Cloudflare Access y similares. Dozzle confía en la cabecera `Remote-User`, así que publica solo el proxy y nunca el puerto de Dozzle. Esto define `authProvider: forward-proxy`. Consulta [Proxy inverso](/es/guide/authentication/forward-proxy).
- **OIDC** muestra un enlace a la guía de [OpenID Connect](/es/guide/authentication/oidc) y las variables de entorno que hay que añadir. OIDC necesita un client secret, así que aquí no se escribe nada y lo configuras tú.

Si Dozzle solo es accesible desde tu propia red, **Continuar sin inicio de sesión** se salta este paso.

Después de guardar una cuenta o un proxy, Dozzle se reinicia en el acto para que el inicio de sesión esté activo antes de cambiar cualquier otra cosa. Llegas a la página de inicio de sesión y, al entrar, el asistente continúa con el siguiente paso.

### 2. Acciones y shell

Dos interruptores controlan lo que Dozzle puede hacer con tus contenedores:

- **Iniciar, detener y reiniciar** activa las [acciones sobre contenedores](/es/guide/actions) (`enableActions`).
- **Shell** activa la posibilidad de [conectarse y ejecutar comandos](/es/guide/shell) dentro de los contenedores (`enableShell`). Está desactivado por defecto. El acceso a la shell de un contenedor suele equivaler a acceso al host, así que actívalo solo si lo necesitas.

Si un ajuste ya está fijado por un flag o una variable de entorno, su interruptor es de solo lectura y lo indica. Igual que el inicio de sesión, estos interruptores necesitan `/data` en un volumen, así que siguen en solo lectura hasta que lo esté.

### 3. Dozzle Cloud

[Dozzle Cloud](/es/guide/dozzle-cloud) envía alertas en cuanto algo falla, un resumen cada mañana de lo que hay que arreglar y guarda un historial que sobrevive a los reinicios. **Conectar Dozzle Cloud** vincula esta instancia y **Ahora no** sigue adelante. Este paso se omite si la instancia ya está vinculada o si no tienes permiso para vincularla.

### 4. Actualización automática

Dozzle puede mantenerse al día solo. Elige **Desactivada**, **Diaria** o **Semanal** (la semanal se ejecuta el domingo) y una hora del día. La hora es la local del servidor y por defecto es `03:00`. A esa hora Dozzle comprueba si su registro tiene una imagen más reciente y, solo si la hay, [se actualiza](#self-update).

Este ajuste se aplica al momento y no necesita reinicio.

Actualizarse es una acción, así que mientras las acciones están desactivadas este paso sigue en la lista, pero en gris y marcado con **Requiere acciones**. Activar las acciones en el paso 2 lo habilita al momento. Si esta instancia no puede actualizarse sola por otro motivo (por ejemplo, porque usa un tag de versión fijo), el paso indica el motivo en su lugar.

### 5. Reinicio

El último paso lista los cambios guardados que todavía no están en uso. **Reiniciar Dozzle** reinicia el contenedor, espera a que vuelva y recarga la página. Si no hay nada pendiente, el paso solo indica que has terminado.

Si Dozzle no puede reiniciarse solo (por ejemplo, cuando no encuentra su propio contenedor), el asistente muestra en su lugar las variables de entorno que puedes añadir a tu archivo compose.

## <Icon icon="mdi:file-cog-outline" inline /> Dónde se guardan los ajustes

El asistente guarda lo que eliges en `/data/dozzle.yml`. Dozzle lee este archivo una sola vez al arrancar, por eso los cambios necesitan un reinicio. Dozzle se reinicia solo desde el asistente, así que no tienes que hacerlo a mano. Las claves de actualización automática son la excepción: Dozzle las vuelve a leer cada minuto, así que se aplican sin reiniciar.

```yaml [/data/dozzle.yml]
authProvider: simple
enableActions: true
enableShell: false
autoUpdate: weekly
autoUpdateTime: "03:00"
```

| Clave            | Valores                           | Equivale a                |
| ---------------- | --------------------------------- | ------------------------- |
| `authProvider`   | `none`, `simple`, `forward-proxy` | `DOZZLE_AUTH_PROVIDER`    |
| `enableActions`  | `true`, `false`                   | `DOZZLE_ENABLE_ACTIONS`   |
| `enableShell`    | `true`, `false`                   | `DOZZLE_ENABLE_SHELL`     |
| `autoUpdate`     | `off`, `daily`, `weekly`          | `DOZZLE_AUTO_UPDATE`      |
| `autoUpdateTime` | `HH:MM`, hora local del servidor  | `DOZZLE_AUTO_UPDATE_TIME` |

Los flags y las variables de entorno siempre tienen prioridad sobre el archivo. Si `DOZZLE_ENABLE_ACTIONS` está definida, el valor de `dozzle.yml` se ignora y el asistente muestra el interruptor bloqueado. Para volver a gestionar un ajuste desde el asistente, quita la variable de tu archivo compose.

## <Icon icon="mdi:update" inline /> Cómo se actualiza Dozzle a sí mismo {#self-update}

Dozzle se actualiza con la acción `Update` sobre su propio contenedor o con la programación de actualización automática. Las dos hacen lo mismo:

1. Dozzle descarga el tag de imagen que está ejecutando. Si el tag sigue apuntando a la imagen en uso, se detiene ahí e informa de que está al día.
2. Dozzle arranca, a partir de la nueva imagen, un contenedor auxiliar de corta duración con acceso al mismo socket de Docker. Unos segundos después Dozzle desaparece.
3. El contenedor auxiliar renombra el contenedor antiguo y crea un reemplazo con el nombre original y la misma configuración, redes y volúmenes. Solo entonces detiene el contenedor antiguo y arranca el reemplazo. Los volúmenes anónimos también se conservan, así que los datos de `/data` sobreviven aunque no haya un volumen con nombre.
4. El contenedor auxiliar espera a que el reemplazo siga en marcha (y sano, si tiene healthcheck). Si lo consigue, se elimina el contenedor antiguo sin tocar sus volúmenes. Si no, se elimina el reemplazo y el contenedor antiguo recupera su nombre y vuelve a arrancar.

Los contenedores arrancados con `--rm` se actualizan igual. El contenedor antiguo se borra solo al detenerse, pero para entonces el reemplazo ya tiene sus volúmenes, así que se conservan. Si la actualización tiene que volver atrás, el contenedor auxiliar recrea el contenedor antiguo a partir de su configuración guardada.

Los logs del contenedor auxiliar son el único registro de una actualización. Se elimina solo al terminar, así que para seguir una, observa el contenedor `dozzle-self-update-*` mientras se ejecuta.

Algunas instalaciones no pueden actualizarse de esta forma:

- **Las acciones tienen que estar activadas.** La autoactualización necesita `DOZZLE_ENABLE_ACTIONS`, y la acción `Update` necesita el rol de acciones cuando el inicio de sesión está activo.
- **Solo en modo servidor.** Un servicio de Swarm de Dozzle se actualiza a través del manager de Swarm como cualquier otro servicio. Kubernetes y los agentes de Dozzle no se actualizan solos.
- **Los tags de versión fijos nunca se actualizan.** Descargar `amir20/dozzle:v8.12.0` siempre devuelve la misma imagen, así que la actualización automática no está disponible y una actualización manual informa de que está al día. Usa `latest` o cambia el tag tú mismo.

## <Icon icon="mdi:shield-lock-outline" inline /> Seguridad

- **El inicio de sesión es el primer paso.** Un reinicio tras guardar una cuenta o un proxy activa el inicio de sesión antes de que se pueda cambiar cualquier otro ajuste.
- **Solo un usuario con sesión iniciada puede cambiar las acciones, la shell y la actualización automática o reiniciar Dozzle.** El usuario necesita todos los roles.
- **Sin inicio de sesión, solo una instalación nueva tiene una ventana de 15 minutos.** Cuando `authProvider` es `none`, estos ajustes solo se pueden cambiar durante los 15 minutos siguientes al primer arranque de una instalación nueva, es decir, una cuyo `/data` estaba vacío. Una instalación que ya tiene datos de arranques anteriores nunca tiene la ventana, así que un reinicio del host o una actualización de la imagen no pueden abrirla. Fuera de la ventana, usa variables de entorno o activa el inicio de sesión.
- **Las rutas se siguen decidiendo al arrancar.** El asistente solo escribe en `dozzle.yml`. Los endpoints de acciones y shell se registran cuando Dozzle arranca, igual que con las variables de entorno, así que no se activa nada hasta que Dozzle se reinicia.
