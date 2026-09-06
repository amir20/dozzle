---
title: FAQ
sourceHash: c964c824dcab
---

# Foire aux questions

## Dozzle ne démarre pas et affiche `client version 1.x is too new`. Qu'est-ce que cela veut dire ?

Dozzle nécessite Docker Engine 19.03 ou plus récent (API version 1.40+). Les démons plus anciens, par exemple Docker 18.06 (API 1.38), ne sont pas pris en charge par le SDK Docker sous-jacent et échouent au démarrage avec une erreur du type `failed to create docker client: ... client version 1.54 is too new. Maximum supported API version is 1.38`.

Mettez Docker Engine à jour vers une version prise en charge. En solution temporaire, figez Dozzle sur `v10.5.2` ou une version antérieure, qui utilisait un SDK Docker capable de négocier avec les anciennes versions de l'API.

## Comment mettre à jour Dozzle ?

Dozzle suit les pratiques habituelles des images Docker. Pour mettre à jour, récupérez la nouvelle image et recréez le conteneur :

```sh
docker pull amir20/dozzle:latest
docker compose up -d dozzle
```

Les paramètres utilisateur, les règles de notification et le reste de l'état sont stockés dans `/data` (voir plus bas), gardez donc ce volume monté d'une mise à jour à l'autre. En production, figez un tag précis (par exemple `amir20/dozzle:v10.9.2`) plutôt que `latest`, pour que les mises à jour soient délibérées. Les notes de version sont publiées sur la [page des releases GitHub](https://github.com/amir20/dozzle/releases). Revenir en arrière revient simplement à redéployer un tag plus ancien.

## Ma plateforme remplace le point d'entrée du conteneur et Dozzle échoue avec `no such file or directory`

L'image par défaut est construite `FROM scratch`, elle contient donc le binaire Dozzle et rien d'autre. Pas de shell, pas d'interpréteur.

Certaines plateformes ajoutent des fonctionnalités optionnelles en montant un wrapper `#!/bin/sh` par-dessus le point d'entrée du conteneur, puis en réexécutant l'original. C'est le cas du bouton Tailscale par conteneur d'Unraid, ainsi que de certains injecteurs de sidecar et d'init. Sans `/bin/sh` dans l'image, le wrapper ne peut pas être exécuté et le conteneur s'arrête avec une erreur qui nomme le wrapper plutôt que le shell manquant :

```
exec /opt/unraid/tailscale: no such file or directory
```

Dans ces cas, utilisez la variante `alpine`, qui est le même binaire sur une base Alpine :

```sh
docker run \
  --volume=/var/run/docker.sock:/var/run/docker.sock \
  -p 8080:8080 \
  amir20/dozzle:alpine
```

Les tags versionnés suivent le même schéma (`amir20/dozzle:v10.9.2-alpine`). Le `latest` basé sur scratch reste l'image recommandée pour tout le reste, car son empreinte est bien plus petite et il n'y a aucun paquet de distribution à corriger.

## Qu'est-ce qui est stocké dans `/data` et comment le sauvegarder ?

Le dossier `/data` est l'endroit où Dozzle conserve tout ce qui doit survivre à un redémarrage du conteneur :

- `users.yml` / `users.yaml`, le fichier des utilisateurs de l'authentification simple (si vous en avez créé un)
- Les règles de notification, les destinations et l'état des envois
- Les paramètres d'interface par utilisateur (uniquement en mode multi-utilisateur ; en mode mono-utilisateur, ils sont dans le localStorage du navigateur)
- Un petit ensemble de fichiers internes, comme l'état des annonces masquées

Le dossier est petit (généralement bien en dessous de 10 Mo) et peut être sauvegardé avec un simple `tar` ou `rsync` du volume monté. Lors d'une mise à jour ou d'une migration vers un nouvel hôte, déplacer le volume `/data` emporte tous les paramètres avec lui.

## J'ai installé Dozzle, mais les logs sont lents ou ne se chargent jamais. Que faire ?

Dozzle utilise les Server Sent Events (SSE), qui se connectent à un serveur via un flux HTTP sans fermer la connexion. Si un proxy essaie de mettre ce flux en tampon, Dozzle ne reçoit jamais les données et attend indéfiniment que le reverse proxy vide son tampon. Depuis la version `1.23.0`, Dozzle envoie l'en-tête `X-Accel-Buffering: no`, qui devrait empêcher les reverse proxies de mettre en tampon. Certains proxies ignorent toutefois cet en-tête. Dans ce cas, vous devez désactiver explicitement toute mise en tampon.

### Désactiver la mise en tampon dans nginx

Voici un exemple avec nginx utilisant `proxy_pass` pour désactiver la mise en tampon :

