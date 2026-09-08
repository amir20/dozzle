---
title: Vos données
sourceHash: f11dd8e45cb5
---

# Vos données

Ce qui quitte votre hôte, comment l'arrêter, et ce que [Dozzle Cloud](/fr/guide/dozzle-cloud) stocke une fois que c'est arrivé.

## Relier n'expose pas votre Dozzle

Votre Dozzle ouvre une connexion **sortante** vers Cloud. Rien d'entrant n'est ouvert, aucun port n'est redirigé, et Cloud ne peut atteindre votre instance que par la connexion que celle-ci a initiée. Si vous dissociez, cet accès prend fin immédiatement.

Relier n'ajoute pas non plus d'authentification à votre Dozzle auto-hébergé. C'est une question distincte, et importante : **par défaut, Dozzle n'a pas de connexion.** Quiconque peut l'atteindre sur votre réseau peut voir vos logs. Si vous avez exposé Dozzle sur internet ou partagez votre réseau, configurez l'[Authentification](/fr/guide/authentication) sur l'instance elle-même. Cela s'applique que vous reliiez ou non.

Les actions sur les conteneurs — démarrer, arrêter, redémarrer — restent refusées par votre instance tant que vous ne les activez pas avec `DOZZLE_ENABLE_ACTIONS`. Voir [Actions](/fr/guide/actions).

## Contrôler ce qui est transmis

Le contrôle de confidentialité le plus efficace consiste à ne pas envoyer la chose du tout. Par défaut, chaque conteneur en cours d'exécution envoie ses logs à Cloud tant que l'instance est reliée. Pour les conteneurs dont le bavardage en niveau info n'a aucune valeur diagnostique, ou qui manipulent des données que vous préférez garder sur l'hôte, filtrez ou désactivez avec un label.

### `dev.dozzle.cloud.min_level`

| Valeur                                        | Effet                                                                                                         |
| --------------------------------------------- | ------------------------------------------------------------------------------------------------------------- |
| _(non défini)_                                | Toutes les lignes de log sont transmises. Valeur par défaut.                                                  |
| `disabled`                                    | Le conteneur est totalement ignoré. Aucun log n'est transmis à Cloud.                                         |
| `trace`                                       | Identique à non défini, puisque trace est le niveau le plus bas. Tout est transmis.                           |
| `debug` / `info` / `warn` / `error` / `fatal` | Seules les lignes de ce niveau ou supérieur sont transmises. Les lignes sans niveau détecté passent toujours. |

Une valeur non reconnue (une faute de frappe comme `warning` ou `wran`) est enregistrée comme erreur et ignorée, le conteneur transmet donc tout comme si le label n'existait pas.

Le label est lu au démarrage du lecteur de logs. Le modifier sur un conteneur en cours d'exécution ne prend effet qu'après son redémarrage.

```yaml
services:
  zigbee2mqtt:
    image: koenkk/zigbee2mqtt
    labels:
      # Ne transmettre que warn/error/fatal à Dozzle Cloud
      - dev.dozzle.cloud.min_level=warn

  noisy-debug-tool:
    image: example/debug
    labels:
      # Ne rien envoyer depuis ce conteneur
      - dev.dozzle.cloud.min_level=disabled
```

Le filtre s'exécute sur votre instance Dozzle **avant que les logs ne quittent l'hôte**, donc les lignes écartées ne touchent jamais le réseau et ne comptent jamais dans votre quota. La consultation locale des logs dans Dozzle n'est pas affectée.

## Ce que Cloud stocke

- **Les lignes de log** transmises par vos instances reliées, pour la recherche plein texte.
- **Les évènements et alertes** qui ont correspondu à vos règles, avec leurs enquêtes et constats.
- **Les métadonnées de conteneurs et d'hôtes** — noms, images, états, consommation de ressources.
- **Votre compte** — adresse e-mail, offre, réglages des canaux de notification.
- **L'historique de chat** avec l'agent.

Tout est cloisonné à votre compte ; les autres utilisateurs ne voient pas vos données. Les données stockées sont conservées pendant la fenêtre de rétention de votre offre puis supprimées automatiquement. Voir [Offres et limites](/fr/guide/dozzle-cloud/plans).

## Clés d'API

Chaque instance reliée s'authentifie avec sa propre clé d'API. Les clés sont hachées avec BLAKE2b, gèrent l'expiration et ne sont jamais stockées en clair.

Supprimer une clé sur la page Instances déconnecte cette instance immédiatement et définitivement. La clé ne peut être ni récupérée ni réattachée : reliez l'instance à nouveau pour en obtenir une nouvelle. Si vous pensez qu'une clé a fuité, supprimez-la et reliez à nouveau. C'est le remède complet : l'ancienne clé cesse de fonctionner à l'instant où elle est supprimée.

## Connexion

Cloud utilise la connexion GitHub ou Google. Aucun mot de passe distinct à créer, et Cloud ne voit jamais votre mot de passe GitHub ou Google. Si vous vous inscrivez avec un fournisseur et vous connectez ensuite avec l'autre en utilisant la même adresse e-mail, vous arrivez sur le même compte.

## Arrêter la collecte sans fermer le compte

1. Supprimez les clés d'API de vos instances sur la page Instances. La transmission s'arrête immédiatement.
2. Désactivez vos canaux sur la page Channels, pour que rien ne soit livré.

L'historique existant expire ensuite de lui-même dans la fenêtre de rétention de votre offre.

## Fermer votre compte

Il n'y a pas encore de bouton de suppression en libre-service. Écrivez à **amir@dozzle.dev** depuis l'adresse du compte et demandez la suppression. Si vous êtes sur une offre payante, annulez-la d'abord depuis les réglages pour ne plus être facturé.

Avant cela, ou à la place, les étapes ci-dessus permettent de retirer l'essentiel vous-même : supprimer vos clés d'API arrête toute collecte, et les données stockées expirent avec la rétention.
