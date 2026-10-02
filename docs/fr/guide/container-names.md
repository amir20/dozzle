---
title: Noms de conteneurs
sourceHash: cafa1526d194
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

## Kubernetes

En mode Kubernetes, les conteneurs sont nommés `<pod>/<container>` par défaut. Définissez `dev.dozzle.name` sur le modèle de pod pour le remplacer. Une annotation est plus adaptée, car les valeurs de label ne peuvent pas contenir d'espaces et sont limitées à 63 caractères. Si les deux sont définis, l'annotation l'emporte. Il en va de même pour tous les autres paramètres `dev.dozzle.*`, comme `dev.dozzle.url` et `dev.dozzle.icon`.

```yaml [deployment.yaml]
spec:
  template:
    metadata:
      annotations:
        dev.dozzle.name: Public API
```

Le nom s'applique à tout le pod. Les conteneurs d'initialisation gardent toujours leur propre nom en suffixe, par exemple `Public API/migrate`, tout comme chaque conteneur d'un pod qui contient plusieurs conteneurs applicatifs, par exemple `Public API/proxy`.

Toutes les réplicas d'un même déploiement reçoivent le même nom, et Dozzle traite les conteneurs portant le même nom comme un seul. Épingler une réplica dans la barre latérale les épingle toutes, et le menu du titre du conteneur liste les autres comme des exécutions précédentes du même conteneur.

## Intégration Coolify

Si vous utilisez [Coolify](https://coolify.io/), Dozzle reconnaît automatiquement ses labels comme valeurs de repli :

- `coolify.serviceName` → utilisé comme nom de conteneur si `dev.dozzle.name` n'est pas défini
- `coolify.projectName` → utilisé pour le regroupement si `dev.dozzle.group` n'est pas défini

Aucune configuration supplémentaire n'est nécessaire pour les déploiements Coolify.
