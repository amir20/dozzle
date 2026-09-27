---
title: Integración con MCP
sourceHash: fe1cee485b1f
---

# Integración con MCP

<Badge type="tip" text="Docker" />
<Badge type="tip" text="Swarm" />

Dozzle es compatible con el [Model Context Protocol (MCP)](https://modelcontextprotocol.io/), que permite a los asistentes de programación con IA interactuar con tus contenedores de Docker. Al activarlo, Dozzle expone un endpoint MCP en `/api/mcp` mediante el transporte Streamable HTTP, servido desde el mismo contenedor: no hacen falta procesos adicionales ni sidecars.

Esta función está **desactivada** por defecto. Para activarla, pon el flag `--enable-mcp` o la variable de entorno `DOZZLE_ENABLE_MCP` a `true`.

::: code-group

```sh [cli]
docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --enable-mcp
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    ports:
      - 8080:8080
    environment:
      DOZZLE_ENABLE_MCP: true
```

:::

## Herramientas disponibles

Todas las herramientas son de **solo lectura** y no modifican los contenedores.

| Herramienta             | Descripción                                                                                  |
| ----------------------- | -------------------------------------------------------------------------------------------- |
| `list_containers`       | Lista todos los contenedores de todos los hosts. Admite un filtro `state` opcional.          |
| `get_container_logs`    | Obtiene logs estructurados con niveles detectados, análisis de JSON y agrupación multilínea. |
| `search_container_logs` | Busca una palabra o frase en los logs de un contenedor. Devuelve solo las coincidencias.     |
| `list_hosts`            | Lista todos los hosts de Docker conectados.                                                  |
| `get_container_stats`   | Obtiene el historial de uso de CPU y memoria de un contenedor.                               |

## Configurar clientes MCP

### VS Code (GitHub Copilot / Copilot Chat)

Añade esto a tu `.vscode/mcp.json` o a la configuración MCP de usuario:

```json
{
  "servers": {
    "dozzle": {
      "type": "http",
      "url": "http://localhost:8080/api/mcp"
    }
  }
}
```

### Claude Desktop

Añade esto a la configuración MCP de Claude Desktop:

```json
{
  "mcpServers": {
    "dozzle": {
      "type": "streamable-http",
      "url": "http://localhost:8080/api/mcp"
    }
  }
}
```

> [!NOTE]
> Sustituye `localhost:8080` por la dirección de tu instancia de Dozzle. Si Dozzle usa una ruta base personalizada (por ejemplo, `--base /dozzle`), el endpoint MCP estará en `/dozzle/api/mcp`.

## Autenticación

El endpoint MCP forma parte del grupo de API autenticada. Cuando la autenticación está activada, los clientes MCP deben aportar credenciales válidas.

### Autenticación simple y OIDC

Con `--auth-provider simple` o `--auth-provider oidc`, Dozzle es un servidor de autorización OAuth para clientes MCP. Los clientes que admiten autorización MCP (VS Code, Claude Code, Claude Desktop y otros) inician sesión por su cuenta. Añade el servidor sin cabeceras y, la primera vez que el cliente se conecte, abrirá una pestaña del navegador:

1. Inicia sesión en Dozzle como siempre (contraseña, GitHub o tu proveedor OIDC).
2. Dozzle muestra una página de consentimiento con el nombre del cliente y la dirección a la que te devolverá. Elige **Permitir**.
3. El navegador vuelve al cliente, que guarda el token y lo renueva por sí solo.

El token solo funciona en `/api/mcp` y lleva los mismos roles y filtros de contenedores que tu sesión del navegador. Los tokens de acceso duran una hora. Los tokens de refresco dejan de funcionar 30 días después de aprobar el cliente, que entonces te pide aprobarlo de nuevo. Todo lo que cierra la sesión de todos en Dozzle, como editar `users.yml` o cambiar el emisor OIDC, también revoca los tokens MCP.

> [!NOTE]
> Dozzle construye sus URL de OAuth a partir de la petición, igual que el callback de OIDC. Detrás de un proxy inverso, reenvía `Host` (o `X-Forwarded-Host`) y `X-Forwarded-Proto`. Con una ruta base personalizada, algunos clientes buscan metadatos en `/.well-known/` en la raíz del dominio, así que dirige `/.well-known/` a Dozzle si puedes.

#### Clientes sin soporte de OAuth

Con autenticación simple, un cliente que solo puede enviar cabeceras fijas puede usar un token de sesión:

1. Envía una petición `POST` a `/api/token` con tu usuario y contraseña.
2. Configura tu cliente MCP para enviar el token como cabecera Bearer.

Por ejemplo, en la configuración MCP de VS Code:

```json
{
  "servers": {
    "dozzle": {
      "type": "http",
      "url": "http://localhost:8080/api/mcp",
      "headers": {
        "Authorization": "Bearer <your-jwt-token>"
      }
    }
  }
}
```

Con oidc esto no está disponible, porque no hay contraseña que cambiar por un token.

### Autenticación por proxy

Con `--auth-provider forward-proxy`, el proxy inverso que hay delante de Dozzle se encarga de la autenticación e inyecta las cabeceras correspondientes. Los clientes MCP deben conectarse a través de ese mismo proxy y la autenticación se resuelve de forma transparente.

### Sin autenticación

Si no hay ningún proveedor de autenticación configurado (lo predeterminado), el endpoint MCP es de acceso público. No hace falta configurar nada más.
