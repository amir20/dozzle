---
title: Statistiques anonymes
sourceHash: 8421075e674a
---

# Collecte de données statistiques

Dozzle collecte des données d'utilisation anonymes via une balise légère, afin de prioriser les fonctionnalités et les correctifs. C'est un projet open source sans financement, ces données sont donc le principal indicateur pour savoir où investir les efforts.

## Ce qui est collecté

Dozzle envoie une balise au démarrage, puis une autre chaque fois que quelqu'un ouvre l'interface. Ensemble, elles contiennent :

- la version de Dozzle, le mode de déploiement (server, swarm, k8s, agent) et la version de Docker Engine
- le fournisseur d'authentification activé, et si les actions et le shell sont activés
- de petits compteurs : hôtes, agents, conteneurs en cours d'exécution et filtres
- la chaîne user agent du navigateur, uniquement dans la balise de l'interface
- l'identifiant de Docker Engine, pour ne pas compter deux fois la même installation

Aucun contenu de log, nom de conteneur, nom d'image, nom d'hôte ni identifiant d'utilisateur n'est jamais transmis. La liste exacte des champs évolue avec le temps. La source de référence est [`types/beacon.go`](https://github.com/amir20/dozzle/blob/master/types/beacon.go), et l'envoi se fait dans [`internal/analytics/http_beacon.go`](https://github.com/amir20/dozzle/blob/master/internal/analytics/http_beacon.go).

## Où les données sont stockées

Les balises sont envoyées à `https://b.dozzle.dev/event` et reçues par [drain](https://github.com/amir20/drain), un service Go open source qui les écrit dans une base de données et dans des fichiers Parquet pour l'analyse. drain ne conserve pas l'adresse IP d'où vient une balise, et les données ne sont partagées avec aucun tiers.

## Se désinscrire

Passez `--no-analytics` ou définissez `DOZZLE_NO_ANALYTICS=true`. Aucune requête de balise ne sera envoyée.

```yaml
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      DOZZLE_NO_ANALYTICS: "true"
```
