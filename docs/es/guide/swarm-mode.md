---
title: Modo Swarm
sourceHash: f73672e30c85
---

# Modo Swarm

<Badge type="warning" text="Solo Docker" />

Dozzle funciona en modo Swarm de Docker. En modo Swarm, Dozzle descubre automáticamente los servicios y los grupos personalizados. Dozzle no usa internamente la API de Swarm porque es [limitada](https://github.com/moby/moby/issues/33183). En su lugar, implementa su propia agrupación a partir de las etiquetas de Swarm. Además, Dozzle combina las estadísticas de los contenedores de un mismo grupo, así que puedes ver los logs y las estadísticas de todos ellos en una sola vista. Eso sí, hay que instalar Dozzle en cada host.

## <Icon icon="mdi:cogs" inline /> ¿Cómo funciona?

Al desplegarse en modo Swarm, Dozzle crea una red mallada segura entre todos los nodos del swarm. Esa red sirve para que las distintas instancias de Dozzle se comuniquen entre sí. La red mallada se crea con [mTLS](https://www.cloudflare.com/learning/access-management/what-is-mutual-tls) y un certificado TLS privado, así que toda la comunicación entre las instancias de Dozzle va cifrada y se puede desplegar en cualquier sitio sin riesgo.

Dozzle admite [stacks](https://docs.docker.com/reference/cli/docker/stack/deploy/), [servicios](https://docs.docker.com/engine/swarm/how-swarm-mode-works/services/) y grupos personalizados de Docker para unir logs. Las etiquetas `com.docker.stack.namespace` y `com.docker.compose.project` se usan para agrupar contenedores. En el caso de los servicios, Dozzle usa el nombre del servicio como nombre del grupo, que viene de `com.docker.swarm.service.name`.

## <Icon icon="mdi:rocket-launch-outline" inline /> ¿Cómo se activa el modo Swarm?

Para desplegarlo en todos los nodos del swarm puedes usar `mode: global`. Así Dozzle se instala en cada nodo. Aquí tienes un ejemplo con Docker Stack:

```yml
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      - DOZZLE_MODE=swarm
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /opt/dozzle/data:/data
    ports:
      - 8080:8080
    networks:
      - dozzle
    deploy:
      mode: global
networks:
  dozzle:
    driver: overlay
```

Fíjate en que la variable de entorno `DOZZLE_MODE` vale `swarm`. Eso le dice a Dozzle que descubra automáticamente las demás instancias de Dozzle del swarm. La red `overlay` sirve para crear la red mallada entre ellas.

El volumen `/data` se monta para conservar la configuración de Dozzle (notificaciones, ajustes de cloud, stacks personalizados). Como Dozzle se despliega de forma global en todos los nodos, monta una ruta del host en cada uno para que cada instancia mantenga su estado local entre reinicios.

> [!WARNING]
> El socket-proxy no se puede usar en modo Swarm de Docker. Es una limitación del propio Docker, no de Dozzle. En modo Swarm los servicios solo pueden comunicarse con otros servicios, pero Dozzle necesita conexiones directas a cada instancia del proxy, y eso no está soportado. Si tienes una solución para usar socket-proxy en modo Swarm, nos encantaría conocerla.

## <Icon icon="mdi:shield-lock-outline" inline /> Configurar la autenticación simple en modo Swarm

Para configurar la autenticación simple puedes guardar el archivo `users.yml` en un secreto de Docker. Aquí tienes un ejemplo con Docker Stack:

```yml
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      - DOZZLE_LEVEL=debug
      - DOZZLE_MODE=swarm
      - DOZZLE_AUTH_PROVIDER=simple
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /opt/dozzle/data:/data
    secrets:
      - source: users
        target: /data/users.yml

    ports:
      - "8080:8080"
    networks:
      - dozzle
    deploy:
      mode: global

networks:
  dozzle:
    driver: overlay
secrets:
  users:
    file: users.yml
```

En este ejemplo, el archivo `users.yml` se guarda en un secreto de Docker. Es igual que el ejemplo de [autenticación simple](/es/guide/authentication#generating-users-yml).

## <Icon icon="mdi:server-plus-outline" inline /> Añadir agentes independientes al modo Swarm

Dozzle permite añadir [agentes](/es/guide/agent) independientes cuando se ejecuta en modo Swarm.

Basta con [añadir el agente remoto](/es/guide/agent#how-to-connect-to-an-agent) a tu Compose de Swarm igual que lo harías normalmente.

> [!NOTE]
> Los agentes remotos sí están soportados, pero las conexiones remotas como el socket proxy no.

```yml
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      - DOZZLE_MODE=swarm
      - DOZZLE_REMOTE_AGENT=agent:7007
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /opt/dozzle/data:/data
    ports:
      - 8080:8080
    networks:
      - dozzle
    deploy:
      mode: global
networks:
  dozzle:
    driver: overlay
```

Los agentes remotos aparecerán ahora junto al resto de nodos en Dozzle.
