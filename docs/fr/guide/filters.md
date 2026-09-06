---
title: Filtres
sourceHash: e380a612fd7f
---

# Filtrer les conteneurs

<Badge type="tip" text="Docker" />
<Badge type="tip" text="K8s" />

Dozzle prend en charge le filtrage conditionnel, comme l'option [--filter](https://docs.docker.com/reference/cli/docker/container/ls/#filter) de Docker, via `DOZZLE_FILTER` ou `--filter`. Les filtres sont transmis directement à Docker pour limiter ce que Dozzle peut voir. Par exemple, le filtrage par label se fait avec `--filter "label=color"`, ce qui équivaut à la commande `docker ps --filter "label=color"`.

::: code-group

```sh
docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --filter label=color
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
      DOZZLE_FILTER: label=color
```

:::

Les filtres les plus courants sont `name` et `label`, pour restreindre l'accès de Dozzle aux conteneurs.

## Filtres d'interface, d'agent et d'utilisateur

Dozzle accepte plusieurs filtres pour limiter les conteneurs visibles. Ils peuvent être définis au niveau de l'interface, de l'agent ou de l'utilisateur.

1. **Filtres d'interface** : ils s'appliquent à l'instance Dozzle et sont envoyés à Docker pour restreindre les conteneurs visibles. Ils affectent tous les agents et tous les utilisateurs qui n'ont pas leurs propres filtres.
2. **Filtres d'agent** : ils sont définis au niveau de l'agent et envoyés à Docker pour limiter les conteneurs exposés par cet agent. Les filtres d'agent et d'interface se combinent pour restreindre les conteneurs.
3. **Filtres d'utilisateur** : ils sont définis au niveau de l'utilisateur et déterminent les conteneurs qu'il peut voir. S'ils ne sont pas définis, Dozzle utilise par défaut les filtres d'interface.

Pour en savoir plus sur les filtres propres à un utilisateur, voir [filtres utilisateur](/fr/guide/authentication#setting-specific-filters-for-users). Pour les filtres des agents, voir [filtres d'agent](/fr/guide/agent#setting-up-filters).

> [!WARNING]
> Il est important de comprendre que plusieurs filtres se combinent pour restreindre les conteneurs. Par exemple, si vous définissez `--filter label=color` au niveau de l'interface et `--filter label=type` au niveau de l'agent, Dozzle n'affichera que les conteneurs qui portent à la fois les labels `color` et `type`.
