---
title: Noms de conteneurs
sourceHash: 31d0ec398f6d
---

# Noms de conteneurs

Par défaut, Dozzle récupère les noms des conteneurs directement depuis Docker. C'est généralement suffisant, car ces noms peuvent être personnalisés avec l'option `--name` de `docker run` ou via le champ `container_name` des services Docker Compose.

## Noms personnalisés

Lorsqu'il n'est pas possible de modifier le nom du conteneur lui-même, vous pouvez le remplacer en ajoutant le label `dev.dozzle.name` à votre conteneur.

Voici un exemple avec Docker Compose ou la CLI Docker :

::: code-group

```sh
docker run --label dev.dozzle.name=hello hello-world
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: hello-world
    labels:
      - dev.dozzle.name=hello
```

:::

## Intégration Coolify

Si vous utilisez [Coolify](https://coolify.io/), Dozzle reconnaît automatiquement ses labels comme valeurs de repli :

- `coolify.serviceName` → utilisé comme nom de conteneur si `dev.dozzle.name` n'est pas défini
- `coolify.projectName` → utilisé pour le regroupement si `dev.dozzle.group` n'est pas défini

Aucune configuration supplémentaire n'est nécessaire pour les déploiements Coolify.
