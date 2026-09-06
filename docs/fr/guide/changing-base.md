---
title: Reverse proxy et chemin de base
sourceHash: f344ea2a42cc
---

# Reverse proxy et chemin de base

Dozzle est souvent placé derrière un reverse proxy pour la terminaison TLS, l'authentification, ou pour partager un nom d'hôte avec d'autres services. Cette page couvre à la fois le montage de Dozzle sur un sous-chemin et les réglages de proxy nécessaires au bon fonctionnement du streaming.

## Changer le chemin de base

Par défaut, Dozzle est monté sur `/`. Cela peut être modifié avec l'option `--base` ou la variable d'environnement `DOZZLE_BASE`. Par exemple, pour le monter sur `/foobar` :

::: code-group

```sh
docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --base /foobar
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
      DOZZLE_BASE: /foobar
```

:::

Dozzle sera accessible sur `http://localhost:8080/foobar/`. Cette option réécrit toutes les ressources vers `/foobar/{file.path}` et redirige automatiquement `/foobar` vers `/foobar/`.

## Prérequis du proxy

Dozzle diffuse les logs via **Server-Sent Events (SSE)** et utilise **WebSocket** pour les shells de conteneurs. Les reverse proxies doivent :

1. **Désactiver la mise en tampon des réponses** : SSE envoie les événements au fil de l'eau. Toute mise en tampon fait arriver les logs par à-coups, voire jamais. Dozzle envoie `X-Accel-Buffering: no`, mais certains proxies l'ignorent.
2. **Transmettre les en-têtes d'upgrade WebSocket** : requis pour les fonctions shell et attach.
3. **Éviter de compresser `text/event-stream`** : les middlewares de compression cassent souvent le SSE.

## Nginx

```nginx
location ^~ /foobar/ {
    proxy_pass http://dozzle:8080;

    chunked_transfer_encoding off;
    proxy_buffering off;
    proxy_cache off;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
}
```

Supprimez le préfixe `^~ /foobar/` si Dozzle est monté à la racine. Voir aussi l'entrée de la FAQ sur la [désactivation de la mise en tampon](/fr/guide/faq#disabling-buffering-in-nginx).

## Traefik

Traefik gère les upgrades WebSocket automatiquement, mais le middleware `compress` par défaut casse le SSE. Excluez `text/event-stream` :

```yaml
http:
  middlewares:
    middlewares-compress:
      compress:
        excludedContentTypes:
          - text/event-stream
```

Voici ensuite un bloc de labels typique sur le service Dozzle :

```yaml
services:
  dozzle:
    image: amir20/dozzle:latest
    labels:
      - traefik.enable=true
      - traefik.http.routers.dozzle.rule=Host(`dozzle.example.com`)
      - traefik.http.routers.dozzle.entrypoints=websecure
      - traefik.http.routers.dozzle.tls.certresolver=letsencrypt
      - traefik.http.services.dozzle.loadbalancer.server.port=8080
```

## Caddy

```caddyfile
dozzle.example.com {
    reverse_proxy dozzle:8080 {
        flush_interval -1
    }
}
```

`flush_interval -1` désactive la mise en tampon des réponses pour les endpoints de streaming.

## Pièges courants

- **Page blanche ou ressources en 404 avec `--base`** : le proxy retire le préfixe de chemin avant de transmettre la requête. Configurez-le pour passer le chemin complet à Dozzle.
- **Les logs s'arrêtent après quelques secondes** : les délais d'expiration de connexion du proxy sont trop courts. Augmentez les délais de lecture/envoi à plusieurs minutes au moins (par ex. Nginx `proxy_read_timeout 3600s`).
- **Le shell se déconnecte immédiatement** : les en-têtes d'upgrade WebSocket ne sont pas transmis. Vérifiez les en-têtes `Upgrade` et `Connection`.
