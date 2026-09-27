---
title: MCP-Integration
sourceHash: fe1cee485b1f
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

### Simple Auth und OIDC

Mit `--auth-provider simple` oder `--auth-provider oidc` ist Dozzle ein OAuth-Autorisierungsserver für MCP-Clients. Clients, die MCP-Autorisierung unterstützen (VS Code, Claude Code, Claude Desktop und andere), melden sich selbst an. Füge den Server ohne Header hinzu. Beim ersten Verbinden öffnet der Client einen Browser-Tab:

1. Melde dich wie gewohnt bei Dozzle an (Passwort, GitHub oder dein OIDC-Anbieter).
2. Dozzle zeigt eine Zustimmungsseite mit dem Namen des Clients und der Adresse, zu der du zurückgeleitet wirst. Wähle **Erlauben**.
3. Der Browser kehrt zum Client zurück, der das Token speichert und selbst erneuert.

Das Token funktioniert nur für `/api/mcp` und trägt dieselben Rollen und Container-Filter wie deine Browser-Sitzung. Access-Tokens gelten eine Stunde. Refresh-Tokens laufen 30 Tage nach der Freigabe des Clients ab, danach bittet der Client dich um eine erneute Freigabe. Alles, was alle Nutzer von Dozzle abmeldet, etwa eine Änderung an `users.yml` oder am OIDC-Issuer, widerruft auch die MCP-Tokens.

> [!NOTE]
> Dozzle baut seine OAuth-URLs aus der Anfrage, genau wie den OIDC-Callback. Hinter einem Reverse Proxy musst du `Host` (oder `X-Forwarded-Host`) und `X-Forwarded-Proto` weiterreichen. Mit einem eigenen Basispfad suchen manche Clients Metadaten unter `/.well-known/` im Root der Domain, leite `/.well-known/` also wenn möglich an Dozzle weiter.

#### Clients ohne OAuth-Unterstützung

Mit Simple Auth kann ein Client, der nur feste Header senden kann, stattdessen ein Sitzungstoken verwenden:

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

Mit oidc geht das nicht, weil es kein Passwort gibt, das gegen ein Token getauscht werden könnte.

### Forward-Proxy-Authentifizierung

Mit `--auth-provider forward-proxy` übernimmt der Reverse Proxy vor Dozzle die Authentifizierung und setzt die passenden Header. MCP-Clients sollten über denselben Proxy verbinden, die Authentifizierung passiert dann transparent.

### Keine Authentifizierung

Ist kein Authentifizierungsanbieter konfiguriert (Standard), ist der MCP-Endpunkt öffentlich erreichbar. Weitere Konfiguration ist nicht nötig.
