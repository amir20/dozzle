---
title: Actions sur les conteneurs
sourceHash: f51478b92485
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

L'action `Update` sur le propre conteneur de Dozzle met Dozzle à jour sur place. Elle récupère la nouvelle image et confie le remplacement à un conteneur auxiliaire éphémère, donc Dozzle disparaît quelques secondes et revient dans la nouvelle version avec la même configuration et les mêmes volumes. Elle peut aussi être planifiée. Consultez [Comment Dozzle se met à jour lui-même](/fr/guide/setup-wizard#self-update) pour savoir ce qui est conservé et quelles installations ne sont pas prises en charge. Faire tourner Dozzle en service Swarm le met à jour via l'orchestrateur. Les agents Dozzle sur les autres hôtes sont des conteneurs ordinaires et se mettent à jour comme n'importe lequel.

## Mettre à jour plusieurs conteneurs à la fois

Quand les actions sont activées, le tableau de bord vérifie tous les conteneurs en une seule passe. Les conteneurs obsolètes reçoivent un petit anneau à côté de leur nom, et un bouton **N mises à jour** apparaît au-dessus de la liste des conteneurs. Les deux ouvrent le panneau des mises à jour, qui liste tous les conteneurs disposant d'une image plus récente, tous sélectionnés. Décochez ceux que vous voulez laisser tels quels, puis appuyez sur **Mettre à jour**.

Les hôtes se mettent à jour en parallèle, et chaque hôte met à jour un conteneur à la fois, pour qu'aucun démon n'ait à récupérer une douzaine d'images en même temps. Le panneau montre chaque conteneur passer par les étapes de récupération, de recréation puis de mise à jour terminée, et l'échec d'un conteneur n'arrête pas les autres. La mise à jour s'exécute sur le serveur, donc fermer l'onglet ne l'interrompt pas. Rouvrir le panneau reprend là où elle en est.

Si le propre conteneur de Dozzle figure dans la liste, il passe toujours en dernier, car le mettre à jour redémarre Dozzle.

Avec `DOZZLE_IMAGE_CHECK_MODE=manual`, le bouton affiche **Rechercher des mises à jour** tant que vous ne l'avez pas pressé. Quand les actions sont désactivées, le tableau de bord reste exactement comme aujourd'hui, et le menu de chaque conteneur indique toujours quand une mise à jour est disponible.

## Mise à jour automatique des conteneurs {#auto-updating-containers}

Dozzle peut mettre à jour des conteneurs selon un planning. Activez-la pour un conteneur avec un label :

```yaml [docker-compose.yml]
services:
  whoami:
    image: traefik/whoami:latest
    labels:
      dev.dozzle.auto-update: true
```

Les conteneurs portant ce label suivent le même planning que [la mise à jour automatique de Dozzle](/fr/guide/setup-wizard#_4-mise-a-jour-automatique), que vous définissez dans l'assistant de configuration ou avec `DOZZLE_AUTO_UPDATE` et `DOZZLE_AUTO_UPDATE_TIME`. À cette heure, Dozzle compare chaque conteneur étiqueté à son registre et ne met à jour que ceux qui ont une image plus récente. Les conteneurs passent d'abord et Dozzle en dernier.

La mise à jour automatique est volontairement opt-in. Une base de données sur un tag flottant comme `postgres:latest` peut passer à une nouvelle version majeure dont elle ne sait pas lire les fichiers de données, donc n'ajoutez ce label qu'aux conteneurs que vous acceptez de voir remplacés sans surveillance. Les conteneurs que Dozzle [ne peut pas vérifier](#ce-qui-ne-peut-pas-etre-verifie), comme ceux d'un registre privé, ne sont jamais mis à jour automatiquement.

La mise à jour automatique fonctionne en mode serveur, y compris pour les conteneurs sur des [agents distants](/fr/guide/agent). Elle nécessite que les actions soient activées.
