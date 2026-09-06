---
title: Démarrage
sourceHash: 6fa151a446d0
---

# Démarrage

Dozzle s'exécute dans un seul conteneur. Choisissez la CLI Docker, Docker Compose, Swarm ou Kubernetes ci-dessous.

## <Icon icon="mdi:docker" inline /> Docker autonome

Montez `docker.sock` pour que Dozzle puisse lire les conteneurs, montez un volume sur `/data` pour que les paramètres survivent à un redémarrage, et publiez le port 8080.

::: code-group

```sh [docker run]
docker run -d -v /var/run/docker.sock:/var/run/docker.sock -v dozzle_data:/data -p 8080:8080 amir20/dozzle:latest
```

```yaml [docker-compose.yml]
# Lancer avec docker compose up -d
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - dozzle_data:/data
    ports:
      - 8080:8080
    environment:
      # Décommentez pour activer les actions sur les conteneurs (arrêt, démarrage, redémarrage). Voir https://dozzle.dev/guide/actions
      # - DOZZLE_ENABLE_ACTIONS=true
      #
      # Décommentez pour autoriser l'accès aux shells des conteneurs. Voir https://dozzle.dev/guide/shell
      # - DOZZLE_ENABLE_SHELL=true
      #
      # Décommentez pour activer l'authentification. Voir https://dozzle.dev/guide/authentication
      # - DOZZLE_AUTH_PROVIDER=simple
      #
      # Nommez cette instance Dozzle (affiché dans l'en-tête et le menu multi-hôtes). Voir https://dozzle.dev/guide/hostname
      # - DOZZLE_HOSTNAME=my-server
      #
      # Connectez-vous à un ou plusieurs agents distants pour surveiller d'autres hôtes Docker. Voir https://dozzle.dev/guide/agent
      # - DOZZLE_REMOTE_AGENT=192.168.1.10:7007,192.168.1.11:7007
      #
      # N'affichez que les conteneurs correspondant à un filtre. Voir https://dozzle.dev/guide/filters
      # - DOZZLE_FILTER=label=com.example.app
volumes:
  dozzle_data:
```

:::

Ouvrez `http://localhost:8080` et c'est terminé. Tout le reste, y compris les actions, l'accès au shell, l'authentification et les agents distants, est facultatif et désactivé par défaut. Les variables d'environnement commentées dans le fichier Compose renvoient vers chaque guide.

> [!WARNING]
> Monter `docker.sock` donne à Dozzle un accès équivalent à root sur l'hôte. Si vous comptez exposer Dozzle au-delà de votre réseau privé, lisez d'abord les [considérations de sécurité](/fr/guide/authentication#security-considerations).

Dozzle nécessite Docker Engine 19.03 ou plus récent (API version 1.40+). Si Docker Hub est bloqué sur votre réseau, récupérez plutôt `ghcr.io/amir20/dozzle:latest` depuis le [GitHub Container Registry](https://ghcr.io/amir20/dozzle:latest).

## <Icon icon="mdi:hexagon-multiple-outline" inline /> Docker Swarm

Dozzle peut fonctionner en mode Swarm en étant déployé sur chaque nœud. Pour lancer Dozzle en mode Swarm, utilisez la configuration suivante :

```yaml [dozzle-stack.yml]
# Lancer avec docker stack deploy -c dozzle-stack.yml <name>
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      - DOZZLE_MODE=swarm
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
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

Vous pouvez ensuite déployer la stack avec la commande suivante :

```bash
docker stack deploy -c dozzle-stack.yml <name>
```

Voir le [mode Swarm](/fr/guide/swarm-mode) pour plus d'informations.

## <Icon icon="mdi:kubernetes" inline /> K8s

Dozzle peut fonctionner dans Kubernetes. Il suffit de le déployer sur un seul nœud du cluster. Vous devrez définir `DOZZLE_MODE=k8s` et configurer le RBAC pour l'accès aux logs des pods.

Voir le [mode Kubernetes](/fr/guide/k8s) pour la configuration complète, y compris les manifestes RBAC, de déploiement et de service.
