---
title: Débogage
sourceHash: 2cb9b9a633e2
---

# Déboguer avec les logs

Par défaut, Dozzle journalise au niveau `info`, volontairement discret. Quand quelque chose ne fonctionne pas, augmentez la verbosité avec l'option `--level` ou la variable d'environnement `DOZZLE_LEVEL`.

| Niveau  | Quand l'utiliser                                                                                        |
| ------- | ------------------------------------------------------------------------------------------------------- |
| `info`  | Par défaut. Détails de démarrage, erreurs et avertissements.                                            |
| `debug` | Diagnostics au niveau des requêtes, décisions d'authentification, connexions des agents, configuration. |
| `trace` | Tout. Évènements de log individuels, contenu des balises, trames gRPC. Très verbeux.                    |

Dozzle écrit tous ses logs sur `stdout`, donc `docker logs dozzle` est le bon endroit pour les lire.

```yaml
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    ports:
      - 8080:8080
    environment:
      DOZZLE_LEVEL: debug
```

## Signaler un bug

Si vous pensez avoir trouvé un bug, ouvrez une issue sur [github.com/amir20/dozzle/issues](https://github.com/amir20/dozzle/issues). Indiquez :

- La version de Dozzle (visible dans le pied de page de l'interface ou avec `dozzle --version`)
- Le mode de déploiement : server, swarm, k8s ou agent
- La version de Docker ou de Kubernetes
- Les logs pertinents au niveau `debug` ou `trace`
- Les étapes pour reproduire, idéalement avec un `docker-compose.yml` minimal

Plus le rapport initial contient de contexte, plus le tri sera rapide.
