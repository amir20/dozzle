---
title: Liens vers les conteneurs
sourceHash: d39d7f2119c5
---

# Liens vers les conteneurs

La plupart des conteneurs qui méritent d'être surveillés exposent aussi une interface web. Ajoutez un label `dev.dozzle.url` et Dozzle affiche un lien à côté du nom du conteneur, pour passer des logs à l'application elle-même.

::: code-group

```sh
docker run --label dev.dozzle.url=https://grafana.example.com grafana/grafana
```

```yaml [docker-compose.yml]
services:
  grafana:
    image: grafana/grafana
    labels:
      - dev.dozzle.url=https://grafana.example.com
```

:::

Le lien apparaît à trois endroits : la barre latérale, le tableau des conteneurs et la barre de titre de la page du conteneur. Il s'ouvre toujours dans un nouvel onglet, et un clic ne vous fait jamais quitter les logs.

## Ce que le label accepte

La valeur doit être une URL absolue en `http` ou `https`. Tout le reste, un chemin relatif, un simple nom d'hôte ou un autre schéma, est ignoré et aucun lien n'est affiché.

Dozzle ne vérifie pas que l'URL est résolvable, et ne la réécrit pas selon l'hôte. Ce que vous écrivez est ce que le lien ouvre, utilisez donc une adresse qui fonctionne depuis le navigateur où vous consultez Dozzle.

## Pourquoi il n'y a pas de détection automatique

Dozzle connaît les ports publiés par un conteneur, mais un port publié n'est pas une URL accessible. Les reverse proxies, les chemins personnalisés, TLS et les réseaux séparés font que la devinette est fausse assez souvent pour être agaçante. Le label garde les choses explicites : Dozzle n'affiche que le lien que vous avez écrit.

Pour les conteneurs qui n'ont pas encore de label, Dozzle affiche une discrète icône de lien à côté du nom, sur le tableau de bord comme sur la page du conteneur. Elle ouvre un extrait à copier dans votre fichier compose, prérempli avec une suggestion. La fermer masque l'indication partout.

La suggestion vient de deux sources. Les labels de routeur Traefik sont lus en premier, car une règle de routeur nomme une adresse qui atteint réellement le conteneur depuis un navigateur :

```yaml
labels:
  - traefik.http.routers.grafana.rule=Host(`grafana.example.com`)
  - traefik.http.routers.grafana.tls=true
```

Les préfixes de chemin sont ajoutés, le schéma suit les réglages TLS et les points d'entrée du routeur, et `traefik.enable=false` désactive tout. À défaut, Dozzle se rabat sur un port publié sur l'hôte, associé au nom d'hôte depuis lequel vous consultez Dozzle. Les deux ne servent qu'à préremplir l'extrait. Aucun ne devient un lien tant que vous ne l'écrivez pas vous-même dans `dev.dozzle.url`.

Les conteneurs derrière un reverse proxy ne publient aucun port sur l'hôte, les labels Traefik sont donc souvent le seul signal disponible. Si vous utilisez un autre proxy, l'indication reste masquée et vous ajoutez le label à la main.

## Swarm

En Swarm, `deploy.labels` définit les labels sur le service et la clé `labels` de premier niveau les définit sur le conteneur. Le provider swarm de Traefik lit les labels du service, c'est donc là que tout le monde les met :

```yaml
services:
  ui:
    image: my/ui
    deploy:
      labels:
        - traefik.http.routers.ui.rule=Host(`app.example.com`)
        - dev.dozzle.url=https://app.example.com
```

Dozzle reporte les labels du service sur chaque conteneur de tâche, donc `dev.dozzle.url` et l'indication Traefik fonctionnent depuis `deploy.labels`. Il en va de même pour `dev.dozzle.name`, `dev.dozzle.group` et `dev.dozzle.icon`. Un label défini sur le conteneur lui-même l'emporte sur celui du service.

Lister les services est une API réservée aux managers. Dans un swarm multi-nœuds, les agents des nœuds workers ne peuvent pas lire les labels de service, les conteneurs qui y sont planifiés ne voient donc que les leurs.

## Labels associés

- [`dev.dozzle.name`](/fr/guide/container-names) définit un nom d'affichage personnalisé
- [`dev.dozzle.group`](/fr/guide/container-groups) regroupe des conteneurs
- [`dev.dozzle.icon`](/fr/guide/app-icons) choisit l'icône d'application
