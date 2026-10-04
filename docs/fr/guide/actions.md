---
title: Actions sur les conteneurs
sourceHash: 121d4b250806
---

# Actions sur les conteneurs

<Badge type="warning" text="Docker Only" />

Dozzle propose des actions sur les conteneurs, qui vous permettent de les démarrer (`start`), arrêter (`stop`), redémarrer (`restart`), supprimer (`remove`) et mettre à jour (`update`) depuis le menu déroulant à droite, à côté des statistiques du conteneur. Cette fonctionnalité est **désactivée** par défaut et s'active en mettant la variable d'environnement `DOZZLE_ENABLE_ACTIONS` à `true`.

L'action `update` récupère la dernière image du conteneur et le recrée avec la même configuration, ce qui est pratique pour mettre à niveau un conteneur sur place sans modifier son fichier compose. `update` n'a un effet réel que si l'image utilise un tag mouvant (par ex. `latest`, `stable`) ; avec un tag figé, la même image sera simplement retéléchargée.

L'ancien conteneur est conservé, renommé, jusqu'à ce que le nouveau ait tourné 10 secondes sans redémarrer et se soit déclaré sain si son image a un healthcheck. Si le nouveau conteneur ne démarre pas, s'arrête, redémarre ou devient non sain, Dozzle le supprime et remet l'ancien en place, et la mise à jour indique **restauré** avec la raison. Le nouveau conteneur reçoit le label `dev.dozzle.previous-image` avec l'id de l'image qu'il remplace, et `dev.dozzle.previous-ref` avec le digest `repo@sha256:…` de cette image (absent pour les images construites localement). Un conteneur arrêté n'est jamais mis à jour, car il peut être arrêté exprès. Une image plus récente reste signalée, mais **Mettre à jour** n'est pas proposé, le planning l'ignore, et une mise à jour demandée malgré tout, depuis Dozzle Cloud par exemple, est refusée avec « Démarrez d'abord le conteneur ». Une fois démarré, il peut être mis à jour.

> [!WARNING]
> `remove` supprime le conteneur : les données de sa couche inscriptible sont perdues et ses volumes anonymes restent derrière lui, détachés. `update` recrée le conteneur et conserve tous les volumes, anonymes compris, ainsi que tous les bind mounts. Seules les données écrites dans la couche inscriptible du conteneur sont perdues.

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

