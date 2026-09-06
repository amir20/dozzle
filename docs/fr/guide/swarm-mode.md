---
title: Mode Swarm
sourceHash: f73672e30c85
---

# Mode Swarm

<Badge type="warning" text="Docker Only" />

Dozzle prend en charge le mode Docker Swarm. En mode Swarm, Dozzle découvre automatiquement les services et les groupes personnalisés. Dozzle n'utilise pas l'API Swarm en interne car elle est [limitée](https://github.com/moby/moby/issues/33183). À la place, Dozzle implémente son propre regroupement à partir des labels swarm. De plus, Dozzle fusionne les statistiques des conteneurs d'un même groupe. Vous pouvez donc voir les logs et les statistiques de tous les conteneurs d'un groupe dans une seule vue. En contrepartie, chaque hôte doit être équipé de Dozzle.

## <Icon icon="mdi:cogs" inline /> Comment ça marche ?

Une fois déployé en mode Swarm, Dozzle crée un réseau maillé sécurisé entre tous les nœuds du swarm. Ce réseau sert à la communication entre les différentes instances Dozzle. Le réseau maillé est créé avec [mTLS](https://www.cloudflare.com/learning/access-management/what-is-mutual-tls) et un certificat TLS privé. Toute la communication entre les instances Dozzle est donc chiffrée et peut être déployée n'importe où en toute sécurité.

Dozzle prend en charge les [stacks](https://docs.docker.com/reference/cli/docker/stack/deploy/), les [services](https://docs.docker.com/engine/swarm/how-swarm-mode-works/services/) Docker et les groupes personnalisés pour réunir les logs. Les labels `com.docker.stack.namespace` et `com.docker.compose.project` servent à regrouper les conteneurs. Pour les services, Dozzle utilise le nom du service comme nom de groupe, c'est-à-dire `com.docker.swarm.service.name`.

## <Icon icon="mdi:rocket-launch-outline" inline /> Comment activer le mode Swarm ?

Pour déployer sur chaque nœud du swarm, utilisez `mode: global`. Dozzle sera ainsi déployé sur tous les nœuds du swarm. Voici un exemple avec Docker Stack :

```yml
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      - DOZZLE_MODE=swarm
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /opt/dozzle/data:/data
    ports:
      - 8080:8080
    networks:
      - dozzle
    deploy:
      mode: global
networks:
  dozzle:
    driver: overlay
```

Notez que la variable d'environnement `DOZZLE_MODE` vaut `swarm`. Cela indique à Dozzle de découvrir automatiquement les autres instances Dozzle du swarm. Le réseau `overlay` sert à créer le réseau maillé entre les différentes instances Dozzle.

Le volume `/data` est monté pour conserver la configuration de Dozzle (notifications, paramètres cloud, stacks personnalisées). Comme Dozzle est déployé globalement sur chaque nœud, montez un chemin de l'hôte sur chaque nœud pour que chaque instance conserve son état local entre les redémarrages.

> [!WARNING]
> Un socket-proxy ne peut pas être utilisé en mode Docker Swarm. Cette limitation vient de Docker lui-même, pas de Dozzle. En mode Swarm, les services ne peuvent communiquer qu'avec d'autres services, alors que Dozzle a besoin de connexions directes vers chaque instance du proxy, ce qui n'est pas pris en charge. Si vous avez une solution pour utiliser un socket-proxy en mode Swarm, nous serions ravis de l'entendre !

## <Icon icon="mdi:shield-lock-outline" inline /> Mettre en place l'authentification simple en mode Swarm

Pour mettre en place l'authentification simple, vous pouvez utiliser les secrets Docker pour stocker le fichier `users.yml`. Voici un exemple avec Docker Stack :

```yml
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      - DOZZLE_LEVEL=debug
      - DOZZLE_MODE=swarm
      - DOZZLE_AUTH_PROVIDER=simple
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /opt/dozzle/data:/data
    secrets:
      - source: users
        target: /data/users.yml

    ports:
      - "8080:8080"
    networks:
      - dozzle
    deploy:
      mode: global

networks:
  dozzle:
    driver: overlay
secrets:
  users:
    file: users.yml
```

Dans cet exemple, le fichier `users.yml` est stocké dans un secret Docker. C'est identique à l'exemple de l'[authentification simple](/fr/guide/authentication#generating-users-yml).

## <Icon icon="mdi:server-plus-outline" inline /> Ajouter des agents autonomes au mode Swarm

Dozzle permet d'ajouter des [agents](/fr/guide/agent) autonomes lorsqu'il tourne en mode Swarm.

Il suffit d'[ajouter l'agent distant](/fr/guide/agent#how-to-connect-to-an-agent) à votre fichier compose Swarm comme vous le feriez normalement.

> [!NOTE]
> Les agents distants sont pris en charge, mais les connexions distantes comme le socket proxy ne le sont pas.

```yml
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      - DOZZLE_MODE=swarm
      - DOZZLE_REMOTE_AGENT=agent:7007
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /opt/dozzle/data:/data
    ports:
      - 8080:8080
    networks:
      - dozzle
    deploy:
      mode: global
networks:
  dozzle:
    driver: overlay
```

Le ou les agents distants apparaîtront maintenant à côté des autres nœuds dans Dozzle.
