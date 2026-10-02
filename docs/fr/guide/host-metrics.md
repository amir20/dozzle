---
title: Métriques de l'hôte
sourceHash: e10c0661efbb
---

# Métriques de l'hôte

La carte de l'hôte peut afficher trois indicateurs sur la machine où tourne Docker. Ils se trouvent dans un petit encadré à droite de l'en-tête de la carte, marqué de l'icône de l'hôte, pour qu'on ne les confonde pas avec les totaux des conteneurs affichés dans les jauges de CPU et de mémoire en dessous :

- **Durée de fonctionnement**, depuis combien de temps l'hôte est démarré
- **Charge**, la charge moyenne sur 1 minute (survolez-la pour voir celles sur 5 et 15 minutes)
- **Disque**, le taux de remplissage du système de fichiers qui contient le répertoire de données de Docker, sous forme d'une petite barre qui passe au jaune au-delà de 70 % et au rouge au-delà de 90 % (survolez-la pour voir l'espace utilisé et total)

Ils se rafraîchissent toutes les 15 secondes tant qu'un onglet est ouvert. Chacun n'apparaît que si Dozzle peut lire une vraie valeur, donc sur une installation par défaut l'encadré peut être absent ou n'en afficher qu'une partie.

## Dozzle dans un conteneur

Dans un conteneur, `/proc` décrit le conteneur et non l'hôte. Dozzle ne fera pas passer les chiffres du conteneur pour ceux de l'hôte, la charge et la durée de fonctionnement restent donc masquées tant que vous ne montez pas le `/proc` de l'hôte sur `/host/proc`.

Le disque est lu depuis le répertoire de données de Docker (`docker info --format '{{.DockerRootDir}}'`, en général `/var/lib/docker`). Montez-le au même chemin pour obtenir l'indicateur de disque.

::: code-group

```sh
docker run -d \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /proc:/host/proc:ro \
  -v /var/lib/docker:/var/lib/docker:ro \
  -p 8080:8080 amir20/dozzle:latest
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /proc:/host/proc:ro
      - /var/lib/docker:/var/lib/docker:ro
    ports:
      - 8080:8080
```

:::

Les deux montages sont facultatifs et indépendants. Monter le répertoire de données donne à Dozzle un accès en lecture au système de fichiers de tous les conteneurs, laissez-le donc de côté si seules la charge et la durée de fonctionnement vous intéressent.

## Dozzle en natif

Un binaire Dozzle lancé directement sur l'hôte lit `/proc` tel quel, la charge et la durée de fonctionnement ne demandent donc aucune configuration. Le disque fonctionne tant que l'utilisateur sous lequel tourne Dozzle peut voir le répertoire de données.

## Limites

- Pour l'instant, seul l'hôte local remonte ces métriques. Les hôtes connectés via un [agent](/fr/guide/agent) ou en tant qu'[hôte distant](/fr/guide/remote-hosts) affichent le CPU et la mémoire comme avant, sans cette ligne.
- Si `DOCKER_HOST` pointe vers une autre machine (`tcp://` ou `ssh://`), Dozzle n'affiche pas ces indicateurs, puisque son propre `/proc` et ses propres disques ne disent rien de ce moteur.
- Docker Desktop fait tourner le moteur dans une VM. Un binaire Dozzle natif sur macOS ou Windows n'a pas de `/proc` à lire, et Dozzle dans un conteneur y remonte les chiffres de la VM, pas ceux de votre machine.
