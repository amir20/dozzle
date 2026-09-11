---
title: Nouveautés de la v11
sourceHash: 4b0ece212354
---

# <Icon icon="mdi:party-popper" inline /> Nouveautés de la v11

La v11 est le plus gros changement visuel qu'ait connu Dozzle. Presque toutes les surfaces ont été redessinées selon un seul langage visuel : plat, calme, des panneaux neutres, la couleur réservée à ce qui mérite vraiment l'attention. S'y ajoutent la connexion avec GitHub et OIDC, et un flux de logs qui comprend quelques formats de plus.

## Une nouvelle apparence

- **Barre latérale** reconstruite autour de groupes repliables avec un compteur, d'icônes d'application portant l'état du conteneur en pastille de coin, et d'une ligne active teintée. Fusionner tout un groupe est désormais un bouton sur le groupe lui-même.
- **Flux de logs** redessiné. Les horodatages sont discrets et sans cadre, une ligne simple reçoit un point de niveau tandis qu'une entrée groupée reçoit une barre, et les lignes `warn` et `error` sont légèrement teintées pour se repérer au défilement.
- **Barre de titre du conteneur** simplifiée. Le nom passe en premier, l'image est un texte atténué qu'un clic copie, et l'épinglage se fait sur une punaise assortie à la section « Épinglés » de la barre latérale.
- **Tableau de bord d'accueil**, **palette de commandes**, **notifications éphémères**, **panneaux attach et shell** et **menu du conteneur** ont tous été redessinés. Le menu est réparti en sections nommées plutôt qu'en une longue liste.
- L'**indicateur de flux en direct** et l'**indicateur de position de défilement** sont nouveaux. Ce dernier flotte au-dessus du flux, indique où l'on se trouve dans la vie du conteneur, et disparaît dès que l'on cesse de défiler.
- Tous les menus utilisent maintenant l'API popover native : ils ne sont plus rognés ni prisonniers d'un panneau qui défile.

## Connexion avec GitHub et OIDC

Les utilisateurs peuvent se connecter avec un compte GitHub ou n'importe quel fournisseur OIDC (Authentik, Keycloak, Pocket ID, Google). Cela fait partie du fournisseur `simple` : `users.yml` reste la liste d'autorisation et décide toujours qui entre. Aucun compte n'est créé automatiquement, et la connexion par mot de passe continue de fonctionner à côté.

```yaml
environment:
  DOZZLE_AUTH_PROVIDER: simple
  DOZZLE_AUTH_GITHUB_CLIENT_ID: Ov23liABCDEFGHIJKLMN
  DOZZLE_AUTH_GITHUB_CLIENT_SECRET: 0123456789abcdef0123456789abcdef01234567
```

La configuration complète est décrite dans [Connexion avec GitHub & OIDC](/fr/guide/authentication/oauth).

## Logs

- Les champs OpenTelemetry `severityText` et `severityNumber` sont reconnus comme niveaux de log, `severityNumber` servant quand le texte n'est pas un nom de niveau.
- Les niveaux numériques de Pino (`30`, `40`, `50`) sont interprétés.
- Les bascules de champs s'appliquent aussi aux vues service et stack, et non plus aux seuls conteneurs isolés.
- Les colonnes épinglées vivent dans l'URL : une vue côte à côte devient un lien que l'on peut envoyer.

## Alertes et Dozzle Cloud

- Les alertes sont mémorisées. Elles apparaissent sur la règle qui les a déclenchées et en pastille sur la ligne du conteneur, et elles survivent à un rechargement.
- Un nouveau rail cloud se place à côté du flux avec trois panneaux : poser une question sur ce qui est à l'écran, les métriques derrière le graphique en direct, et les alertes déclenchées sur les conteneurs de la vue. Il n'est monté que si Cloud est relié, et se replie en un onglet sur le bord.
- Les outils cloud sont limités à la personne qui pose la question : l'assistant voit exactement ce que ce compte a le droit de voir.
- `min_level=disabled` n'arrête plus les métriques en même temps que les logs.

## Performances et corrections

- Les premières lignes de logs s'affichent nettement plus tôt.
- Les flux de logs que le navigateur avait silencieusement abandonnés se reconnectent.
- Les points de statistiques dans la charge utile des changements de conteneurs sont bien plus petits.
- Les flux de logs fusionnés ne bloquent plus le store des conteneurs.

## Mise à jour

Les jetons de session sont désormais signés avec un secret aléatoire conservé dans `session_secret`, au niveau du répertoire de données, à côté de `users.yml`. **Tout le monde est déconnecté une fois après la mise à jour.** Si `/data` n'est pas accessible en écriture, Dozzle démarre quand même avec un secret en mémoire et affiche un avertissement, ce qui veut dire que les sessions disparaissent à chaque redémarrage.

L'ancienne clé était dérivée de `users.yml` seul, ce qui ne suffisait plus dès lors qu'un compte peut être prouvé par OAuth et ne porter aucune empreinte de mot de passe.
