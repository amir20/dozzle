---
title: Dans votre Dozzle
sourceHash: 8053f9b40bcd
---

# Dans votre Dozzle

Ce qui change dans votre propre Dozzle une fois qu'une instance est [reliée](/fr/guide/dozzle-cloud/connecting). Tout ce qui suit vit dans l'interface Dozzle que vous utilisez déjà, à côté des logs concernés, et non sur un site séparé qu'il faudrait aller consulter.

## Rien de local n'est verrouillé

Dozzle ne perd aucune fonction quand Cloud n'est pas configuré. Les alertes se déclenchent toujours, s'insèrent toujours dans le flux de logs et atteignent toujours vos webhooks. Ce que la liaison ajoute, c'est la **mémoire** : la même alerte est encore là après un rechargement, après un redémarrage, et une semaine plus tard.

C'est toute la frontière entre les deux. Dozzle possède le conteneur qui est devant vous et sait agir dessus. Cloud possède ce qui relève de l'historique, ce qui traverse plusieurs instances, et ce qui appartient au compte.

Une installation sans Cloud affiche donc une section d'historique vide avec une ligne qui dit ce qu'elle contiendrait, et non une carte verrouillée :

> Les alertes apparaîtront ici une fois cette instance reliée à Dozzle Cloud. En attendant, elles apparaissent dans le flux de logs et sont oubliées au rechargement.

## Le rail cloud

Une bande d'icônes sur le bord droit de la vue des logs, et le panneau que l'une d'elles ouvre. Il n'est monté que si Cloud est relié, et seulement sur une vue où il y a des logs à l'écran : il n'apparaît donc jamais sur la page d'accueil ni dans les réglages.

Le panneau se place **à côté** du flux plutôt que par-dessus : la page réserve exactement sa largeur, si bien que rien sur le rail ne recouvre jamais les lignes dont il parle. Sur un téléphone, il n'y a pas la place pour une bande permanente : le panneau devient une feuille plein écran, ouverte depuis la barre d'outils ou la palette de commandes.

Le fait de masquer le rail est mémorisé. Un petit onglet sur le bord le ramène, comme la barre latérale se replie de l'autre côté.

### <Icon icon="mdi:message-outline" inline /> Demander à Dozzle

Une question sur la vue que vous regardez, à laquelle on répond à partir des logs qu'elle contient. Au-dessus du champ de saisie figure ce qui part avec la question : quels conteneurs sont dans la vue, combien de lignes, et la ligne de log que vous avez désignée si vous êtes parti du menu d'une ligne. Rien n'est envoyé en silence.

Les réponses se terminent dans Dozzle plutôt que par un lien vers l'extérieur. Quand la réponse porte sur un moment précis, l'action conduit votre propre flux jusqu'à lui.

Ouvrez-le avec <kbd>Maj</kbd> + <kbd>⌘</kbd> + <kbd>K</kbd>, depuis le menu du conteneur, ou depuis une ligne de log.

### <Icon icon="mdi:chart-line" inline /> Métriques

Dozzle garde 300 échantillons dans le navigateur et rien derrière : « est-ce que c'était comme ça il y a une heure ? » n'a donc pas de réponse locale. Ce panneau relit les échantillons que votre instance envoie depuis le début : CPU et mémoire sur les dernières 1 h, 6 h ou 24 h, avec le pic mis en avant, et le survol qui vous lit le graphique. Pro ajoute une fenêtre de 7 jours.

### <Icon icon="mdi:bell-outline" inline /> Alertes

Ce qui s'est déclenché sur les conteneurs actuellement dans la vue. La page [notifications](/fr/guide/alerts-and-webhooks) répond à la même question pour toute l'instance ; ce panneau se limite à ce qui est à l'écran, et c'est la seule raison qui lui vaut une place à côté du flux. L'ouvrir éteint la pastille non lue sur la cloche.

## Des alertes qui survivent au rechargement

Une fois les alertes mémorisées, elles apparaissent à trois endroits de plus :

- **Sur la règle qui les a déclenchées**, de sorte qu'une règle écrite il y a des mois se juge à ce qu'elle a réellement attrapé.
- **En pastille sur la ligne du conteneur** dans le tableau, colorée selon la gravité. Un clic ouvre un petit panneau avec le titre, le niveau, l'heure de déclenchement, le nombre d'événements regroupés et un résumé d'une ligne.
- **Dans la liste d'activité** de la page notifications, filtrable et lisible longtemps après la disparition de la notification éphémère.

## « Montre-moi les lignes »

Chacune de ces surfaces se termine par la même action, et elle y est toujours l'action principale.

**Montre-moi les lignes** ouvre la vue historique du conteneur, positionnée sur le moment décrit par l'alerte ou le constat, la ligne exacte surlignée et le terme de recherche déjà rempli. Les alertes de métrique et d'événement ne portent aucune ligne de log : elles arrivent donc sur le moment plutôt que sur une ligne.

C'est le seul geste que Dozzle peut faire et que Cloud ne peut pas, et c'est bien pour cela que ces panneaux vivent dans Dozzle. Les liens sortants restent réservés à ce que Cloud fait vraiment mieux : facturation et clés d'API, archive des rapports, synthèses entre instances, et transcriptions d'enquêtes complètes.

## Constats

Les constats ne sont pas des alertes. Une alerte est une règle que vous avez écrite ; un constat est quelque chose que Cloud a remarqué sans qu'aucune règle ne le couvre. Les deux ne sont jamais listés côte à côte, et la surface des constats n'existe que si Cloud est configuré, plutôt que de rester dans la navigation comme une publicité permanente.

Voir [Dozzle Cloud](/fr/guide/dozzle-cloud) pour ce que couvrent les constats selon le forfait.

## Tout désactiver

Délier l'instance retire toutes les surfaces de cette page et laisse Dozzle exactement tel qu'il était : les alertes se déclenchent toujours, apparaissent toujours dans le flux, et sont oubliées au rechargement. Voir [Vos données](/fr/guide/dozzle-cloud/your-data) pour ce qui cesse de quitter votre hôte.
