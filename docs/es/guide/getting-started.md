---
title: Primeros pasos
sourceHash: 6fa151a446d0
---

# Primeros pasos

Dozzle se ejecuta como un único contenedor. Elige abajo entre la CLI de Docker, Docker Compose, Swarm o Kubernetes.

## <Icon icon="mdi:docker" inline /> Docker independiente

Monta `docker.sock` para que Dozzle pueda leer los contenedores, monta un volumen en `/data` para que la configuración sobreviva a un reinicio y publica el puerto 8080.

::: code-group

```sh [docker run]
docker run -d -v /var/run/docker.sock:/var/run/docker.sock -v dozzle_data:/data -p 8080:8080 amir20/dozzle:latest
```

```yaml [docker-compose.yml]
# Ejecutar con docker compose up -d
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - dozzle_data:/data
    ports:
      - 8080:8080
    environment:
      # Descomenta para habilitar las acciones sobre contenedores (parar, iniciar, reiniciar). Consulta https://dozzle.dev/guide/actions
      # - DOZZLE_ENABLE_ACTIONS=true
      #
      # Descomenta para permitir el acceso a la shell de los contenedores. Consulta https://dozzle.dev/guide/shell
      # - DOZZLE_ENABLE_SHELL=true
      #
      # Descomenta para habilitar la autenticación. Consulta https://dozzle.dev/guide/authentication
      # - DOZZLE_AUTH_PROVIDER=simple
      #
      # Pon nombre a esta instancia de Dozzle (se muestra en la cabecera y en el menú multihost). Consulta https://dozzle.dev/guide/hostname
      # - DOZZLE_HOSTNAME=my-server
      #
      # Conecta con uno o varios agentes remotos para monitorizar otros hosts de Docker. Consulta https://dozzle.dev/guide/agent
      # - DOZZLE_REMOTE_AGENT=192.168.1.10:7007,192.168.1.11:7007
      #
      # Muestra solo los contenedores que coincidan con un filtro. Consulta https://dozzle.dev/guide/filters
      # - DOZZLE_FILTER=label=com.example.app
volumes:
  dozzle_data:
```

:::

Abre `http://localhost:8080` y ya está. Todo lo demás, incluidas las acciones, el acceso a la shell, la autenticación y los agentes remotos, es opcional y viene desactivado. Las variables de entorno comentadas del archivo de Compose enlazan con cada guía.

> [!WARNING]
> Montar `docker.sock` le da a Dozzle un acceso al host equivalente al de root. Si piensas exponer Dozzle fuera de tu red privada, lee antes [Consideraciones de seguridad](/es/guide/authentication#security-considerations).

Dozzle necesita Docker Engine 19.03 o posterior (API versión 1.40+). Si Docker Hub está bloqueado en tu red, descarga `ghcr.io/amir20/dozzle:latest` desde el [GitHub Container Registry](https://ghcr.io/amir20/dozzle:latest).

## <Icon icon="mdi:hexagon-multiple-outline" inline /> Docker Swarm

Dozzle puede funcionar en modo Swarm desplegándolo en todos los nodos. Para ejecutar Dozzle en modo Swarm, usa esta configuración:

```yaml [dozzle-stack.yml]
# Ejecutar con docker stack deploy -c dozzle-stack.yml <name>
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      - DOZZLE_MODE=swarm
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
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

Después despliega el stack con este comando:

```bash
docker stack deploy -c dozzle-stack.yml <name>
```

Consulta [modo Swarm](/es/guide/swarm-mode) para más información.

## <Icon icon="mdi:kubernetes" inline /> K8s

Dozzle puede ejecutarse en Kubernetes. Solo hace falta desplegarlo en un nodo del clúster. Tendrás que definir `DOZZLE_MODE=k8s` y configurar RBAC para acceder a los logs de los pods.

Consulta [modo Kubernetes](/es/guide/k8s) para ver la configuración completa, incluidos los manifiestos de RBAC, deployment y service.