```
server {
    ...

    location / {
        proxy_pass                  http://<dozzle.container.ip.address>:8080;
    }

    location /api {
        proxy_pass                  http://<dozzle.container.ip.address>:8080;

        proxy_buffering             off;
        proxy_cache                 off;
    }
}
```

### Désactiver la compression dans traefik

Le reverse proxy Traefik peut être configuré via des middlewares pour gérer la compression. Le cas échéant, la configuration ressemble en général à ceci :

```
http:
  middlewares:
    middlewares-compress:
      compress: {}
```

Avec cette configuration, vous pouvez constater que certains conteneurs n'affichent plus de logs dans dozzle si vous ouvrez dozzle via traefik (par exemple dozzle.mydomain.com).
Vous remarquerez aussi que la même instance dozzle affiche bien les logs en accès direct (par exemple localhost:8080).

Les conteneurs sur lesquels cela a été observé (liste non exhaustive) sont : dozzle, homepage, glances, filebrowser.

Pour que les logs circulent à nouveau, excluez `text/event-stream` du middleware de compression :

```
http:
  middlewares:
    middlewares-compress:
      compress:
        excludedContentTypes:
          - text/event-stream
```

## Nous avons des outils qui utilisent Dozzle à la création d'un conteneur. Comment obtenir un lien direct vers un conteneur par son nom ?

Dozzle dispose d'une [route](https://github.com/amir20/dozzle/blob/master/assets/pages/show.vue) spéciale qui permet de chercher un conteneur par son nom puis de rediriger vers lui. Par exemple, si vous avez un conteneur nommé `"foo.bar"` avec l'id `abc123`, vous pouvez envoyer vos utilisateurs sur `/show?name=foo.bar`, qui redirigera vers `/container/abc123`.

## J'ai installé Dozzle mais la consommation mémoire ne s'affiche pas !

_Ce problème est spécifique aux appareils ARM._

Dozzle utilise l'API Docker pour récupérer les informations d'utilisation mémoire des conteneurs. Si l'utilisation mémoire ne s'affiche pas, c'est probablement que l'API Docker ne la renvoie pas.

Vous pouvez le vérifier en lançant docker info, où vous devriez voir ceci :

```
WARNING: No memory limit support
WARNING: No swap limit support
```

Dans ce cas, vous devez ajouter la ligne suivante à votre fichier `/boot/cmdline.txt` et redémarrer l'appareil :

```
cgroup_enable=cpuset cgroup_enable=memory cgroup_memory=1
```

## Je vois une erreur d'hôtes en double dans les logs. Comment la corriger ?

Si vous voyez l'erreur suivante dans les logs, c'est que vous avez peut-être des hôtes en double configurés avec le même identifiant d'hôte :

```
time="2024-07-10T13:35:53Z" level=warning msg="duplicate host ID: *********, Endpoint: 1.1.1.1:7007 found, skipping"
```

Dozzle utilise l'API Docker pour collecter les informations sur les hôtes. Chaque hôte doit avoir un identifiant unique. Cet identifiant sert à distinguer l'hôte dans l'interface. En mode swarm, Dozzle utilise l'identifiant de nœud renvoyé par `docker system info`. Si vous n'utilisez pas le mode swarm, Dozzle utilise l'identifiant système de `docker system info` comme identifiant d'hôte.

Il arrive que des VM soient restaurées depuis des sauvegardes avec le même identifiant d'hôte. Dozzle croit alors que l'hôte est déjà présent et ne l'ajoute pas à la liste. Pour corriger cela, supprimez le fichier `/var/lib/docker/engine-id`. Ce fichier contient l'identifiant d'hôte et est créé au démarrage du démon Docker.

## Je vois une erreur d'hôte introuvable dans les logs. Comment la corriger ?

C'est avant tout une erreur propre à Podman : contrairement à Docker, Podman ne crée pas de engine-id.
Si vous utilisez Docker, vérifiez que le fichier `engine-id` existe dans `/var/lib/docker` avec les bonnes permissions et qu'il contient bien l'UUID.

Pour résoudre l'erreur, suivez ces étapes :

1. Créez les dossiers : `mkdir -p /var/lib/docker`
2. Installez uuidgen si nécessaire
3. Générez un UUID avec uuidgen : `uuidgen > engine-id`

Le fichier engine-id devrait maintenant contenir un UUID.

Un exemple de configuration pour Ansible se trouve dans [Podman](/fr/guide/podman)

Il peut être nécessaire de nettoyer votre déploiement Dozzle existant sous Podman : arrêtez le conteneur et supprimez les données associées (conteneur/volumes). Vous pouvez ensuite redéployer le conteneur Dozzle et vos logs devraient s'afficher.

## Pourquoi ne vois-je que les conteneurs en cours d'exécution ? Comment voir les conteneurs arrêtés ?

