---
title: Integración con MCP
sourceHash: 07d02a3201c5
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

### Autenticación simple

Con `--auth-provider simple`, los clientes MCP tienen que incluir un token JWT válido en la cabecera `Authorization`. Para obtener un token:

1. Envía una petición `POST` a `/api/token` con tu usuario y contraseña.
2. Configura tu cliente MCP para que envíe el token como cabecera Bearer.

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

### Autenticación por proxy

Con `--auth-provider forward-proxy`, el proxy inverso que hay delante de Dozzle se encarga de la autenticación e inyecta las cabeceras correspondientes. Los clientes MCP deben conectarse a través de ese mismo proxy y la autenticación se resuelve de forma transparente.

### Sin autenticación

Si no hay ningún proveedor de autenticación configurado (lo predeterminado), el endpoint MCP es de acceso público. No hace falta configurar nada más.
