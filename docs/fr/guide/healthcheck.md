---
title: Healthcheck
sourceHash: 4ab4dd21a26a
---

# Activer le healthcheck

Dozzle embarque une sous-commande `dozzle healthcheck`. Elle n'est pas activée par défaut dans l'image car elle ajoute une légère charge CPU. Activez-la depuis votre fichier compose :

```yaml
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    ports:
      - 8080:8080
    healthcheck:
      test: ["CMD", "/dozzle", "healthcheck"]
      interval: 3s
      timeout: 30s
      retries: 5
      start_period: 30s
```

## Ce qui est vérifié

En mode serveur, `dozzle healthcheck` envoie une requête HTTP `GET` sur son propre point d'accès `/healthcheck`. Celui-ci teste chaque client Docker **local** (jusqu'à 3 s par client) et renvoie :

- `200 OK` : au moins un client Docker local a répondu, **ou** aucun client local n'est configuré mais au moins un hôte agent distant est connu.
- `500 Internal Server Error` : tous les clients locaux ont échoué et aucun hôte agent n'est connu.

Les agents distants sont volontairement **exclus** du healthcheck du serveur : un agent injoignable ne doit pas rendre le processus Dozzle principal non sain. Chaque agent peut exposer son propre healthcheck, voir [Healthcheck d'un agent](/fr/guide/agent#setting-up-healthcheck).

## Codes de sortie

- `0` : sain (HTTP 200)
- différent de zéro : non sain, erreur réseau ou réponse autre que 200. L'URL en échec et le statut sont écrits sur stdout.

La commande respecte `--addr` et `--base`, elle fonctionne donc avec des ports et des chemins de base personnalisés sans configuration supplémentaire.

> [!WARNING]
> La commande `healthcheck` ne fonctionne pas avec l'option `--health-cmd` à cause d'un bug dans Docker. Utilisez le bloc `healthcheck` de `docker-compose.yml` comme ci-dessus. Voir [docker/cli#3719](https://github.com/docker/cli/issues/3719) pour les détails.
