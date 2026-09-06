---
title: Nom d'hôte
sourceHash: 8769ba2c0e47
---

# Changer le nom d'hôte de Dozzle

Par défaut, la connexion de Dozzle s'appelle localhost. L'option `--hostname` permet de lui donner n'importe quel nom. Cette valeur apparaît dans le titre de la page et sous le logo Dozzle.

Cela change également le libellé de la connexion `localhost` affichée dans le menu multi-hôtes. L'exemple ci-dessous utilise `--hostname` pour remplacer le sous-titre par `mywebsite.xyz`.

::: code-group

```sh
docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --hostname mywebsite.xyz
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    ports:
      - 8080:8080
    environment:
      DOZZLE_HOSTNAME: mywebsite.xyz
```

:::

## Multi-hôtes et agents

`--hostname` ne renomme que l'hôte sur lequel tourne **ce** processus Dozzle. Les [agents](/fr/guide/agent) distants annoncent leur propre nom : définissez `DOZZLE_HOSTNAME` (ou `--hostname`) sur chaque agent pour choisir comment il apparaît dans le menu multi-hôtes. En [mode Swarm](/fr/guide/swarm-mode), chaque nœud exécute son propre agent, donnez donc un nom d'hôte distinct à chacun pour les différencier.
