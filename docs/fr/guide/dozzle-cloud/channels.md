---
title: Canaux de notification
sourceHash: baae80ce439a
---

# Canaux de notification

Les canaux se configurent dans [Dozzle Cloud](/fr/guide/dozzle-cloud) et déterminent _où_ vont les alertes. Ce qui les _déclenche_ se configure sur votre instance auto-hébergée : voir [Alertes](/fr/guide/alerts-and-webhooks).

Activez-en autant que vous voulez. Chaque canal activé reçoit toutes les alertes, et chacun s'active ou se désactive indépendamment.

## Canaux disponibles

| Canal                                                         | Alertes | Résumé quotidien | Agent bidirectionnel |
| ------------------------------------------------------------- | :-----: | :--------------: | :------------------: |
| <Icon icon="mdi:email-outline" inline /> E-mail               |    ✓    |        ✓         |                      |
| <Icon icon="mdi:telegram" inline /> Telegram                  |    ✓    |        ✓         |          ✓           |
| <Icon icon="ic:baseline-discord" inline /> Bot Discord (MP)   |    ✓    |        ✓         |          ✓           |
| <Icon icon="ic:baseline-discord" inline /> Webhook Discord    |    ✓    |        ✓         |                      |
| <Icon icon="mdi:slack" inline /> Slack                        |    ✓    |                  |                      |
| <Icon icon="simple-icons:ntfy" inline /> ntfy                 |    ✓    |                  |                      |
| <Icon icon="mdi:webhook" inline /> Webhooks                   |    ✓    |                  |                      |
| <Icon icon="mdi:bell-badge-outline" inline /> Push navigateur |    ✓    |                  |                      |

Tous les canaux sont disponibles sur toutes les offres, y compris la gratuite.

## E-mail

Configuré automatiquement avec l'adresse utilisée à l'inscription. Rien à régler. Pour l'arrêter, désactivez le canal e-mail. Si les alertes cessent d'arriver sans raison, regardez d'abord dans les spams : la première alerte y atterrit parfois, et la marquer « non indésirable » règle le problème définitivement.

## Telegram

Choisissez **Telegram** sur la page Channels, suivez le lien pour ouvrir le bot et appuyez sur **Start**. Le canal s'active dès que le bot a eu de vos nouvelles.

Telegram est bidirectionnel. Vous pouvez répondre dans la même conversation et poser des questions sur vos conteneurs (« des erreurs aujourd'hui ? », « montre l'utilisation CPU », « quelles alertes ai-je ? ») et obtenir des réponses sur l'état en direct.

## Discord

Discord propose **deux types de canaux distincts**, et les avoir tous les deux actifs est la raison habituelle de recevoir chaque alerte en double.

**Bot Discord (message privé)** — le bot vous envoie les alertes personnellement en MP. Bidirectionnel, vous pouvez donc lui poser des questions. Se configure en autorisant le bot depuis la page Channels.

**Webhook Discord (salon de serveur)** — les alertes sont publiées dans un salon de votre serveur, par exemple `#alerts`. Sens unique. Se configure en créant un webhook dans les paramètres de votre serveur Discord et en collant l'URL dans Cloud.

Si les alertes arrivent à la fois dans vos MP et dans un salon de serveur, vous avez les deux configurés. Désactivez celui dont vous ne voulez pas ; en couper un laisse l'autre tourner. Une configuration courante est de garder le salon partagé et de couper le MP.

## Slack

Créez un webhook entrant dans votre espace Slack et collez l'URL dans le canal Slack de la page Channels.

## ntfy

Saisissez l'URL de votre topic. ntfy.sh comme un serveur ntfy auto-hébergé fonctionnent. Très utilisé pour les notifications sur téléphone sans compte supplémentaire.

## Webhooks

Saisissez n'importe quelle URL qui accepte un POST. Les alertes sont livrées en JSON, vous pouvez donc les router vers ce que vous faites déjà tourner : Home Assistant, n8n, un script, un autre outil d'alerting.

> [!NOTE]
> C'est un canal Cloud, distinct des webhooks que votre Dozzle auto-hébergé peut appeler directement. Ceux-là sont dans [Alertes](/fr/guide/alerts-and-webhooks), avec les variables de template Go.

## Push navigateur

Activez-le sur la page Channels et autorisez les notifications quand le navigateur le demande. Les alertes arrivent alors en notifications de bureau.

