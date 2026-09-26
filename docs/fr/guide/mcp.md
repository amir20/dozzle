---
title: Intégration MCP
sourceHash: fe1cee485b1f
---

# Intégration MCP

<Badge type="tip" text="Docker" />
<Badge type="tip" text="Swarm" />

Dozzle prend en charge le [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) afin de permettre aux assistants de code IA d'interagir avec vos conteneurs Docker. Une fois activé, Dozzle expose un endpoint MCP sur `/api/mcp` en utilisant le transport Streamable HTTP, servi depuis le même conteneur : aucun processus ni sidecar supplémentaire n'est nécessaire.

Cette fonctionnalité est **désactivée** par défaut. Pour l'activer, mettez l'option `--enable-mcp` ou la variable d'environnement `DOZZLE_ENABLE_MCP` à `true`.

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

## Outils disponibles

Tous les outils sont en **lecture seule** et ne modifient pas les conteneurs.

| Outil                   | Description                                                                                          |
| ----------------------- | ---------------------------------------------------------------------------------------------------- |
| `list_containers`       | Liste tous les conteneurs de tous les hôtes. Accepte un filtre `state` facultatif.                   |
| `get_container_logs`    | Récupère des logs structurés avec détection des niveaux, analyse JSON et regroupement multiligne.    |
| `search_container_logs` | Recherche un mot-clé ou une phrase dans les logs d'un conteneur. Ne renvoie que les correspondances. |
| `list_hosts`            | Liste tous les hôtes Docker connectés.                                                               |
| `get_container_stats`   | Récupère l'historique d'utilisation CPU et mémoire d'un conteneur.                                   |

## Configurer les clients MCP

### VS Code (GitHub Copilot / Copilot Chat)

Ajoutez ce qui suit à votre `.vscode/mcp.json` ou à vos paramètres MCP utilisateur :

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

Ajoutez ce qui suit à votre configuration MCP de Claude Desktop :

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
> Remplacez `localhost:8080` par l'adresse de votre instance Dozzle. Si Dozzle est configuré avec un chemin de base personnalisé (par ex. `--base /dozzle`), l'endpoint MCP se trouvera sur `/dozzle/api/mcp`.

## Authentification

L'endpoint MCP fait partie du groupe d'API authentifiées. Lorsque l'authentification est activée, les clients MCP doivent fournir des identifiants valides.

### Authentification simple et OIDC

Avec `--auth-provider simple` ou `--auth-provider oidc`, Dozzle est un serveur d'autorisation OAuth pour les clients MCP. Les clients qui prennent en charge l'autorisation MCP (VS Code, Claude Code, Claude Desktop et d'autres) se connectent d'eux-mêmes. Ajoutez le serveur sans en-tête : à la première connexion, le client ouvre un onglet du navigateur.

1. Connectez-vous à Dozzle comme d'habitude (mot de passe, GitHub ou votre fournisseur OIDC).
2. Dozzle affiche une page de consentement avec le nom du client et l'adresse vers laquelle vous serez renvoyé. Choisissez **Autoriser**.
3. Le navigateur revient au client, qui enregistre le jeton et le renouvelle tout seul.

Le jeton ne fonctionne que sur `/api/mcp` et porte les mêmes rôles et filtres de conteneurs que votre session navigateur. Les jetons d'accès durent une heure. Les jetons de rafraîchissement expirent 30 jours après l'approbation du client, qui vous demande alors de l'approuver à nouveau. Tout ce qui déconnecte tout le monde de Dozzle, comme une modification de `users.yml` ou de l'émetteur OIDC, révoque aussi les jetons MCP.

> [!NOTE]
> Dozzle construit ses URL OAuth à partir de la requête, comme pour le callback OIDC. Derrière un reverse proxy, transmettez `Host` (ou `X-Forwarded-Host`) et `X-Forwarded-Proto`. Avec un chemin de base personnalisé, certains clients cherchent les métadonnées sous `/.well-known/` à la racine du domaine : redirigez `/.well-known/` vers Dozzle si possible.

#### Clients sans prise en charge d'OAuth

Avec l'authentification simple, un client qui ne sait envoyer que des en-têtes fixes peut utiliser un jeton de session à la place :

1. Envoyez une requête `POST` à `/api/token` avec votre nom d'utilisateur et votre mot de passe.
2. Configurez votre client MCP pour envoyer le jeton dans un en-tête Bearer.

Par exemple, dans les paramètres MCP de VS Code :

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

Ce n'est pas possible avec oidc, car il n'y a pas de mot de passe à échanger contre un jeton.

### Authentification par proxy

Avec `--auth-provider forward-proxy`, le reverse proxy placé devant Dozzle gère l'authentification et injecte les en-têtes appropriés. Les clients MCP doivent se connecter à travers ce même proxy, et l'authentification est alors transparente.

### Sans authentification

Sans fournisseur d'authentification configuré (le cas par défaut), l'endpoint MCP est accessible publiquement. Aucune configuration supplémentaire n'est nécessaire.
