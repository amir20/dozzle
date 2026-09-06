---
title: Configuration des hôtes distants
sourceHash: 6ae09165c838
---

# Configuration des hôtes distants

<Badge type="warning" text="Docker Only" />

Dozzle peut se connecter à des hôtes Docker distants. C'est utile quand Dozzle tourne dans un conteneur et que vous voulez surveiller un autre hôte Docker.

Cela dit, avec les agents Dozzle, vous pouvez vous connecter à des hôtes distants sans exposer le socket Docker. Voir la page [agent](/fr/guide/agent) pour plus d'informations.

Les agents Dozzle suppriment le besoin d'exposer le socket Docker à distance, mais ils ne peuvent pas être utilisés avec un proxy de socket Docker à l'intérieur de la stack de l'agent Dozzle. Si vous souhaitez utiliser un proxy de socket seul, sans agent, voir la section [se connecter avec un proxy de socket](#connecting-with-a-socket-proxy).

> [!WARNING]
> Les hôtes distants ont été remplacés par les agents. Les agents offrent un moyen plus sûr de se connecter à des hôtes distants. Bien que les hôtes distants soient toujours pris en charge, il est recommandé d'utiliser les agents. Voir la page [agent](/fr/guide/agent) pour plus d'informations et des exemples. Pour une comparaison, voir la section [comparaison des agents et des connexions distantes](/fr/guide/agent#comparing-agents-with-remote-connection). Je ne pourrai pas enquêter sur les problèmes liés aux hôtes distants, cela prend beaucoup trop de temps.

## Se connecter à des hôtes distants avec TLS

Les hôtes distants se configurent avec `--remote-host` ou `DOZZLE_REMOTE_HOST`. Tous les certificats doivent être montés dans le répertoire `/certs`. Le répertoire `/certs` doit contenir `/certs/{ca,cert,key}.pem`, ou `/certs/{host}/{ca,cert,key}.pem` en cas d'hôtes multiples.

Notez que la valeur `{host}` désigne ici l'IP ou le FQDN configuré, et non le [label facultatif](#adding-labels-to-hosts).

Plusieurs options `--remote-host` peuvent être utilisées pour indiquer plusieurs hôtes. En revanche, avec `DOZZLE_REMOTE_HOST`, la valeur doit être séparée par des virgules.

::: code-group

```sh [cli]
$ docker run -v /var/run/docker.sock:/var/run/docker.sock -v /path/to/certs:/certs -p 8080:8080 amir20/dozzle --remote-host tcp://167.99.1.1:2376 --remote-host tcp://167.99.1.2:2376
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /path/to/certs:/certs
    ports:
      - 8080:8080
    environment:
      DOZZLE_REMOTE_HOST: tcp://167.99.1.1:2376,tcp://167.99.1.2:2376
```

:::

## Se connecter avec un proxy de socket

Si vous êtes sur un réseau privé, vous pouvez utiliser [Docker Socket Proxy](https://github.com/Tecnativa/docker-socket-proxy), qui expose le fichier `docker.sock` sans avoir besoin de TLS. Cela supprime le besoin d'un agent Dozzle : Dozzle se connectera directement au proxy de socket. Dozzle n'essaiera jamais d'écrire dans Docker, mais il a besoin d'accéder aux API de listing. La commande suivante démarre un proxy avec un accès minimal :

```sh
$ docker container run --privileged -e CONTAINERS=1 -e INFO=1 -v /var/run/docker.sock:/var/run/docker.sock -p 2375:2375 tecnativa/docker-socket-proxy
```

> [!TIP]
> `CONTAINERS=1` est nécessaire pour lister les conteneurs en cours d'exécution. `EVENTS` est également requis, mais il est activé par défaut. `INFO=1` est nécessaire pour lister les informations système.

Lancer Dozzle sans aucun certificat devrait fonctionner. Voici un exemple :

::: code-group

```sh [cli]
$ docker run -p 8080:8080 amir20/dozzle --remote-host tcp://123.1.1.1:2375
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    ports:
      - 8080:8080
    environment:
      DOZZLE_REMOTE_HOST: tcp://123.1.1.1:2375
```

:::

Avec un hôte distant, monter `/var/run/docker.sock` est facultatif. Il vous faut au moins un hôte distant auquel vous connecter.

> [!WARNING]
> Docker Socket Proxy expose l'API Docker à Internet. Cela peut représenter un risque de sécurité si ce n'est pas correctement protégé.

## Ajouter des labels aux hôtes

`--remote-host` accepte des labels d'hôte en les ajoutant à la chaîne de connexion avec `|`. Par exemple, `--remote-host tcp://123.1.1.1:2375|foobar.com` utilisera foobar.com comme label dans l'interface. Voici un exemple complet avec la CLI ou Compose :

::: code-group

```sh
docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --remote-host tcp://123.1.1.1:2375|foobar.com
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /path/to/certs:/certs
    ports:
      - 8080:8080
    environment:
      DOZZLE_REMOTE_HOST: tcp://167.99.1.1:2376|foo.com,tcp://167.99.1.2:2376|bar.com
```

:::

> [!WARNING]
> Dozzle utilise l'API Docker pour collecter des informations sur les hôtes. Chaque agent a besoin d'un identifiant d'hôte unique. Ils utilisent l'identifiant système de Docker ou l'identifiant de nœud pour identifier l'hôte. Si vous utilisez Swarm, c'est l'identifiant de nœud qui est utilisé. Si vous ne voyez pas tous vos hôtes, vous avez peut-être configuré des hôtes en double partageant le même identifiant. Pour corriger cela, supprimez le fichier `/var/lib/docker/engine-id`. Voir la [FAQ](/fr/guide/faq#i-am-seeing-duplicate-hosts-error-in-the-logs-how-do-i-fix-it) pour plus d'informations.

## Changer le label de localhost

`localhost` est une connexion particulière qui utilise une configuration différente de `--remote-host`. Le label de localhost se change avec l'option `--hostname` ou la variable d'environnement `DOZZLE_HOSTNAME`. Voir la page [hostname](/fr/guide/hostname) pour des exemples d'utilisation.
