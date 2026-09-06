---
title: MCP-Integration
sourceHash: 07d02a3201c5
---

# MCP-Integration

<Badge type="tip" text="Docker" />
<Badge type="tip" text="Swarm" />

Dozzle unterstützt das [Model Context Protocol (MCP)](https://modelcontextprotocol.io/), damit KI-Coding-Assistenten mit deinen Docker-Containern arbeiten können. Ist es aktiviert, stellt Dozzle unter `/api/mcp` einen MCP-Endpunkt über den Streamable-HTTP-Transport bereit, direkt aus demselben Container heraus — keine zusätzlichen Prozesse oder Sidecars nötig.

Diese Funktion ist standardmäßig **deaktiviert**. Setze zum Aktivieren die Option `--enable-mcp` oder die Umgebungsvariable `DOZZLE_ENABLE_MCP` auf `true`.

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

## Verfügbare Tools

Alle Tools sind **schreibgeschützt** und verändern keine Container.

| Tool                    | Beschreibung                                                                                     |
| ----------------------- | ------------------------------------------------------------------------------------------------ |
| `list_containers`       | Listet alle Container über alle Hosts hinweg. Unterstützt einen optionalen `state`-Filter.       |
| `get_container_logs`    | Holt strukturierte Logs mit erkannten Leveln, JSON-Parsing und mehrzeiliger Gruppierung.         |
| `search_container_logs` | Durchsucht Container-Logs nach einem Stichwort oder einer Phrase. Liefert nur passende Einträge. |
| `list_hosts`            | Listet alle verbundenen Docker-Hosts.                                                            |
| `get_container_stats`   | Liefert den Verlauf von CPU- und Speichernutzung eines Containers.                               |

## MCP-Clients konfigurieren

### VS Code (GitHub Copilot / Copilot Chat)

Füge Folgendes in deine `.vscode/mcp.json` oder deine MCP-Benutzereinstellungen ein:

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

Füge Folgendes in deine MCP-Konfiguration für Claude Desktop ein:

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
> Ersetze `localhost:8080` durch die Adresse deiner Dozzle-Instanz. Wenn Dozzle mit einem eigenen Basispfad konfiguriert ist (z. B. `--base /dozzle`), liegt der MCP-Endpunkt unter `/dozzle/api/mcp`.

## Authentifizierung

Der MCP-Endpunkt gehört zur authentifizierten API-Gruppe. Ist die Authentifizierung aktiv, müssen MCP-Clients gültige Zugangsdaten mitliefern.

### Simple Auth

Mit `--auth-provider simple` müssen MCP-Clients ein gültiges JWT-Token im Header `Authorization` mitschicken. So bekommst du ein Token:

1. Sende eine `POST`-Anfrage an `/api/token` mit deinem Benutzernamen und Passwort.
2. Konfiguriere deinen MCP-Client so, dass er das Token als Bearer-Header sendet.

Zum Beispiel in den MCP-Einstellungen von VS Code:

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

### Forward-Proxy-Authentifizierung

Mit `--auth-provider forward-proxy` übernimmt der Reverse Proxy vor Dozzle die Authentifizierung und setzt die passenden Header. MCP-Clients sollten über denselben Proxy verbinden, die Authentifizierung passiert dann transparent.

### Keine Authentifizierung

Ist kein Authentifizierungsanbieter konfiguriert (Standard), ist der MCP-Endpunkt öffentlich erreichbar. Weitere Konfiguration ist nicht nötig.