Par défaut, Dozzle n'affiche que les conteneurs en cours d'exécution. Pour voir les conteneurs arrêtés, activez l'option `Afficher les conteneurs arrêtés` dans les paramètres. Cette option est désactivée par défaut afin de réduire le nombre de conteneurs affichés dans l'interface.

## Existe-t-il un moyen de synchroniser mes paramètres entre plusieurs instances de Dozzle ?

En mode mono-utilisateur, Dozzle stocke les paramètres dans le stockage local du navigateur. Ils ne sont donc disponibles que sur le navigateur où ils ont été définis. Pour que Dozzle puisse synchroniser les paramètres entre plusieurs instances, il doit savoir qui est l'utilisateur. En mode multi-utilisateur, Dozzle utilise le nom d'utilisateur pour stocker les paramètres sur le disque et les synchroniser entre plusieurs instances. Ces informations sont enregistrées dans le dossier `/data`. Si vous voulez synchroniser vos paramètres entre plusieurs instances, vous devez [activer](/fr/guide/authentication) le mode multi-utilisateur et fournir un nom d'utilisateur.

## Pourquoi Dozzle ne prend-il pas directement en charge les notifications Slack, Discord, Telegram, email, etc. ?

Par choix de conception, Dozzle n'impose rien sur la destination de vos alertes. Plutôt que d'embarquer des intégrations pour des plateformes de notification précises, Dozzle fournit des **webhooks** avec des modèles de payload personnalisables. Vous pouvez ainsi envoyer des alertes vers _n'importe quel_ service qui accepte des requêtes HTTP : Slack, Discord, Telegram, ntfy, PagerDuty, Opsgenie ou vos propres outils internes, sans attendre que Dozzle ajoute une prise en charge explicite.

Il y a plusieurs raisons à cette approche :

- **Universalité.** Les webhooks fonctionnent avec pratiquement toutes les plateformes de notification. Ajouter des intégrations spécifiques ne couvrirait qu'une fraction des besoins, là où les webhooks les couvrent tous.
- **Maintenance.** Chaque intégration apporte ses propres particularités d'API, ses flux d'authentification, ses limites de débit et ses changements cassants. Les prendre en charge rendrait les mainteneurs de Dozzle responsables du débogage de services tiers, ce qui dépasse le cadre d'un visualiseur de logs.
- **Simplicité.** Dozzle est un outil léger et ciblé pour consulter les logs Docker. Garder la couche de notification générique maintient une base de code réduite et un projet viable.

Si vous voulez une expérience plus intégrée avec des connexions plus riches aux fournisseurs (notifications web push, boutons d'action ntfy, etc.), [Dozzle Cloud](/fr/guide/dozzle-cloud) est fait pour ça.

Pour configurer des webhooks avec le service de votre choix, consultez le guide [Alertes et webhooks](/fr/guide/alerts-and-webhooks) : il inclut des modèles de payload intégrés pour Slack, Discord et ntfy, utilisables tels quels ou personnalisables.

## Pourquoi Dozzle maintient-il dockerd et containerd à un CPU légèrement élevé alors qu'aucun navigateur n'est connecté ?

Dozzle continue de streamer les statistiques des conteneurs jusqu'à 6 heures (2 heures sur Kubernetes) après la déconnexion du dernier navigateur, puis arrête de lui-même le collecteur de statistiques. C'est intentionnel. Les statistiques sont streamées en continu pour qu'à la réouverture de l'interface vous voyiez l'historique CPU et mémoire au lieu d'un graphique vide. Si le streaming s'arrêtait à la fermeture de l'onglet, il n'y aurait aucun historique à afficher.

Le coût est une consommation CPU faible et régulière dans dockerd et containerd, car l'API de statistiques de Docker fonctionne par interrogation. Redémarrer le conteneur Dozzle remet le minuteur à zéro immédiatement, c'est pourquoi un redémarrage ramène l'hôte au repos. Ce comportement n'est volontairement pas configurable. Un délai court casserait d'autres fonctionnalités qui supposent que les statistiques sont toujours streamées, donc le réduire irait à l'encontre de l'historique des statistiques.

## Mes instances Dozzle expirent en mode Swarm ou je ne vois pas tous mes nœuds Swarm derrière un load balancer. Comment corriger cela ?

En mode Swarm, les instances Dozzle peuvent avoir besoin de leur propre réseau overlay. Si vous constatez un comportement incohérent en vous connectant à différents nœuds Dozzle, envisagez d'ajouter un réseau overlay séparé qui ne contient que les instances Dozzle, comme ci-dessous :

```
services:
  logs:
    ...
    networks: [ traefik, dozzle ]
    ...

networks:
  dozzle:
    driver: overlay
  traefik:
    external: true
```

Le réseau externe `traefik` est le réseau overlay utilisé pour la découverte de services du load balancer, et nous avons créé un nouveau réseau overlay `dozzle` pour que les nœuds Dozzle communiquent entre eux.
