---
title: Profil par défaut
sourceHash: 1ef0edd24fb4
---

# Profil par défaut

Dozzle enregistre sur disque les préférences d'interface de chaque utilisateur (thème, langue, conteneurs épinglés, groupes repliés, clés JSON visibles, etc.) dans `/data/<username>/profile.json`. Lorsque l'[authentification](/fr/guide/authentication) est désactivée, ou pour tout utilisateur qui ne s'est pas encore connecté et n'a pas personnalisé ses paramètres, Dozzle bascule sur un profil spécial nommé `__default__`.

Vous pouvez livrer un profil préconfiguré en créant le fichier `/data/__default__/profile.json`. Les visiteurs anonymes et tout nouvel utilisateur sans profil enregistré chargeront ces paramètres à leur première visite.

## Emplacement du fichier

```
/data/__default__/profile.json
```

Si le fichier n'existe pas, Dozzle démarre avec ses valeurs par défaut internes. Vous n'avez besoin de le créer que si vous souhaitez les remplacer.

## Exemple

```json
{
  "settings": {
    "showTimestamp": true,
    "showStd": false,
    "showAllContainers": false,
    "softWrap": true,
    "collapseNav": false,
    "smallerScrollbars": false,
    "search": false,
    "compact": false,
    "menuWidth": 15,
    "size": "medium",
    "lightTheme": "auto",
    "hourStyle": "auto",
    "dateLocale": "auto",
    "locale": "en",
    "groupContainers": "at-least-2",
    "automaticRedirect": "delayed"
  },
  "pinned": [],
  "visibleKeys": [],
  "collapsedGroups": []
}
```

Tous les champs sont facultatifs : n'incluez que ceux que vous voulez remplacer.

## Paramètres disponibles

| Champ               | Type    | Description                                                                |
| ------------------- | ------- | -------------------------------------------------------------------------- |
| `showTimestamp`     | booléen | Afficher l'horodatage à côté de chaque ligne de log                        |
| `showStd`           | booléen | Afficher l'indicateur de flux stdout/stderr                                |
| `showAllContainers` | booléen | Inclure les conteneurs arrêtés dans la barre latérale                      |
| `softWrap`          | booléen | Retour à la ligne des lignes longues au lieu du défilement horizontal      |
| `collapseNav`       | booléen | Démarrer avec la barre latérale repliée                                    |
| `smallerScrollbars` | booléen | Utiliser des barres de défilement plus fines                               |
| `search`            | booléen | Activer la recherche intégrée par défaut                                   |
| `compact`           | booléen | Espacement compact des lignes de log                                       |
| `menuWidth`         | nombre  | Largeur de la barre latérale en pourcentage de la fenêtre. Limitée à `50`. |
| `size`              | chaîne  | Taille de police : `small`, `medium`, `large`                              |
| `lightTheme`        | chaîne  | Préférence de thème : `auto`, `light`, `dark`                              |
| `hourStyle`         | chaîne  | Format de l'heure : `auto`, `12`, `24`                                     |
| `dateLocale`        | chaîne  | Format de date/heure : `auto`, `en-US`, `en-GB`, `de-DE`, `en-CA`          |
| `locale`            | chaîne  | Langue de l'interface (par ex. `en`, `fr`, `de`)                           |
| `groupContainers`   | chaîne  | Regroupement dans la barre latérale : `always`, `at-least-2`, `never`      |
| `automaticRedirect` | chaîne  | Redirection vers un nouveau conteneur : `instant`, `delayed`, `none`       |

Les valeurs hors de ces ensembles ne sont pas acceptées, donc `groupContainers: "stack"` ou un `dateLocale` valant `fr-FR` ne feront pas ce que vous attendez.

Les champs de premier niveau `pinned`, `visibleKeys` et `collapsedGroups` acceptent des tableaux et permettent d'épingler des conteneurs ou de replier des groupes à l'avance pour les nouveaux visiteurs. Dozzle écrit aussi `releaseSeen`, `dismissedImageUpdates` et `dismissedLinkHint` au premier niveau pour mémoriser ce qu'un utilisateur a déjà ignoré. Définir `dismissedLinkHint: true` supprime l'astuce de lien du premier lancement pour tout le monde.

## Fonctionnement

- Au chargement de la page, Dozzle lit `/data/<username>/profile.json` pour l'utilisateur connecté, ou `/data/__default__/profile.json` quand aucun utilisateur n'est authentifié.
- Quand un utilisateur modifie un paramètre dans l'interface, la nouvelle valeur est enregistrée sous son propre nom d'utilisateur (ou de nouveau dans `__default__` si l'authentification est désactivée).
- Le profil `__default__` est donc à la fois le **modèle pour les nouveaux visiteurs** et le **profil actif de l'utilisateur anonyme** dans les déploiements sans authentification.

::: tip
Si vous voulez seulement définir des valeurs par défaut tout en laissant l'utilisateur anonyme les personnaliser à l'exécution, montez le fichier en lecture seule : Dozzle ne pourra pas enregistrer les changements mais l'interface continue de fonctionner.
:::