Si rien n'arrive après l'activation, le navigateur a très probablement refusé la permission. Les navigateurs ne redemandent pas après un refus : effacez la permission de notification du site dans les réglages du navigateur et réactivez-la. Le push navigateur ne fonctionne pas en fenêtre privée ou de navigation privée.

## <Icon icon="mdi:bell-sleep-outline" inline /> Faire moins de bruit

Vous ne devriez être dérangé que quand ça compte. Si Cloud est bruyant, c'est un problème de réglage, et voici les outils pour ça.

| Situation                                        | Faites ceci                            |
| ------------------------------------------------ | -------------------------------------- |
| Une erreur récurrente que vous connaissez déjà   | **Mettez le motif en sourdine**        |
| Les alertes sont utiles mais trop fréquentes     | **Votez pouce vers le bas**            |
| Maintenance planifiée, sauvegardes, mises à jour | **Coupez le motif avant de commencer** |
| Bonne alerte, mauvaise application               | **Désactivez ce canal**                |
| Vous n'en voulez plus du tout, de nulle part     | **Désactivez tous les canaux**         |

Supprimer la règle d'alerte n'est presque jamais la bonne réponse : cela retire toute une catégorie de surveillance pour régler une seule ligne bruyante.

### Mettre une alerte récurrente en sourdine

La mise en sourdine est par motif : elle fait taire _ce type d'alerte_, pas seulement celle que vous avez sous les yeux. Les occurrences suivantes restent silencieuses, et tout ce qui est réellement différent passe toujours.

- **Depuis une alerte** — ouvrez-la dans Cloud et choisissez de la mettre en sourdine.
- **Dans le chat** — dites « coupe ça » ou « arrête de me parler de X ». L'agent énonce le motif exact qu'il s'apprête à couper et attend votre confirmation, car une mise en sourdine est durable et pourrait masquer une vraie panne plus tard.

La sourdine dure jusqu'à ce que vous la leviez. Demandez « qu'est-ce que j'ai coupé ? » pour lister vos règles, et levez-les de la même façon. Les alertes en sourdine restent enregistrées : la sourdine change ce qui vous interrompt, pas ce qui est surveillé.

### Moins, pas rien

Si une alerte est réellement utile mais arrive trop souvent, votez **pouce vers le bas** plutôt que de la couper. C'est le signal « continue de surveiller ça, dérange-moi moins ». Le pouce en l'air sur les alertes qui ont vu juste aide de la même manière.

### Les répétitions sont déjà regroupées

Avant de couper, vérifiez si le problème est bien de la répétition. Les occurrences répétées d'une même panne sont fusionnées en une seule alerte avec un compteur. Si vous recevez beaucoup d'alertes, ce sont généralement beaucoup de problèmes _différents_, ou vous avez dépassé le quota d'évènements de votre offre et les alertes sont retombées en brut, non regroupées. Voir [Offres et limites](/fr/guide/dozzle-cloud/plans).

### Filtrer à la source

Pour un conteneur bruyant en fonctionnement normal, le meilleur correctif est en amont : le label `dev.dozzle.cloud.min_level` empêche les lignes de faible gravité de quitter votre hôte. Voir [Vos données](/fr/guide/dozzle-cloud/your-data).

## Pourquoi n'ai-je pas reçu d'alerte ?

**1. Existe-t-il une règle pour ça ?** Une erreur dans vos logs ne produit pas d'alerte à elle seule ; quelque chose doit la guetter. La règle par défaut ne couvre que les conteneurs qui se terminent en erreur : un conteneur qui journalise des erreurs tout en restant debout demande une règle de log.

**2. Un canal est-il activé ?** Une règle sans canal activé n'a nulle part où livrer.

**3. L'instance est-elle connectée ?** Si elle était hors ligne au moment du problème, rien n'a été transmis. Voir [Relier votre instance](/fr/guide/dozzle-cloud/connecting).

**4. A-t-elle été regroupée dans une alerte déjà reçue ?** Quarante défaillances produisent une alerte qui dit quarante. C'est voulu, pas un raté.

**5. L'avez-vous mise en sourdine ?** Vérifiez vos règles de sourdine.

**6. Le conteneur est-il exclu de la transmission ?** Voir [Vos données](/fr/guide/dozzle-cloud/your-data).

**7. Êtes-vous au-delà des limites de votre offre ?** Passé le quota, la livraison change et les alertes sont échantillonnées.

**8. Regardez vos spams**, pour l'e-mail en particulier.
