---
title: Dozzle vs Portainer
description: Ce qui distingue Dozzle et Portainer pour lire les logs Docker, et comment les faire tourner côte à côte.
sourceHash: 676bc9bd86c4
---

# Dozzle vs Portainer

Portainer est conçu pour gérer les conteneurs : stacks, volumes, réseaux, images et utilisateurs. Dozzle est conçu pour lire leurs logs. Portainer a un visualiseur de logs, et Dozzle peut démarrer, arrêter et mettre à jour des conteneurs, mais chacun est centré sur un travail différent. Beaucoup de gens utilisent les deux.

## En un coup d'œil

|                       | Dozzle                                                                                                                      | Portainer                                                        |
| --------------------- | --------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------- |
| Rôle principal        | Lire les logs                                                                                                               | Gérer les conteneurs, stacks, volumes, réseaux et images         |
| Logs en direct        | Diffusés à mesure que les lignes arrivent                                                                                   | Rafraîchissement automatique                                     |
| Plusieurs conteneurs  | Une stack, un service ou un groupe personnalisé dans une seule vue fusionnée                                                | Un conteneur à la fois                                           |
| Logs JSON             | Détectés, formatés et colorés. [SQL](/fr/guide/sql-engine) sur les champs JSON                                              | Affichés en texte                                                |
| Recherche             | Recherche et filtrage, y compris par expressions régulières                                                                 | Recherche et filtrage par texte                                  |
| CPU et mémoire        | Graphiques en direct avec l'historique récent                                                                               | Statistiques en direct                                           |
| Alertes               | Sur les lignes de log, le CPU ou la mémoire, et les évènements de conteneurs. Voir [Alertes](/fr/guide/alerts-and-webhooks) | Absentes du visualiseur de logs                                  |
| Mises à jour d'images | Vérifie les nouvelles images et met à jour selon un planning. Voir [Mises à jour](/fr/guide/updates)                        | Pull et redéploiement à la main                                  |
| Plusieurs hôtes       | [Agents](/fr/guide/agent), [Swarm](/fr/guide/swarm-mode), [Kubernetes](/fr/guide/k8s)                                       | Environnements                                                   |
| Installation          | Un conteneur, rien à configurer                                                                                             | Un conteneur, plus un compte administrateur au premier démarrage |

## Quand choisir Portainer

Choisissez Portainer si vous voulez créer et modifier des stacks depuis un navigateur, gérer volumes, réseaux et images, ou donner à une équipe l'accès à certains environnements et pas à d'autres. Dozzle n'essaie de faire rien de tout cela.

## Quand Dozzle convient mieux

Choisissez Dozzle quand la question est « que fait ce conteneur en ce moment ? ». Il s'ouvre directement sur les logs, suit plusieurs conteneurs à la fois et rend le JSON lisible sans aucune configuration. Il est aussi en lecture seule par défaut : les [actions](/fr/guide/actions) et l'[accès shell](/fr/guide/shell) sont désactivés jusqu'à ce que vous les activiez.

## Utiliser les deux

Les deux n'entrent pas en conflit. Tous deux lisent le socket Docker et tournent sur des ports différents :

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    ports:
      - 8080:8080
  portainer:
    image: portainer/portainer-ce:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - portainer-data:/data
    ports:
      - 9443:9443
volumes:
  portainer-data:
```

Pour passer de Portainer aux logs d'un conteneur dans Dozzle, voir [créer un lien vers un conteneur par son nom](/fr/guide/faq#nous-avons-des-outils-qui-utilisent-dozzle-a-la-creation-d-un-conteneur-comment-obtenir-un-lien-direct-vers-un-conteneur-par-son-nom).
