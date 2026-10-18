---
title: Dozzle vs docker logs
description: Ce que Dozzle apporte par rapport à docker logs et aux outils en terminal, et quand le terminal suffit.
sourceHash: 6b766c4db7ff
---

# Dozzle vs `docker logs`

Dozzle lit les mêmes logs que `docker logs`. Rien n'est copié ni stocké, il n'affiche donc jamais rien que la commande ne pourrait pas afficher. Ce qui change, c'est la façon de les lire : dans un navigateur, sur plusieurs conteneurs et plusieurs hôtes, sans shell sur le serveur.

## En un coup d'œil

|                      | Dozzle                                                                                               | `docker logs`                                        |
| -------------------- | ---------------------------------------------------------------------------------------------------- | ---------------------------------------------------- |
| Où les lire          | Un navigateur, depuis n'importe quel appareil                                                        | Un shell sur l'hôte Docker                           |
| Plusieurs conteneurs | N'importe quel ensemble de conteneurs dans une seule vue fusionnée, ou toute une stack ou un service | Un conteneur. `docker compose logs` couvre un projet |
| Plusieurs hôtes      | Une seule interface pour tous les hôtes, via les [agents](/fr/guide/agent)                           | Un hôte par shell, ou un `DOCKER_HOST` par commande  |
| Recherche            | Recherche à la frappe, avec expressions régulières                                                   | Redirection vers `grep`                              |
| Logs JSON            | Formatés et colorés, [SQL](/fr/guide/sql-engine) sur les champs                                      | Lignes brutes, redirection vers `jq`                 |
| Traces d'appels      | Entrées multilignes regroupées                                                                       | Une ligne à la fois                                  |
| CPU et mémoire       | Graphiques à côté des logs                                                                           | `docker stats` dans un autre shell                   |
| Partage              | Un lien vers un conteneur, derrière une [authentification](/fr/guide/authentication)                 | Copier-coller                                        |
| Installation         | Un conteneur                                                                                         | Déjà là                                              |

## Quand le terminal suffit

Si vous êtes déjà sur l'hôte et voulez les cent dernières lignes d'un conteneur, `docker logs --tail 100 -f <name>` est le plus rapide. Pour une vue en direct de tous les conteneurs dans un terminal, [dtop](/fr/guide/dtop) les affiche dans un tableau façon `docker ps` et ouvre Dozzle quand vous avez besoin des logs complets.

## Quand Dozzle convient mieux

Utilisez Dozzle quand plus d'une personne lit les logs, quand vous préférez ne pas distribuer d'accès SSH pour les lire, ou quand vous suivez une requête à travers plusieurs conteneurs à la fois. C'est aussi le moyen le plus simple de lire des logs depuis un téléphone.

## Essayer

```sh
docker run -d -v /var/run/docker.sock:/var/run/docker.sock -v dozzle_data:/data -p 8080:8080 amir20/dozzle:latest
```

Ouvrez ensuite `http://localhost:8080`. D'autres options se trouvent dans [Démarrage](/fr/guide/getting-started).