Pour ne plus vérifier un seul conteneur, par exemple un conteneur volontairement figé sur une version, ajoutez-lui ce label. Il sort aussi du [planning de mise à jour automatique](#auto-updating-containers).

```yaml [docker-compose.yml]
services:
  database:
    image: postgres:18-alpine
    labels:
      dev.dozzle.update: off
```

Une notification peut aussi être affichée quand une mise à jour est trouvée. Elle est désactivée par défaut et se trouve dans les paramètres.

### Ce qui ne peut pas être vérifié

Certains conteneurs n'ont rien à comparer, et Dozzle reste silencieux plutôt que de deviner :

- Les images construites localement, qui n'ont pas d'empreinte de registre
- Les références figées sur une empreinte, qui ne peuvent pas changer
- Les registres privés, puisque Dozzle n'a pas d'identifiants propres. Dans Kubernetes, cela inclut les images récupérées avec `imagePullSecrets`.

### Kubernetes

En mode Kubernetes, la vérification compare le digest enregistré dans le statut du pod avec celui du registre. Elle fonctionne donc sur les clusters containerd comme k3s, EKS et GKE sans permission supplémentaire. La notification est purement informative et n'est jamais accompagnée d'un bouton `Update`, car l'image d'un pod appartient à son workload. Pour un tag mouvant comme `:latest` avec `imagePullPolicy: Always`, un [redémarrage progressif](/fr/guide/k8s#rollout-restart) récupère la nouvelle image. Tout le reste passe par une modification de la spec du workload. Le bouton et le tiroir des mises à jour du tableau de bord ne s'affichent pas en mode Kubernetes.

### Mettre à jour Dozzle lui-même

L'action `Update` sur le propre conteneur de Dozzle met Dozzle à jour sur place. Elle récupère la nouvelle image et confie le remplacement à un conteneur auxiliaire éphémère, donc Dozzle disparaît quelques secondes et revient dans la nouvelle version avec la même configuration et les mêmes volumes. Elle peut aussi être planifiée. Consultez [Comment Dozzle se met à jour lui-même](/fr/guide/setup-wizard#self-update) pour savoir ce qui est conservé et quelles installations ne sont pas prises en charge. Faire tourner Dozzle en service Swarm le met à jour via l'orchestrateur. Les agents Dozzle sur les autres hôtes sont des conteneurs ordinaires et se mettent à jour comme n'importe lequel.

## Mettre à jour plusieurs conteneurs à la fois

Quand les actions sont activées, le tableau de bord vérifie tous les conteneurs en une seule passe. Les conteneurs obsolètes reçoivent un petit anneau à côté de leur nom, et un bouton **N mises à jour** apparaît au-dessus de la liste des conteneurs. Les deux ouvrent le panneau des mises à jour, qui liste tous les conteneurs disposant d'une image plus récente, tous sélectionnés. Décochez ceux que vous voulez laisser tels quels, puis appuyez sur **Mettre à jour**.

Les hôtes se mettent à jour en parallèle, et chaque hôte met à jour un conteneur à la fois, pour qu'aucun démon n'ait à récupérer une douzaine d'images en même temps. Le panneau montre chaque conteneur passer par les étapes de récupération, de recréation puis de mise à jour terminée, et l'échec d'un conteneur n'arrête pas les autres. La mise à jour s'exécute sur le serveur, donc fermer l'onglet ne l'interrompt pas. Rouvrir le panneau reprend là où elle en est.

Si le propre conteneur de Dozzle figure dans la liste, il passe toujours en dernier, car le mettre à jour redémarre Dozzle.

Avec `DOZZLE_IMAGE_CHECK_MODE=manual`, le bouton affiche **Rechercher des mises à jour** tant que vous ne l'avez pas pressé. Quand les actions sont désactivées, le tableau de bord reste exactement comme aujourd'hui, et le menu de chaque conteneur indique toujours quand une mise à jour est disponible.

## Mise à jour automatique des conteneurs {#auto-updating-containers}

Dozzle peut mettre à jour des conteneurs selon un planning. Cela se règle dans **Paramètres → Mises à jour** ou dans l'[assistant de configuration](/fr/guide/setup-wizard#auto-update) :

- **Quand :** désactivé, tous les jours ou chaque semaine le dimanche, à une heure donnée. Équivalent à `DOZZLE_AUTO_UPDATE` et `DOZZLE_AUTO_UPDATE_TIME`.
- **Quels conteneurs :** **Dozzle seulement**, **Conteneurs avec label** (par défaut) ou **Tout**. Dozzle lui-même suit le planning dans les trois cas. Équivalent à `DOZZLE_UPDATE_CONTAINERS` (`off`, `labelled` ou `all`).

Un label sur le conteneur décide du reste :

| `dev.dozzle.update` | Ce qui se passe                                                                                                      |
| ------------------- | -------------------------------------------------------------------------------------------------------------------- |
| `auto`              | Mis à jour selon le planning, sauf si **Quels conteneurs** vaut **Dozzle seulement**                                 |
| _(aucun label)_     | Mis à jour selon le planning avec **Tout**. Sinon vérifié et affiché comme mise à jour, que vous appliquez vous-même |
| `off`               | Jamais vérifié, jamais mis à jour                                                                                    |

```yaml
services:
  app:
    image: ghcr.io/example/app:latest
    labels:
      dev.dozzle.update: auto
```

Les anciens labels fonctionnent toujours : `dev.dozzle.auto-update=true` est lu comme `auto`, et `dev.dozzle.update-check=false` comme `off`.

À l'heure prévue, Dozzle compare chaque conteneur du planning à son registre et ne met à jour que ceux qui ont une image plus récente, Dozzle lui-même en dernier. Chaque mise à jour est l'échange sûr décrit plus haut : si le nouveau conteneur ne reste pas en marche, l'ancien est remis en place. Les conteneurs arrêtés ou en mauvaise santé, ceux que Dozzle [ne peut pas vérifier](#ce-qui-ne-peut-pas-etre-verifie) et ceux qui ont été [restaurés](#rolling-back) depuis l'image proposée sont ignorés. **Paramètres → Mises à jour** liste les conteneurs que la prochaine exécution mettra à jour.

Avec **Tout**, une base de données sur un tag flottant comme `postgres:latest` peut passer à une version majeure dont elle ne sait pas lire les fichiers de données. Choisir **Tout** liste les conteneurs qui gardent des données dans des volumes nommés. Ajoutez-leur le label `dev.dozzle.update: off` pour les exclure.

**Quels conteneurs** est enregistré dans [`dozzle.yml`](/fr/guide/setup-wizard#dozzle-yml) sous `updateContainers`, donc le modifier depuis l'interface nécessite `/data` sur un volume. La mise à jour automatique fonctionne en mode serveur, y compris pour les conteneurs des [agents distants](/fr/guide/agent), et nécessite les actions. Vous venez de Watchtower ? Consultez [Passer de Watchtower à Dozzle](/fr/guide/moving-from-watchtower).

## Nettoyer les anciennes images {#cleaning-up-old-images}

Chaque mise à jour laisse sur l'hôte l'image qu'elle remplace, donc Dozzle supprime les anciennes images après une mise à jour, comme le `--cleanup` de Watchtower. Le nettoyage a toujours lieu, pour toutes les mises à jour : planifiées, lancées depuis l'action `Update` d'un conteneur ou depuis le panneau des mises à jour. Il n'y a rien à activer.

Dozzle garde l'image sur laquelle le conteneur tournait jusque-là, pour qu'il puisse encore y revenir, et supprime celle d'avant. Une mise à jour de 1.4.1 vers 1.4.2 supprime 1.4.0 et garde 1.4.1, donc chaque conteneur garde au plus une image de réserve. Dozzle lit l'image à supprimer dans le label `dev.dozzle.previous-image` de l'ancien conteneur, donc la première mise à jour d'un conteneur ne supprime rien.

Le nettoyage n'a lieu qu'une fois la mise à jour aboutie et l'ancien conteneur supprimé. Une mise à jour restaurée ne supprime rien. Dozzle ne supprime qu'une image sans tag qu'aucun conteneur n'utilise : une image qui a encore un tag, par exemple une image que vous avez téléchargée ou construite vous-même, est conservée, et la suppression n'est pas forcée, donc Docker refuse tant qu'un autre conteneur, démarré ou arrêté, l'utilise encore. Un refus ne fait jamais échouer la mise à jour.

Les conteneurs sur des [agents distants](/fr/guide/agent) sont nettoyés de la même façon, tout comme le propre conteneur de Dozzle : le conteneur auxiliaire de la [mise à jour automatique](/fr/guide/setup-wizard#self-update) supprime l'image d'avant la précédente une fois que le nouveau Dozzle reste en marche. Les services Swarm, y compris Dozzle lorsqu'il tourne comme service Swarm, ne sont pas nettoyés, car chaque nœud garde ses propres images et Swarm élague lui-même son historique de tâches.

## Restaurer la version précédente {#rolling-back}

Annuler une mise à jour est une fonctionnalité de [Dozzle Cloud](/fr/guide/dozzle-cloud). Dozzle Cloud surveille chaque mise à jour faite par le planning et, quand la nouvelle version commence à échouer, propose de revenir en arrière. Dozzle remet alors le conteneur sur l'image qu'il exécutait avant, celle que nomme son label `dev.dozzle.previous-image`, de la même façon qu'une mise à jour : le conteneur actuel est conservé jusqu'à ce que l'image précédente reste en marche, et remis en place sinon. Les réglages et les volumes ne changent pas. Rien n'est téléchargé : si l'image précédente n'est plus sur l'hôte, la restauration échoue et le conteneur reste tel quel.

Le conteneur restauré porte le label `dev.dozzle.rolled-back-from` avec l'image qu'il a quittée, et le planning de mise à jour automatique le laisse de côté jusqu'à ce que son tag pointe vers une image plus récente. Une fois la restauration stable, l'image quittée est [nettoyée](#cleaning-up-old-images) comme toute ancienne image, donc supprimée seulement quand plus aucun tag ne pointe vers elle.

La restauration fonctionne pour les conteneurs autonomes, y compris ceux des [agents distants](/fr/guide/agent). Elle n'est pas disponible pour les services Swarm, Kubernetes ni le conteneur de Dozzle lui-même.

## Les mises à jour dans les logs

Quand Dozzle met à jour un conteneur, selon le planning, depuis l'interface de Dozzle ou depuis Dozzle Cloud, les logs du nouveau conteneur commencent par un repère qui indique l'image de départ et d'arrivée et ce qui a lancé la mise à jour. Une restauration et une mise à jour annulée sont aussi repérées. Avec Dozzle Cloud relié, le repère montre aussi l'avis de Dozzle Cloud sur la mise à jour, avec un lien vers celle-ci. Dozzle garde ses mises à jour récentes en mémoire, donc le repère disparaît après un redémarrage de Dozzle.
