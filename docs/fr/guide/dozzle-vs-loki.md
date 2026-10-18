---
title: Dozzle vs Grafana Loki
description: Faut-il Grafana et Loki pour lire les logs Docker, ou Dozzle suffit-il, et comment utiliser les deux.
sourceHash: b561d1507aca
---

# Dozzle vs Grafana Loki

Loki est un stockage de logs. Un collecteur y envoie chaque ligne de log, Loki les conserve aussi longtemps que vous le configurez, et Grafana est l'endroit où vous les interrogez. Dozzle ne stocke rien : il lit les logs directement depuis Docker, les mêmes que ceux qu'affiche `docker logs`, et les diffuse dans votre navigateur.

La vraie question est donc de savoir si vous devez conserver les logs ailleurs que dans Docker.

## En un coup d'œil

|                            | Dozzle                                                                                 | Grafana + Loki                                             |
| -------------------------- | -------------------------------------------------------------------------------------- | ---------------------------------------------------------- |
| Ce que vous faites tourner | Un conteneur                                                                           | Loki, Grafana et un collecteur comme Grafana Alloy         |
| Où vivent les logs         | Dans Docker, tels qu'il les conserve déjà                                              | Copiés dans le stockage propre à Loki                      |
| Jusqu'où remonter          | Aussi loin que la rotation des logs de Docker les conserve                             | Aussi longtemps que la rétention configurée dans Loki      |
| Suivi en direct            | La vue par défaut                                                                      | Disponible dans Grafana Explore                            |
| Requêtes                   | Recherche, expressions régulières, [SQL](/fr/guide/sql-engine) sur le JSON             | LogQL, y compris des métriques calculées à partir des logs |
| Tableaux de bord           | CPU et mémoire des conteneurs                                                          | Tout tableau de bord que vous construisez dans Grafana     |
| Alertes                    | [Alertes](/fr/guide/alerts-and-webhooks) sur les logs, les métriques et les évènements | Alertes Grafana sur des requêtes LogQL                     |
| Logs hors de Docker        | Non                                                                                    | Oui : fichiers, journal systemd, Kubernetes et plus encore |

## Quand choisir Loki

Choisissez Loki quand vous avez besoin des logs après que Docker les a supprimés par rotation, quand un conteneur est supprimé et que ses logs doivent rester, ou quand vous devez chercher dans des semaines d'historique. C'est aussi le meilleur choix si vous utilisez déjà Grafana et voulez les logs à côté de vos métriques, ou si vous avez besoin de logs provenant d'autre chose que des conteneurs.

Prévoyez l'installation. Promtail, le collecteur utilisé par la plupart des anciens guides, est arrivé en fin de vie en mars 2026, donc les nouvelles installations utilisent Grafana Alloy ou le driver Docker de Loki.

## Quand Dozzle convient mieux

Choisissez Dozzle quand vous voulez voir ce que fait un conteneur maintenant, ou ce qu'il a fait ces dernières heures ou ces derniers jours, sans faire tourner une chaîne de collecte de logs. Pas de stockage à dimensionner, pas de rétention à régler, pas de langage de requête à apprendre. Il démarre en quelques secondes. Voir [Démarrage](/fr/guide/getting-started).

L'historique que Dozzle peut afficher dépend de la rotation des logs de Docker. Pour en garder davantage, augmentez les limites dans `/etc/docker/daemon.json` :

```json
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "50m",
    "max-file": "5"
  }
}
```

Les nouvelles limites s'appliquent aux conteneurs créés après le redémarrage de Docker.

## Utiliser les deux

Beaucoup d'installations font tourner Loki pour l'historique et Dozzle pour le débogage au quotidien. Ils ne se gênent pas : Dozzle lit depuis Docker, et Alloy lit depuis Docker. Si vous passez plutôt vos conteneurs au driver Docker de Loki, laissez activé le [dual logging](https://docs.docker.com/engine/logging/dual-logging/) de Docker (c'est le cas par défaut), pour que `docker logs`, et donc Dozzle, puissent toujours les lire.
