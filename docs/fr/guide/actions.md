---
title: Actions sur les conteneurs
sourceHash: 7eab1f511f5f
---

# Actions sur les conteneurs

<Badge type="warning" text="Docker Only" />

Dozzle propose des actions sur les conteneurs, qui vous permettent de les démarrer (`start`), arrêter (`stop`), redémarrer (`restart`), supprimer (`remove`) et mettre à jour (`update`) depuis le menu déroulant à droite, à côté des statistiques du conteneur. Cette fonctionnalité est **désactivée** par défaut et s'active en mettant la variable d'environnement `DOZZLE_ENABLE_ACTIONS` à `true`.

L'action `update` récupère la dernière image du conteneur et le recrée avec la même configuration, ce qui est pratique pour mettre à niveau un conteneur sur place sans modifier son fichier compose. `update` n'a un effet réel que si l'image utilise un tag mouvant (par ex. `latest`, `stable`) ; avec un tag figé, la même image sera simplement retéléchargée.

> [!WARNING]
> `remove` et `update` recréent le conteneur. Les données écrites dans des **volumes anonymes** ou dans la couche inscriptible du conteneur seront perdues. Les volumes nommés et les bind mounts sont préservés.

::: code-group

```sh
docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --enable-actions
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    ports:
      - 8080:8080
    environment:
      DOZZLE_ENABLE_ACTIONS: true
```

:::

## Vérification des mises à jour

Dozzle vérifie si l'image que fait tourner un conteneur est toujours celle que sert son registre. En cas de différence, un point apparaît sur le menu du conteneur et le menu indique qu'une mise à jour est disponible.

La vérification demande au registre l'empreinte du tag à partir duquel le conteneur a été créé, et la compare à celle de l'image réellement en cours d'exécution. Elle le fait avec une requête `HEAD` sur le manifeste de l'image, donc aucune couche n'est téléchargée et cela ne compte pas dans les limites de téléchargement de Docker Hub. Les réponses sont mises en cache six heures, et une même image n'est interrogée qu'une seule fois, quel que soit le nombre de conteneurs ou d'hôtes qui l'utilisent.

Comme la comparaison porte sur ce que le conteneur _exécute_, un conteneur reste obsolète jusqu'à sa recréation, même si une image plus récente a déjà été téléchargée sur l'hôte.

La vérification est indépendante des actions. Savoir qu'un conteneur est obsolète est utile, que Dozzle ait le droit d'y faire quelque chose ou non, donc l'avertissement apparaît même si `DOZZLE_ENABLE_ACTIONS` est désactivé. Seul le bouton `Update` nécessite les actions.

### Désactiver la vérification

`DOZZLE_IMAGE_CHECK_MODE` contrôle si Dozzle contacte les registres.

| Valeur      | Comportement                                                                          |
| ----------- | ------------------------------------------------------------------------------------- |
| `automatic` | Vérifie en arrière-plan lorsqu'un conteneur est consulté.                             |
| `manual`    | Ne vérifie jamais de lui-même. Le menu propose une action « Check for updates ».      |
| `off`       | La fonctionnalité disparaît. Aucun endpoint n'est enregistré et aucune requête faite. |

Sa valeur par défaut est celle de `DOZZLE_RELEASE_CHECK_MODE` : si vous avez déjà indiqué à Dozzle de ne pas récupérer les versions automatiquement, il ne vérifiera pas non plus les images automatiquement.

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      DOZZLE_IMAGE_CHECK_MODE: off
```

Pour faire taire un seul conteneur, par exemple un conteneur volontairement figé sur une version, ajoutez-lui ce label :

```yaml [docker-compose.yml]
services:
  database:
    image: postgres:18-alpine
    labels:
      dev.dozzle.update-check: false
```

Une notification peut aussi être affichée quand une mise à jour est trouvée. Elle est désactivée par défaut et se trouve dans les paramètres.

### Ce qui ne peut pas être vérifié

Certains conteneurs n'ont rien à comparer, et Dozzle reste silencieux plutôt que de deviner :

- Les images construites localement, qui n'ont pas d'empreinte de registre
- Les références figées sur une empreinte, qui ne peuvent pas changer
- Les registres privés, puisque Dozzle n'a pas d'identifiants propres
- Kubernetes, où le déploiement des images relève du cluster

### Mettre à jour Dozzle lui-même

Dozzle ne peut pas s'arrêter lui-même pour se mettre à jour sur place, donc un conteneur Dozzle autonome affiche l'avertissement de mise à jour avec un lien vers les notes de version au lieu d'un bouton `Update`. Faire tourner Dozzle en service Swarm fonctionne normalement, puisque la mise à jour est confiée à l'orchestrateur. Les agents Dozzle sur les autres hôtes sont des conteneurs ordinaires et se mettent à jour comme n'importe lequel.
