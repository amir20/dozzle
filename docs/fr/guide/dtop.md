---
title: dtop CLI
sourceHash: 88753cae7439
---

# dtop

`dtop` est un compagnon en ligne de commande pour Dozzle qui affiche en temps réel les conteneurs Docker qui tournent sur votre système. Voyez-le comme un `docker ps` enrichi que vous pouvez laisser ouvert dans un panneau tmux. Et quand vous avez besoin de l'historique complet des logs, de la recherche ou des graphiques, `dtop` vous permet de basculer directement dans Dozzle.

Il se connecte aux hôtes Docker via `ssh`, `tcp` ou une `unix socket` locale, ce qui le rend adapté aux mêmes configurations multi-hôtes que Dozzle.

![Capture d'écran de dtop](https://github.com/amir20/dtop/raw/master/demo.gif)

## Installation

Installation avec Homebrew :

```bash
brew install dtop
```

Ou exécutez-le via Docker sans rien installer :

```bash
docker run -v /var/run/docker.sock:/var/run/docker.sock -it ghcr.io/amir20/dtop:latest
```

Les instructions d'installation complètes sont disponibles sur [https://github.com/amir20/dtop](https://github.com/amir20/dtop?tab=readme-ov-file#installation).

## Périmètre

`dtop` est volontairement plus restreint que Dozzle. Il répond depuis le terminal à la question "qu'est-ce qui tourne en ce moment, et est-ce que quelque chose brûle", et passe la main à Dozzle pour tout ce qui demande un navigateur : historique des logs, recherche, requêtes SQL et graphiques de statistiques.

Il est développé dans son propre dépôt et publié à son propre rythme. Les suggestions et les rapports de bugs vont sur [https://github.com/amir20/dtop/issues](https://github.com/amir20/dtop/issues).
