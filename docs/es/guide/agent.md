---
title: Modo agente
sourceHash: a73c7ed62d8a
---

# Modo agente

<Badge type="warning" text="Solo Docker" />

Dozzle puede ejecutarse en modo agente para exponer hosts de Docker a otras instancias de Dozzle. Toda la comunicación va por una conexión segura con TLS. Así puedes desplegar Dozzle en un host remoto y conectarte a él desde tu máquina local.

> [!WARNING] Un agente es tan privado como la red en la que escucha
> El certificado que trae Dozzle es el mismo en todas las copias de la imagen. Cifra la conexión, pero no demuestra quién está al otro lado. Cualquiera que alcance el puerto `7007` puede conectar su propio Dozzle a tu agente, leer todos los logs de ese host y ejecutar comandos dentro de sus contenedores. El agente no mira `DOZZLE_ENABLE_SHELL` ni `DOZZLE_ENABLE_ACTIONS`, esas opciones solo controlan lo que ofrece la interfaz.
>
> Mantén el puerto `7007` en una red privada y, si el agente es accesible desde fuera, [genera tu propio certificado](#certificados-propios) para que los agentes solo acepten tu hub.

> [!NOTE] ¿Usas Docker Swarm?
> Si usas el modo Docker Swarm, no necesitas agentes. Dozzle se descubrirá a sí mismo y creará un clúster usando el modo swarm. Consulta [Modo Swarm](/es/guide/swarm-mode) para más información.

## <Icon icon="mdi:plus-box-outline" inline /> Cómo crear un agente

Para crear un agente de Dozzle, ejecuta Dozzle con el subcomando `agent`. Aquí tienes un ejemplo:

::: code-group

```sh
docker run -v /var/run/docker.sock:/var/run/docker.sock -p 7007:7007 amir20/dozzle:latest agent
```

```yaml [docker-compose.yml]
services:
  dozzle-agent:
    image: amir20/dozzle:latest
    command: agent
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    ports:
      - 7007:7007
```

:::

> [!NOTE] Si usas un proxy del socket de Docker
> Si usas un agente remoto, **NO PUEDES** poner un proxy de socket por encima del agente. Los agentes de Dozzle **SUSTITUYEN** al proxy; consulta [Hosts remotos](/es/guide/remote-hosts) para más información y para saber cómo usar un proxy de socket en lugar de un agente.

El agente arrancará y escuchará en el puerto `7007`. Puedes conectarte al agente desde la interfaz de Dozzle indicando la dirección IP y el puerto del agente. El agente solo mostrará los contenedores disponibles en el host donde se ejecuta.

> [!TIP]
> No necesitas exponer el puerto 7007 si usas una red de Docker. El agente estará disponible para otros contenedores de la misma red. Es la forma más segura de ejecutar un agente, porque nada fuera de esa red puede alcanzarlo.

## <Icon icon="mdi:connection" inline /> Cómo conectarse a un agente

Para conectarte a un agente tienes que indicar su dirección IP y su puerto. Aquí tienes un ejemplo:

::: code-group

```sh
docker run -p 8080:8080 amir20/dozzle:latest --remote-agent agent:7007
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      - DOZZLE_REMOTE_AGENT=agent:7007
    ports:
      - 8080:8080 # puerto de la interfaz de Dozzle
```

:::

Ten en cuenta que no hace falta montar el socket local de Docker al conectarte a agentes; en ese caso la interfaz solo mostrará los contenedores disponibles en los agentes.

> [!TIP]
> Si además quieres incluir los contenedores del host en la interfaz, monta el socket `docker.sock` como se muestra en el ejemplo de [primeros pasos](/es/guide/getting-started).

> [!TIP]
> Puedes conectarte a varios agentes indicando varias variables de entorno `DOZZLE_REMOTE_AGENT`. Por ejemplo, `DOZZLE_REMOTE_AGENT=agent1:7007,agent2:7007`.

## <Icon icon="mdi:group" inline /> Grupos de hosts

Cuando gestionas muchos agentes en distintos entornos, puedes asignar cada agente a un grupo con nombre. Los grupos aparecen como secciones plegables en la barra lateral, y cada grupo tiene un botón de "unir todo" para ver los logs combinados de todos los hosts del grupo.

El formato de la cadena de conexión es `endpoint|name|group`, y las tres partes son opcionales:

| Formato                         | Resultado                          |
| ------------------------------- | ---------------------------------- |
| `agent:7007`                    | Sin nombre propio, sin grupo       |
| `agent:7007\|web-1`             | Nombre propio, sin grupo           |
| `agent:7007\|web-1\|Production` | Nombre propio + grupo              |
| `agent:7007\|\|Production`      | Nombre de host por defecto + grupo |

::: code-group

```sh
docker run -p 8080:8080 amir20/dozzle:latest \
  --remote-agent agent1:7007|web-1|Production \
  --remote-agent agent2:7007|web-2|Production \
  --remote-agent agent3:7007|dev-1|Development
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      - DOZZLE_REMOTE_AGENT=agent1:7007|web-1|Production,agent2:7007|web-2|Production,agent3:7007|dev-1|Development
    ports:
      - 8080:8080
```

:::

La barra lateral mostrará:

```
▾ Production
    web-1
    web-2
▾ Development
    dev-1
  ungrouped-host   ← agents without a group appear below
```

Al pulsar el icono de unir junto al nombre de un grupo se abre una vista de logs combinada con el streaming de todos los hosts de ese grupo. La vista combinada también está disponible directamente en `/host-group/<group-name>`.

Los agentes sin grupo siguen funcionando exactamente igual que antes y aparecen debajo de las secciones agrupadas.

## <Icon icon="mdi:alert-circle-outline" inline /> Problemas habituales

### El agente no aparece

Si ves `An agent with an existing ID was found. Removing the duplicate host.`, tienes dos hosts que usan el mismo ID de servidor.

Dozzle usa la API de Docker para recopilar información sobre los hosts. Cada agente necesita un ID de host único que se mantenga estable entre reinicios para poder identificarlo correctamente. Actualmente los agentes identifican el host usando el ID de sistema o el ID de nodo de Docker.

Si trabajas en un entorno Swarm, se usará el ID de nodo. Aun así, si notas que no se ven todos los hosts, puede deberse a hosts duplicados configurados con el mismo ID de host.

Para resolverlo, elimina `/var/lib/docker/engine-id` de tu sistema y reinicia. Esto ayudará a eliminar los conflictos causados por IDs de host duplicados. Para más información y consejos de diagnóstico, consulta las [preguntas frecuentes](/es/guide/faq#veo-un-error-de-hosts-duplicados-en-los-logs-como-lo-soluciono).

## <Icon icon="mdi:cog-outline" inline /> Opciones avanzadas

### Configurar el healthcheck

Puedes configurar un healthcheck para el agente, igual que el de la instancia principal de Dozzle. En modo agente, el healthcheck comprueba la conexión del agente con Docker. Si Docker no es accesible, el agente se marcará como no saludable y no se mostrará en la interfaz.

Para configurar el healthcheck usa el subcomando `healthcheck`. Aquí tienes un ejemplo:

```yml
services:
  dozzle-agent:
    image: amir20/dozzle:latest
    command: agent
    healthcheck:
      test: ["CMD", "/dozzle", "healthcheck"]
      interval: 5s
      retries: 5
      start_period: 5s
      start_interval: 5s
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    ports:
      - 7007:7007
```

### Cambiar el nombre del agente

Igual que con una instancia de Dozzle, puedes cambiar el nombre del agente con la variable de entorno `DOZZLE_HOSTNAME`. Aquí tienes un ejemplo:

::: code-group

```sh
docker run -v /var/run/docker.sock:/var/run/docker.sock -p 7007:7007 amir20/dozzle:latest agent --hostname my-special-name
```

```yaml [docker-compose.yml]
services:
  dozzle-agent:
    image: amir20/dozzle:latest
    command: agent
    environment:
      - DOZZLE_HOSTNAME=my-special-name
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    ports:
      - 7007:7007
```

:::

Esto cambiará el nombre del agente a `my-special-name` y se reflejará en la interfaz al conectarte al agente.

### Configurar filtros

Puedes configurar filtros en el agente para limitar los contenedores a los que accede. Estos filtros se pasan directamente a Docker y restringen lo que Dozzle puede ver.

```yaml
services:
  dozzle-agent:
    image: amir20/dozzle:latest
    command: agent
    environment:
      - DOZZLE_FILTER=label=color
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
```

Esto hará que el agente muestre solo los contenedores con la etiqueta `color`. Ten en cuenta que estos filtros se combinan con los de la interfaz para acotar los contenedores. Para conocer los distintos tipos de filtros, lee la [documentación de filtros](/es/guide/filters#ui-agents-and-user-filters).

### Certificados propios

Dozzle incluye un certificado autofirmado que ambos extremos se presentan entre sí, y cada extremo confía solo en ese certificado y en ninguno más. Está compilado dentro del binario, así que es idéntico en todas las instalaciones de Dozzle y cualquiera puede extraerlo de la imagen pública.

Eso te da cifrado, pero no autenticación. Un agente no puede distinguir tu hub del de otra persona. En un agente arrancado con los valores por defecto, lo único que mantiene fuera a los desconocidos es que no pueden alcanzar el puerto `7007`.

> [!WARNING] Cuándo necesitas tu propio certificado
> Si el puerto `7007` es accesible para algo que no controlas, incluido publicarlo en un host con IP pública, genera tu propio par. Cada despliegue que lo hace obtiene un secreto que nadie más tiene.

Ejecuta `generate-certs` para escribir un par único:

```sh
docker run --rm -v "$PWD":/out amir20/dozzle:latest generate-certs --cert-out /out/dozzle_cert.pem --key-out /out/dozzle_key.pem
```

Copia los dos archivos al hub y a cada agente. Dozzle los busca en `/dozzle_cert.pem` y `/dozzle_key.pem`, así que basta con montarlos ahí. Puedes ponerlos en otro sitio con los flags `--cert` y `--key` o con las variables de entorno `DOZZLE_CERT` y `DOZZLE_KEY`.

Trata la clave como una contraseña. Quien la tenga puede conectarse a tus agentes, y un agente rechaza cualquier hub que presente otra cosa, así que el hub y sus agentes tienen que recibir el mismo par y reiniciarse juntos.

Aquí tienes un ejemplo con las rutas por defecto:

```yml
services:
  agent:
    image: amir20/dozzle:latest
    command: agent
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    secrets:
      - source: cert
        target: /dozzle_cert.pem
      - source: key
        target: /dozzle_key.pem
    ports:
      - 7007:7007
secrets:
  cert:
    file: ./cert.pem
  key:
    file: ./key.pem
```

O con rutas personalizadas mediante variables de entorno:

```yml
services:
  agent:
    image: amir20/dozzle:latest
    command: agent
    environment:
      - DOZZLE_CERT=/certs/my-cert.pem
      - DOZZLE_KEY=/certs/my-key.pem
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./certs:/certs
    ports:
      - 7007:7007
```

O con flags de línea de comandos:

::: code-group

```sh
docker run -v /var/run/docker.sock:/var/run/docker.sock -v ./certs:/certs -p 7007:7007 amir20/dozzle:latest agent --cert /certs/my-cert.pem --key /certs/my-key.pem
```

```yaml [docker-compose.yml]
services:
  agent:
    image: amir20/dozzle:latest
    command: agent --cert /certs/my-cert.pem --key /certs/my-key.pem
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./certs:/certs
    ports:
      - 7007:7007
```

:::

> [!TIP]
> Es preferible usar secretos de Docker para aportar los certificados. Se pueden crear con el comando `docker secret create` o desde `docker-compose.yml` como en el ejemplo anterior. Hay que dar los mismos certificados a la instancia de Dozzle que se conecta al agente.

Esto montará los archivos de certificado y clave en el agente. El agente usará esos certificados para la comunicación. Hay que dar los mismos certificados a la instancia de Dozzle que se conecta al agente.

Si prefieres generarlos con openssl en lugar de con `generate-certs`:

```sh
$ openssl genpkey -algorithm Ed25519 -out key.pem
$ openssl req -new -key key.pem -out request.csr -subj "/C=US/ST=California/L=San Francisco/O=My Company"
$ openssl x509 -req -in request.csr -signkey key.pem -out cert.pem -days 365
```

## <Icon icon="mdi:compare-horizontal" inline /> Comparación entre agentes y conexión remota

Los agentes son parecidos a las conexiones remotas, pero tienen algunas ventajas. En general se prefieren los agentes por rendimiento y seguridad. Aquí tienes una comparación:

| Función     | Agente                               | Conexión remota                      |
| ----------- | ------------------------------------ | ------------------------------------ |
| Rendimiento | Mejor, con la carga distribuida      | Peor en la interfaz                  |
| Seguridad   | Cifrado, certificado propio opcional | Insegura o TLS de Docker             |
| Facilidad   | Fácil desde el primer momento        | Requiere exponer el socket de Docker |
| Permisos    | Acceso completo a Docker             | Se pueden controlar con un proxy     |
| Reconexión  | Se reconecta automáticamente         | Requiere reiniciar la interfaz       |
| Healthcheck | Healthcheck integrado                | Sin healthcheck                      |
| Filtros     | Admite filtros                       | No admite filtros                    |

Si aun así piensas usar conexiones remotas, asegúrate de proteger la conexión con TLS de Docker o un proxy inverso.
