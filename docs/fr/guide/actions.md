---
title: Actions sur les conteneurs
sourceHash: 0b048a644851
---

# Actions sur les conteneurs

<Badge type="warning" text="Docker Only" />

Dozzle propose des actions sur les conteneurs, qui vous permettent de les démarrer (`start`), arrêter (`stop`), redémarrer (`restart`), supprimer (`remove`) et mettre à jour (`update`) depuis le menu déroulant à droite, à côté des statistiques du conteneur. Cette fonctionnalité est **désactivée** par défaut et s'active en mettant la variable d'environnement `DOZZLE_ENABLE_ACTIONS` à `true`.

L'action `update` récupère la dernière image du conteneur et le recrée avec la même configuration, ce qui est pratique pour mettre à niveau un conteneur sur place sans modifier son fichier compose. `update` n'a un effet réel que si l'image utilise un tag mouvant (par ex. `latest`, `stable`) ; avec un tag figé, la même image sera simplement retéléchargée.

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

## Mises à jour {#auto-updating-containers}

Chaque mise à jour conserve l'ancien conteneur jusqu'à ce que le nouveau reste en marche, et le remet en place sinon. La vérification des nouvelles images, la mise à jour de plusieurs conteneurs à la fois, le planning de mise à jour automatique, le nettoyage et la restauration sont décrits dans [Mises à jour des conteneurs](/fr/guide/updates).
