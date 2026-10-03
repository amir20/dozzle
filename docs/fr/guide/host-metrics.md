---
title: Métriques de l'hôte
sourceHash: 5d6314385b49
---

# Métriques de l'hôte

La carte de l'hôte peut afficher trois indicateurs sur la machine où tourne Docker. Ils se trouvent dans un petit encadré à droite de l'en-tête de la carte, marqué d'une icône de pouls, pour qu'on ne les confonde pas avec les totaux des conteneurs affichés dans les jauges de CPU et de mémoire en dessous. Sur un téléphone, le disque devient une troisième jauge à côté du CPU et de la mémoire, et la durée de fonctionnement et la charge passent sur une ligne sous les jauges :

- **Durée de fonctionnement**, depuis combien de temps l'hôte est démarré
- **Charge**, la charge moyenne sur 1 minute, qui passe au jaune dès qu'elle dépasse le nombre de cœurs et au rouge au-delà du double (survolez-la pour voir celles sur 5 et 15 minutes ainsi que le nombre de cœurs)
- **Disque**, le taux de remplissage du système de fichiers qui contient le répertoire de données de Docker, sous forme d'une petite barre qui passe au jaune au-delà de 70 % et au rouge au-delà de 90 % (survolez-la pour voir l'espace utilisé et total)

Ils se rafraîchissent toutes les 15 secondes tant qu'un onglet est ouvert. Chacun n'apparaît que si Dozzle peut lire une vraie valeur, donc sur une installation par défaut l'encadré peut être absent ou n'en afficher qu'une partie.

## Dozzle dans un conteneur

Dans un conteneur, `/proc` décrit le conteneur et non l'hôte. Dozzle ne fera pas passer les chiffres du conteneur pour ceux de l'hôte, la charge et la durée de fonctionnement restent donc masquées tant que vous ne montez pas le `/proc` de l'hôte sur `/host/proc`.

::: code-group

```sh
docker run -d \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /proc:/host/proc:ro \
  -v dozzle_data:/data \
  -p 8080:8080 amir20/dozzle:latest
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /proc:/host/proc:ro
      - dozzle_data:/data
    ports:
      - 8080:8080
volumes:
  dozzle_data:
```

:::

Le disque ne demande aucun montage supplémentaire. Dozzle mesure le système de fichiers qui se trouve derrière son propre `/data`, lequel repose sur le disque de Docker, que `/data` soit un volume nommé, comme ci-dessus, ou qu'il ne soit pas monté du tout. Si vous y montez plutôt un dossier de l'hôte (`./data:/data`), l'indicateur décrit le disque sur lequel se trouve ce dossier, qui est en général le même.

## Dozzle en natif

Un binaire Dozzle lancé directement sur l'hôte lit `/proc` tel quel, la charge et la durée de fonctionnement ne demandent donc aucune configuration. Le disque est lu depuis le répertoire de données de Docker (`docker info | grep "Docker Root Dir"`, en général `/var/lib/docker`), il fonctionne donc tant que l'utilisateur sous lequel tourne Dozzle peut voir ce répertoire.

## Agents

Chaque [agent](/fr/guide/agent) lit sa propre machine et envoie les valeurs au Dozzle que vous consultez, si bien que la carte de chaque agent affiche sa propre durée de fonctionnement, sa charge et son disque. Donnez au conteneur de l'agent les mêmes montages que ceux que vous donneriez à Dozzle :

```yaml [docker-compose.yml]
services:
  dozzle-agent:
    image: amir20/dozzle:latest
    command: agent
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /proc:/host/proc:ro
    ports:
      - 7007:7007
```

Les disques supplémentaires fonctionnent de la même façon, montés sous `/host/disks` sur l'agent. En [mode Swarm](/fr/guide/swarm-mode), chaque nœud fait tourner Dozzle comme son propre agent, ajoutez donc les montages au service et chaque nœud remontera ses propres valeurs.

Un agent plus ancien que le Dozzle auquel il se rapporte n'envoie aucune métrique, et sa carte reste telle qu'elle était. Mettez l'agent à jour pour les voir.

## Plus de disques

Par défaut, le disque couvre celui de Docker. Pour surveiller aussi d'autres disques, montez chacun d'eux sous `/host/disks/<name>`. Le nom du dossier devient le libellé du disque.

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /proc:/host/proc:ro
      - /mnt/media:/host/disks/media:ro
      - /mnt/backup:/host/disks/backup:ro
```

La barre affiche alors le disque le plus rempli, puisque c'est lui qui sera plein en premier, et la survoler liste chaque disque avec son espace utilisé et total.

Dozzle n'a besoin que du point de montage pour mesurer un disque, pas de ses fichiers. Monter la racine d'un disque permet quand même à Dozzle de lire ce qu'il contient, donc si cela compte pour vous, créez un dossier vide sur le disque et montez-le à la place (`/mnt/media/.dozzle:/host/disks/media:ro`). Il se trouve sur le même système de fichiers et remonte les mêmes chiffres.

Une installation native peut faire de même avec des liens symboliques : `ln -s /mnt/media /host/disks/media`.

## Limites

- Un [hôte distant](/fr/guide/remote-hosts) connecté en TCP n'a rien de son côté pour lire la machine, il affiche donc le CPU et la mémoire comme avant, sans l'encadré. Faites plutôt tourner un agent dessus pour les obtenir.
- Si `DOCKER_HOST` pointe vers une autre machine (`tcp://` ou `ssh://`), Dozzle n'affiche pas ces indicateurs, puisque son propre `/proc` et ses propres disques ne disent rien de ce moteur.
- Docker Desktop fait tourner le moteur dans une VM. Un binaire Dozzle natif sur macOS ou Windows n'a pas de `/proc` à lire, et Dozzle dans un conteneur y remonte les chiffres de la VM, pas ceux de votre machine.
