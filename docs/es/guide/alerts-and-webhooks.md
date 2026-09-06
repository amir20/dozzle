---
title: Alertas y webhooks
sourceHash: daf372975955
---

# Alertas y webhooks

Dozzle tiene un sistema de alertas que te permite vigilar los logs de los contenedores, las métricas de recursos y los eventos del ciclo de vida, y recibir notificaciones cuando se cumplen ciertas condiciones. Las alertas usan expresiones personalizables para filtrar contenedores y definir la condición que las dispara, y pueden enviar notificaciones a webhooks, Slack, Discord, ntfy o [Dozzle Cloud](/es/guide/dozzle-cloud).

## <Icon icon="mdi:format-list-bulleted-type" inline /> Tipos de alerta

Dozzle admite tres tipos de alerta, todos se configuran igual desde la página de **Notificaciones**:

| Tipo                               | Se dispara con                                    | Caso de uso de ejemplo                 |
| ---------------------------------- | ------------------------------------------------- | -------------------------------------- |
| [**Log**](#alertas-de-log)         | Un mensaje de log que coincide con un patrón      | Errores 5xx, trazas de pila            |
| [**Métrica**](#alertas-de-metrica) | CPU o memoria que cruza un umbral                 | Un contenedor que supera el 90% de CPU |
| [**Evento**](#alertas-de-evento)   | Eventos de ciclo de vida del contenedor en Docker | OOM kills, contenedores no saludables  |

Cada alerta combina una **expresión de contenedor** (qué contenedores vigilar) con una **expresión de disparo** (la condición que la activa).

> [!IMPORTANT]
> La configuración de alertas y destinos se guarda en el directorio `/data`. Tienes que montar ese directorio como volumen para conservar los ajustes de notificación entre reinicios del contenedor.

::: code-group

```sh
docker run -v /var/run/docker.sock:/var/run/docker.sock -v /path/to/data:/data -p 8080:8080 amir20/dozzle:latest
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /path/to/data:/data
    ports:
      - 8080:8080
```

:::

## <Icon icon="mdi:send-outline" inline /> Configurar un destino

Antes de crear alertas necesitas configurar al menos un destino de notificación. Ve a la página de **Notificaciones** en Dozzle y pulsa **Añadir destino**.

### Webhook

Los webhooks envían una petición HTTP POST a la URL que elijas. Dozzle incluye plantillas de payload para servicios populares:

- **Slack**, con formato de bloques y markdown
- **Discord**, con formato para la API de webhooks de Discord
- **ntfy**, con formato para las notificaciones push de [ntfy.sh](https://ntfy.sh)
- **Personalizado**, un payload JSON genérico que puedes adaptar

También puedes escribir tu propia plantilla de payload con la sintaxis `text/template` de Go. Estas son las variables disponibles:

<div v-pre>

| Variable                  | Descripción                                      |
| ------------------------- | ------------------------------------------------ |
| `{{.Detail}}`             | Resumen (mensaje de log o valores de la métrica) |
| `{{.Container.Name}}`     | Nombre del contenedor                            |
| `{{.Container.Image}}`    | Imagen del contenedor                            |
| `{{.Container.HostName}}` | Nombre del host de Docker                        |
| `{{.Container.State}}`    | Estado del contenedor                            |
| `{{.Log.Message}}`        | Contenido del mensaje de log                     |
| `{{.Log.Level}}`          | Nivel del log                                    |
| `{{.Log.Timestamp}}`      | Marca de tiempo del log                          |
| `{{.Log.Stream}}`         | Tipo de flujo (stdout/stderr)                    |
| `{{.Stat.CPUPercent}}`    | Porcentaje de uso de CPU                         |
| `{{.Stat.MemoryPercent}}` | Porcentaje de uso de memoria                     |
| `{{.Stat.MemoryUsage}}`   | Uso de memoria en bytes                          |
| `{{.Subscription.Name}}`  | Nombre de la regla de alerta                     |

</div>

> [!TIP]
> Usa el botón **Probar** para comprobar que tu webhook funciona antes de guardarlo.

### Dozzle Cloud

También puedes enviar alertas a [Dozzle Cloud](/es/guide/dozzle-cloud) para centralizar la supervisión de varias instancias de Dozzle. Consulta la [guía de Dozzle Cloud](/es/guide/dozzle-cloud) para más detalles.

## <Icon icon="mdi:plus-circle-outline" inline /> Crear una alerta

Ve a la página de **Notificaciones** y pulsa **Añadir alerta**. Toda alerta tiene una **expresión de contenedor** y, además, una expresión de disparo de tipo **log**, **métrica** o **evento**.

### Expresión de contenedor

La expresión de contenedor selecciona qué contenedores vigilar. Propiedades disponibles:

| Propiedad  | Tipo   | Ejemplo                         |
| ---------- | ------ | ------------------------------- |
| `name`     | cadena | `name contains "api"`           |
| `image`    | cadena | `image == "nginx:latest"`       |
| `state`    | cadena | `state == "running"`            |
| `health`   | cadena | `health == "unhealthy"`         |
| `hostName` | cadena | `hostName == "prod-host"`       |
| `labels`   | mapa   | `labels["env"] == "production"` |

Puedes combinar condiciones con `&&` (Y), `||` (O) y `!` (NO):

```
name contains "api" && labels["env"] == "production"
```

## <Icon icon="mdi:text-search" inline /> Alertas de log

### Expresión de log

La expresión de log filtra qué mensajes de log disparan la alerta. Propiedades disponibles:

| Propiedad | Tipo        | Ejemplo                    |
| --------- | ----------- | -------------------------- |
| `message` | cadena/mapa | `message contains "error"` |
| `level`   | cadena      | `level == "error"`         |
| `stream`  | cadena      | `stream == "stderr"`       |
| `type`    | cadena      | `type == "complex"`        |

En los logs JSON puedes acceder a campos anidados con notación de punto:

```
message.status >= 500 && message.path contains "/api"
```

Entre los operadores de cadena admitidos están `contains`, `startsWith`, `endsWith` y `matches` (expresión regular).

### Ejemplos de log

**Alertar de todos los errores de los contenedores de producción:**

```
Container: labels["env"] == "production"
Log:       level == "error"
```

**Alertar de errores HTTP 5xx en los contenedores de API:**

```
Container: name contains "api"
Log:       message.status >= 500
```

**Alertar de cualquier salida por stderr de una imagen concreta:**

```
Container: image startsWith "myapp/"
Log:       stream == "stderr"
```

**Alertar de respuestas lentas de la API en producción:**

```
Container: name contains "api" && labels["env"] == "production"
Log:       message.duration > 5000 && message.path contains "/api"
```

**Alertar de fallos de autenticación con una expresión regular:**

```
Container: name contains "auth" || name contains "gateway"
Log:       message matches "(?i)(unauthorized|forbidden|invalid token)"
```

> [!NOTE]
> El editor de alertas incluye autocompletado y validación en tiempo real. Puedes previsualizar los contenedores y logs que coinciden antes de guardar.

## <Icon icon="mdi:chart-line" inline /> Alertas de métrica

Las alertas de métrica se disparan cuando el uso de CPU o memoria de un contenedor cruza un umbral. La expresión de disparo se evalúa sobre una media suavizada de las estadísticas tomadas en una ventana móvil, lo que evita falsas alarmas por picos breves.

### Expresión de métrica

Propiedades disponibles:

| Propiedad     | Tipo   | Descripción                                                |
| ------------- | ------ | ---------------------------------------------------------- |
| `cpu`         | número | Porcentaje de uso de CPU (0-100), igual que en la interfaz |
| `memory`      | número | Porcentaje de uso de memoria (0-100)                       |
| `memoryUsage` | número | Uso de memoria en bytes                                    |

### Enfriamiento y ventana de muestreo

- **Ventana de muestreo**: cuántos segundos de estadísticas se promedian antes de evaluar la expresión. Las ventanas largas suavizan los picos; las cortas reaccionan más rápido.
- **Enfriamiento**: segundos mínimos entre dos disparos consecutivos para el mismo contenedor. Evita una avalancha de alertas cuando un contenedor se mantiene por encima del umbral.

### Ejemplos de métrica

**CPU alta en los contenedores de producción:**

```
Container: labels["env"] == "production"
Metric:    cpu > 90
```

**Presión de memoria en un servicio concreto:**

```
Container: name contains "api"
Metric:    memory > 85
```

**Uso absoluto de memoria (1 GiB):**

```
Container: name == "postgres"
Metric:    memoryUsage > 1073741824
```

## <Icon icon="mdi:bell-outline" inline /> Alertas de evento

Las alertas de evento se disparan con los eventos del ciclo de vida de los contenedores de Docker, útiles para detectar caídas, OOM kills y cambios de estado de salud sin analizar logs.

### Expresión de evento

Propiedades disponibles:

| Propiedad    | Tipo   | Descripción                                               |
| ------------ | ------ | --------------------------------------------------------- |
| `name`       | cadena | Nombre del evento (ver abajo)                             |
| `actorId`    | cadena | ID del actor de Docker (normalmente el ID del contenedor) |
| `attributes` | mapa   | Atributos del evento de Docker (varían según el tipo)     |
| `timestamp`  | tiempo | Cuándo ocurrió el evento                                  |

Entre los nombres de evento habituales de Docker están `start`, `stop`, `die`, `kill`, `oom`, `restart`, `destroy` y `health_status`.

En los eventos `health_status`, Dozzle expone el estado actual como `attributes["healthStatus"]` (`healthy` o `unhealthy`).

### Ejemplos de evento

**Alertar cuando muere cualquier contenedor de producción:**

```
Container: labels["env"] == "production"
Event:     name == "die"
```

**Alertar de OOM kills:**

```
Container: true
Event:     name == "oom"
```

**Alertar cuando un contenedor pasa a estar no saludable:**

```
Container: true
Event:     name == "health_status" && attributes["healthStatus"] == "unhealthy"
```

**Alertar de salidas inesperadas (ignorando los apagados limpios y ordenados):**

Los códigos de salida 0 (éxito), 130 (SIGINT), 143 (SIGTERM) y 137 (SIGKILL) se producen con `docker stop`, Ctrl+C y los ciclos de actualización, así que se excluyen para evitar ruido. Las salidas con error reales (1, 2, 125, ...) siguen alertando.

```
Container: name contains "worker"
Event:     name == "die" && !(attributes["exitCode"] in ["0", "130", "143", "137"])
```

## <Icon icon="mdi:cog-outline" inline /> Gestionar las alertas

Desde la página de Notificaciones puedes:

- **Activar o desactivar** alertas sin borrarlas
- **Editar** las expresiones y los destinos de una alerta
- **Ver estadísticas**, incluidos el número de disparos, los contenedores coincidentes y la última vez que se disparó
- **Borrar** las alertas que ya no necesites
