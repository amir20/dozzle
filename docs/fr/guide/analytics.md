---
title: Statistiques anonymes
sourceHash: 0fedd8c524b8
---

# Collecte de données statistiques

Dozzle collecte des données d'utilisation anonymes via une balise légère, afin de prioriser les fonctionnalités et les correctifs. C'est un projet open source sans financement, ces données sont donc le principal indicateur pour savoir où investir les efforts.

## Ce qui est collecté

En résumé, la balise contient la version de Dozzle, le mode de déploiement (server, swarm, k8s, agent), le fournisseur d'authentification activé, quelques indicateurs de fonctionnalités, la version du moteur Docker et de petits compteurs (nombre d'hôtes, de conteneurs, de filtres). Un identifiant aléatoire par installation est inclus pour la déduplication.

Aucun contenu de log, nom de conteneur, nom d'image, adresse IP ou identifiant utilisateur n'est jamais transmis. L'ensemble exact des champs évolue avec le temps. La source de référence est [`types/beacon.go`](https://github.com/amir20/dozzle/blob/master/types/beacon.go), et l'envoi se fait depuis [`internal/analytics/http_beacon.go`](https://github.com/amir20/dozzle/blob/master/internal/analytics/http_beacon.go).

## Où les données sont stockées

Les évènements sont envoyés à `https://b.dozzle.dev/event`, un petit service Go qui les écrit dans un fichier plat sur DigitalOcean pour traitement ultérieur.

## Se désinscrire

Passez `--no-analytics` ou définissez `DOZZLE_NO_ANALYTICS=true`. Aucune requête de balise ne sera envoyée.

```yaml
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      DOZZLE_NO_ANALYTICS: "true"
```
