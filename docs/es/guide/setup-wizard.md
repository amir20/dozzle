---
title: Asistente de configuración
sourceHash: 413c966adbb1
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

Si un ajuste ya está fijado por un flag o una variable de entorno, su interruptor es de solo lectura y lo indica.

### 3. Dozzle Cloud

[Dozzle Cloud](/es/guide/dozzle-cloud) envía alertas en cuanto algo falla, un resumen cada mañana de lo que hay que arreglar y guarda un historial que sobrevive a los reinicios. **Conectar Dozzle Cloud** vincula esta instancia y **Ahora no** sigue adelante. Este paso se omite si la instancia ya está vinculada o si no tienes permiso para vincularla.

### 4. Reinicio

El último paso lista los cambios guardados que todavía no están en uso. **Reiniciar Dozzle** reinicia el contenedor, espera a que vuelva y recarga la página. Si no hay nada pendiente, el paso solo indica que has terminado.

Si Dozzle no puede reiniciarse solo (por ejemplo, cuando no encuentra su propio contenedor), el asistente muestra en su lugar las variables de entorno que puedes añadir a tu archivo compose.

## <Icon icon="mdi:file-cog-outline" inline /> Dónde se guardan los ajustes

El asistente guarda lo que eliges en `/data/dozzle.yml`. Dozzle lee este archivo una sola vez al arrancar, por eso los cambios necesitan un reinicio. Dozzle se reinicia solo desde el asistente, así que no tienes que hacerlo a mano.

```yaml [/data/dozzle.yml]
authProvider: simple
enableActions: true
enableShell: false
```

| Clave           | Valores                           | Equivale a              |
| --------------- | --------------------------------- | ----------------------- |
| `authProvider`  | `none`, `simple`, `forward-proxy` | `DOZZLE_AUTH_PROVIDER`  |
| `enableActions` | `true`, `false`                   | `DOZZLE_ENABLE_ACTIONS` |
| `enableShell`   | `true`, `false`                   | `DOZZLE_ENABLE_SHELL`   |

Los flags y las variables de entorno siempre tienen prioridad sobre el archivo. Si `DOZZLE_ENABLE_ACTIONS` está definida, el valor de `dozzle.yml` se ignora y el asistente muestra el interruptor bloqueado. Para volver a gestionar un ajuste desde el asistente, quita la variable de tu archivo compose.

## <Icon icon="mdi:shield-lock-outline" inline /> Seguridad

- **El inicio de sesión es el primer paso.** Un reinicio tras guardar una cuenta o un proxy activa el inicio de sesión antes de que se pueda cambiar cualquier otro ajuste.
- **Solo un usuario con sesión iniciada puede cambiar las acciones y la shell o reiniciar Dozzle.** El usuario necesita todos los roles.
- **Sin inicio de sesión hay una ventana de 15 minutos.** Cuando `authProvider` es `none`, estos ajustes solo se pueden cambiar durante los 15 minutos siguientes al arranque de Dozzle. Después, el asistente queda en solo lectura hasta que actives el inicio de sesión o reinicies Dozzle.
- **Las rutas se siguen decidiendo al arrancar.** El asistente solo escribe en `dozzle.yml`. Los endpoints de acciones y shell se registran cuando Dozzle arranca, igual que con las variables de entorno, así que no se activa nada hasta que Dozzle se reinicia.
