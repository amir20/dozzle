---
title: Offres et limites
sourceHash: be1bd7a795e4
---

# Offres et limites

Les offres, les prix et les quotas sont sur la [page des tarifs de Dozzle Cloud](https://cloud.dozzle.dev/pricing), qui est toujours à jour. Cette page décrit à quoi ressemblent les limites de votre côté quand vous les atteignez.

## Les alertes sont soudain brutes et répétitives

Vous avez très probablement dépassé votre quota mensuel d'évènements. Rien ne casse : l'historique des évènements continue d'être enregistré, mais le triage se met en pause, environ un évènement sur dix passe en alerte brute, et les répétitions ne sont plus fusionnées en une alerte avec compteur. Vérifiez d'abord la page d'utilisation dans Cloud. Les quotas se réinitialisent au début de chaque mois, et cela vaut aussi pour les offres payantes.

## Une recherche ne renvoie rien de la semaine dernière

La recherche ne remonte pas plus loin que la fenêtre de rétention de votre offre. Tout ce qui est plus ancien a déjà été supprimé, même si l'évènement a bien eu lieu. Demander une fenêtre de métriques plus longue que ce que permet votre offre renvoie la fenêtre dont vous disposez réellement, pas une erreur.

## La limite d'instances

L'offre gratuite relie une instance à la fois. En relier une deuxième affiche un message de limite. Vous avez deux options :

- **Déplacer la place.** Supprimez la clé d'API de l'instance existante sur la page Instances, puis reliez la nouvelle. C'est définitif pour l'ancienne instance : son historique reste, mais il faudrait la relier de zéro.
- **Passer à une offre supérieure** pour garder les deux connectées en même temps.

## Vérifier votre consommation

La page d'utilisation dans Cloud affiche les évènements, les octets de logs et les conversations avec l'assistant consommés ce mois-ci par rapport à votre quota. Vous pouvez aussi demander dans le chat : « combien ai-je consommé ce mois-ci ? ».

## Changer ou annuler

Passez à une offre supérieure depuis la page des tarifs ou depuis les réglages. La facturation passe par Stripe ; moyens de paiement, factures et reçus s'y gèrent via le lien de facturation dans vos réglages.

L'annulation arrête les prélèvements futurs et vous bascule sur l'offre gratuite à la fin de la période déjà payée. Votre compte et votre historique restent.
