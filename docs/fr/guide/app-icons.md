---
title: Icônes d'applications
sourceHash: b11b42409e46
---

# Icônes d'applications

Dozzle associe les images de conteneurs connues au logo de leur projet et l'affiche à côté du nom du conteneur dans la barre latérale, le tableau des conteneurs et la palette de commandes. Si vous utilisez une stack \*arr, Plex ou Home Assistant, la liste devient beaucoup plus rapide à parcourir.

Les icônes sont embarquées dans Dozzle. Elles ne sont jamais téléchargées depuis un CDN, donc rien concernant vos conteneurs ne sort de votre réseau et tout fonctionne hors ligne.

## Désactiver la fonctionnalité

Le réglage se trouve dans **Paramètres → Options → Afficher les icônes d'applications**. C'est un paramètre par profil, il ne s'applique donc qu'à votre navigateur.

## Comment fonctionne la correspondance

Dozzle regarde le nom de l'image, en ignorant le registre, le tag et le digest. Le dernier segment du chemin l'emporte, donc tous ces noms correspondent à Sonarr :

- `sonarr`
- `linuxserver/sonarr:latest`
- `lscr.io/linuxserver/sonarr`
- `ghcr.io/hotio/sonarr@sha256:...`

Quand le nom de l'image est générique, Dozzle se rabat sur l'espace de noms. C'est ainsi que `ghcr.io/goauthentik/server` correspond à Authentik.

## Remplacer l'icône

Certaines images ne correspondent à rien, et un fork peut tomber sur le mauvais logo. Définissez le label `dev.dozzle.icon` pour choisir vous-même une icône, ou mettez-le à `none` pour la masquer sur ce conteneur.

::: code-group

```sh
docker run --label dev.dozzle.icon=plex my-custom-media-server
```

```yaml [docker-compose.yml]
services:
  media:
    image: my-custom-media-server
    labels:
      - dev.dozzle.icon=plex

  scratch:
    image: alpine
    labels:
      - dev.dozzle.icon=none
```

:::

La valeur est un nom d'icône de [dashboard-icons](https://github.com/homarr-labs/dashboard-icons). Seules les icônes embarquées dans Dozzle sont disponibles. Un nom inconnu revient à ne pas afficher d'icône.

## Une icône manque ?

Dozzle embarque une sélection plutôt que les 3 000 icônes du jeu complet, pour garder l'image légère. Si une icône populaire manque, [ouvrez une issue](https://github.com/amir20/dozzle/issues) avec le nom de l'image et elle pourra être ajoutée.
