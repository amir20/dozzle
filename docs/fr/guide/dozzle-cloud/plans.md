---
title: Offres et limites
sourceHash: f7c6bbdc83ea
---

# Offres et limites

Ce que contient chaque offre, ce qui est décompté, et ce qui se passe en cas de dépassement.

L'offre gratuite est le produit d'alerting complet, pas son essai. Ce que les offres payantes achètent, c'est la moitié proactive (la revue qui lit vos logs et trouve ce sur quoi rien n'a jamais alerté) plus de la marge.

## Offres

|                                                          |   Gratuit    |        Pro        |       Team        |
| -------------------------------------------------------- | :----------: | :---------------: | :---------------: |
| Prix                                                     |     0 $      |    5 $ / mois     |    15 $ / mois    |
| Constats (lit vos logs, trouve ce qui n'a jamais alerté) | 1 / semaine  | Tous, chaque jour | Tous, chaque jour |
| Correctif fourni avec chaque constat                     |      —       |         ✓         |         ✓         |
| Le triage inspecte conteneurs et logs en cas de doute    |      —       |         ✓         |         ✓         |
| Enquête complète à la demande                            |      —       |         ✓         |         ✓         |
| Évènements traités par mois                              |    2 000     |        50K        |       250K        |
| Logs interrogeables                                      | 10 Go · 24 h |   50 Go · 30 j    |   100 Go · 30 j   |
| Historique des alertes et évènements                     |    1 jour    |     14 jours      |     30 jours      |
| Historique des métriques (CPU, mémoire, réseau, disque)  |     24 h     |     30 jours      |     30 jours      |
| Instances connectées                                     |      1       |    Illimitées     |    Illimitées     |
| Conversations avec l'assistant par mois                  |      10      |        200        |       1 000       |
| Support prioritaire                                      |      —       |         ✓         |         ✓         |

Les alertes intelligentes, le regroupement des répétitions, la suppression et les filtres de gravité, l'indexation pour la recherche, le suivi des métriques, tous les canaux de notification, les actions sur les conteneurs et l'accès MCP illimité sont sur **toutes** les offres, y compris la gratuite.

Les tarifs à jour sont sur [cloud.dozzle.dev](https://cloud.dozzle.dev).

## Ce qui compte comme évènement traité

Un **évènement traité** est un évènement de conteneur ou une ligne de log correspondante passée par le pipeline de triage, qui décide s'il faut envoyer une nouvelle alerte, la fusionner dans une existante, ou rester silencieux. Vous payez ce travail, pas du stockage brut.

Les lignes de log ordinaires ne sont pas des évènements : elles comptent dans le volume de logs interrogeables. Un conteneur très bavard coûte donc du stockage, tandis qu'un conteneur en boucle de crash coûte des évènements.

Si un conteneur se termine 47 fois, cela fait 47 évènements sur le quota, mais une seule alerte qui dit 47. C'est tout l'intérêt.

**Les constats ne coûtent ni l'un ni l'autre.** La revue lit vos logs plutôt que votre historique d'alertes, elle ne touche donc pas au compteur d'évènements et fonctionne sans aucune règle d'alerte configurée. Seul votre volume mensuel de logs s'applique.

**Recherche et métriques sont gratuites sur toutes les offres.** L'indexation est active dès qu'une instance se connecte, et les séries CPU, mémoire, réseau et disque remontent de vos instances sans frais. L'offre ne change que la quantité et la profondeur.

## Dépasser le quota

Rien ne casse. Vous basculez en mode échantillonnage :

- Le triage se met en pause.
- L'historique des évènements continue d'être enregistré, rien n'est perdu.
- Environ un évènement sur dix passe en alerte **brute**, pour que vous voyiez toujours ce qui se passe.
- Les répétitions ne sont plus fusionnées en une alerte avec compteur.

En pratique, les alertes deviennent plus bruyantes et moins utiles au lieu de disparaître, et vous le sentirez dans votre boîte mail avant de le voir sur une page d'utilisation. Si vos alertes sont soudain brutes et répétitives, vérifiez d'abord votre consommation.

Cela vaut aussi pour les offres payantes. Les quotas se réinitialisent au début de chaque mois.

> [!TIP]
> La plupart des comptes n'en approchent jamais. Seul un vrai déluge — un conteneur en boucle de crash pendant des jours — dépasse le quota gratuit, et l'alerte qui nomme ce conteneur arrive bien avant la limite.

## Rétention

La rétention détermine jusqu'où remonte votre historique : alertes, évènements et résultats de recherche dans les logs. Sur l'offre gratuite c'est un jour, donc une recherche portant sur la semaine dernière ne renvoie rien même si l'évènement a bien eu lieu. C'est la raison la plus fréquente pour laquelle une recherche semble « perdre » des données.

La rétention des métriques est de 24 heures en gratuit et de 30 jours sur les offres payantes. Demander une fenêtre plus longue que ce que permet votre offre renvoie la fenêtre dont vous disposez réellement, pas une erreur.

## La limite d'instances

L'offre gratuite relie une instance à la fois. En relier une deuxième affiche un message de limite. Vous avez deux options :

- **Déplacer la place.** Supprimez la clé d'API de l'instance existante sur la page Instances, puis reliez la nouvelle. C'est définitif pour l'ancienne instance : son historique reste, mais il faudrait la relier de zéro.
- **Passer à une offre supérieure** pour garder les deux connectées en même temps.

La limite porte sur ce qu'un compte gratuit transmet, ce n'est pas un verrou sur une fonctionnalité. Tout le reste fonctionne en gratuit avec l'instance que vous avez reliée.

## Vérifier votre consommation

La page d'utilisation dans Cloud affiche les évènements, les octets de logs et les conversations avec l'assistant consommés ce mois-ci par rapport à votre quota. Vous pouvez aussi demander dans le chat : « combien ai-je consommé ce mois-ci ? ».

## Changer ou annuler

Passez à une offre supérieure depuis la page des tarifs ou depuis les réglages. La facturation passe par Stripe ; moyens de paiement, factures et reçus s'y gèrent via le lien de facturation dans vos réglages.

L'annulation arrête les prélèvements futurs et vous bascule sur l'offre gratuite à la fin de la période déjà payée. Votre compte et votre historique restent.
