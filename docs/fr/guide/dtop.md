---
title: Présentation de dtop
sourceHash: 3137db243510
---

# Qu'est-ce que dtop ?

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

## État du projet

`dtop` est un projet récent et n'est pas aussi complet que Dozzle. Je travaille cependant activement à y ajouter des fonctionnalités. Je m'en sers personnellement pour surveiller tous mes conteneurs sur plusieurs hôtes depuis la ligne de commande. Si vous avez des suggestions, ouvrez une issue sur [https://github.com/amir20/dtop/issues](https://github.com/amir20/dtop/issues).
