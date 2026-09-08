---
title: Dozzle Cloud
sourceHash: 49197a749322
---

# Dozzle Cloud

[Dozzle Cloud](https://cloud.dozzle.dev) est un compagnon managé optionnel de Dozzle auto-hébergé. Dozzle reste entièrement open source et auto-hébergé ; Cloud se pose par-dessus et prend en charge la partie vraiment difficile à faire tourner soi-même : décider ce qui mérite de vous réveiller, et déterminer ce qui a réellement cassé.

Votre Dozzle ouvre une connexion sortante vers Cloud. Aucun port entrant, aucune IP publique, aucun agent à installer.

**Le gratuit vous laisse tranquille. Pro part à la recherche.**

## <Icon icon="mdi:bell-ring-outline" inline /> Gratuit : une couche de notification intelligente

La plupart des alertes de logs se résument à une regex et un webhook, ce qui veut dire que la première boucle de crash produit deux cents messages identiques et que vous coupez le canal. L'offre gratuite existe pour corriger cette partie-là, et c'est le produit d'alerting complet, pas son essai.

- **Alertes intelligentes** — chaque alerte déclenchée par vos règles Dozzle devient une phrase qui nomme la cause, le conteneur et la gravité, avec un lien vers la ligne de log exacte dans votre propre Dozzle.
- **Les répétitions sont regroupées** — 47 plantages arrivent en une alerte qui dit 47. Vous recevez un avis de rétablissement quand ça revient.
- **Silencieux par défaut** — suppression, filtres de gravité et mise en sourdine par motif sur chaque canal. Coupez _ce type d'alerte_ plutôt que celle-ci en particulier, et tout ce qui est réellement différent passe toujours.
- **Tous les canaux** — e-mail, Telegram, Discord, Slack, ntfy, webhooks et notifications navigateur, tous en gratuit. Voir [Canaux de notification](/fr/guide/dozzle-cloud/channels).
- **Recherche et métriques incluses** — chaque évènement est interrogeable dès son arrivée, et CPU, mémoire, réseau et disque sont conservés en historique. Ni l'un ni l'autre ne compte dans votre quota d'évènements.
- **Un constat par semaine** — même en gratuit, Cloud lit vos logs et fait remonter la chose la plus grave sur laquelle rien n'a alerté.
- **Une règle par défaut qui marche** — relier une instance en crée une pour vous (les conteneurs qui se terminent en erreur), donc un compte tout neuf reçoit une alerte utile dès le premier jour sans rien configurer.
- **Agent conversationnel et MCP** — demandez « des erreurs aujourd'hui ? » dans Telegram ou Discord, et démarrez, arrêtez ou redémarrez un conteneur depuis la même conversation dès que vous activez les [Actions](/fr/guide/actions) sur votre instance. L'accès MCP est illimité sur toutes les offres.

> [!TIP]
> Une instance fraîchement reliée bénéficie de 7 jours de l'expérience Pro complète : tous les constats, chaque matin. Le gratuit se stabilise ensuite à un constat par semaine.

## <Icon icon="mdi:robot-outline" inline /> Pro : il cherche avant que quoi que ce soit n'alerte

Le gratuit vous dit *qu'*il s'est passé quelque chose, et se tait quand rien ne s'est passé. Pro est la moitié qui n'attend pas qu'une alerte existe.

- **Revue proactive, chaque matin** — Cloud lit vos logs d'erreur, les rassemble en motifs et signale ce qui mérite d'être corrigé. C'est là qu'un disque qui se remplit doucement, ou un conteneur qui redémarre en boucle sans bruit, apparaît un jour où rien n'a été déclenché. Aucune règle d'alerte n'a besoin d'exister pour ça.
- **Tous les constats, chaque jour, avec le correctif** — pas un par semaine et le reste verrouillé. Les constats vieillissent au fil des jours tant que le problème dure (« toujours en cours, jour quatre, trois fois pire ») et se referment d'eux-mêmes quand ça s'arrête.
- **Un triage qui va voir** — quand le texte de l'alerte ne suffit pas à trancher, il inspecte le conteneur et lit les logs alentour avant de se prononcer, au lieu de deviner.
- **Enquêtes complètes à la demande** — un clic lance davantage de passes avec un modèle plus puissant, en corrélant vos conteneurs, vos hôtes et la chronologie, et vous rend une cause racine avec des étapes concrètes.
- **Tous les hôtes, un tableau de bord** — reliez autant d'instances Dozzle que vous en exploitez. Les questions posées dans le chat les couvrent toutes d'un coup.
- **Une mémoire plus longue** — 30 jours de logs et de métriques interrogeables au lieu de 24 heures, soit la différence entre « qu'est-ce qui s'est passé cette nuit » et « est-ce que ça dure depuis un mois ».

La comparaison complète est dans [Offres et limites](/fr/guide/dozzle-cloud/plans).

## Pour aller plus loin

| Page                                                       | Ce qu'elle couvre                                                                                  |
| ---------------------------------------------------------- | -------------------------------------------------------------------------------------------------- |
| [Relier votre instance](/fr/guide/dozzle-cloud/connecting) | Le lien, pourquoi ni IP publique ni port ouvert ne sont nécessaires, pare-feu, dépannage           |
| [Canaux de notification](/fr/guide/dozzle-cloud/channels)  | Tous les canaux, comment configurer chacun, et comment faire moins de bruit                        |
| [Offres et limites](/fr/guide/dozzle-cloud/plans)          | Ce que contient chaque offre, ce qu'est un évènement traité, ce qui se passe en cas de dépassement |
| [Vos données](/fr/guide/dozzle-cloud/your-data)            | Ce qui quitte votre hôte, comment l'arrêter, ce que Cloud stocke, les clés d'API                   |

Les règles d'alerte se configurent sur votre propre instance, pas dans Cloud. Voir [Alertes](/fr/guide/alerts-and-webhooks).

## Retours

Dozzle Cloud est construit par la même personne que Dozzle, et l'exigence est la même : des choses que les gens ont vraiment envie d'utiliser. Si vous l'essayez et que quelque chose vous semble bancal, manquant ou franchement utile, [ouvrez une discussion](https://github.com/amir20/dozzle/discussions). Ces retours orientent ce qui sera construit ensuite.
