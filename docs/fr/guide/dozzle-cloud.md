---
title: Dozzle Cloud
sourceHash: ddf1502ea23d
---

# Dozzle Cloud

[Dozzle Cloud](https://cloud.dozzle.dev) est un compagnon managé optionnel de Dozzle auto-hébergé. Dozzle reste entièrement open source et auto-hébergé ; Cloud se pose par-dessus et prend en charge la partie vraiment difficile à faire tourner soi-même : décider ce qui mérite de vous réveiller, et déterminer ce qui a réellement cassé.

Votre Dozzle ouvre une connexion sortante vers Cloud. Aucun port entrant, aucune IP publique, aucun agent à installer.

L'offre gratuite est toute la couche d'alerting : chaque alerte déclenchée par vos règles Dozzle est triée en un message lisible, les répétitions sont fusionnées en une alerte avec compteur, et elle part par e-mail, Telegram, Discord, Slack, ntfy, webhook ou notification push du navigateur. Les offres payantes ajoutent la moitié proactive : Cloud lit vos logs chaque matin et signale les problèmes sur lesquels rien n'a jamais alerté, avec un historique plus long et davantage d'instances.

Le tour complet est sur [Fonctionnalités](https://cloud.dozzle.dev/features), et ce que contient chaque offre sur [Tarifs](https://cloud.dozzle.dev/pricing).

## Pour aller plus loin

| Page                                                       | Ce qu'elle couvre                                                                        |
| ---------------------------------------------------------- | ---------------------------------------------------------------------------------------- |
| [Relier votre instance](/fr/guide/dozzle-cloud/connecting) | Le lien, pourquoi ni IP publique ni port ouvert ne sont nécessaires, pare-feu, dépannage |
| [Dans votre Dozzle](/fr/guide/dozzle-cloud/in-dozzle)      | Le rail cloud, les alertes qui survivent au rechargement, et ce qui marche sans Cloud    |
| [Canaux de notification](/fr/guide/dozzle-cloud/channels)  | Tous les canaux, comment configurer chacun, et comment faire moins de bruit              |
| [Offres et limites](/fr/guide/dozzle-cloud/plans)          | Atteindre une limite, la limite d'instances, consommation, annulation                    |
| [Vos données](/fr/guide/dozzle-cloud/your-data)            | Ce qui quitte votre hôte, comment l'arrêter, ce que Cloud stocke, les clés d'API         |

Les règles d'alerte se configurent sur votre propre instance, pas dans Cloud. Voir [Alertes](/fr/guide/alerts-and-webhooks).

## Retours

Dozzle Cloud est construit par la même personne que Dozzle, et l'exigence est la même : des choses que les gens ont vraiment envie d'utiliser. Si vous l'essayez et que quelque chose vous semble bancal, manquant ou franchement utile, [ouvrez une discussion](https://github.com/amir20/dozzle/discussions). Ces retours orientent ce qui sera construit ensuite.
