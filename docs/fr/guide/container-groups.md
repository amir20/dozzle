---
title: Groupes de conteneurs
sourceHash: 83c35e4fb25c
---

# Groupes de conteneurs

Dozzle regroupe automatiquement les conteneurs selon leur nom de stack ou de service. Vous pouvez aussi créer vos propres groupes à l'aide de labels.

## Groupes par défaut

En mode hôte, les conteneurs sont regroupés par défaut selon leur nom de stack. Si le label `com.docker.swarm.service.name` est présent, Dozzle active automatiquement un « mode Swarm » où tous les conteneurs partageant le même nom de service sont réunis.

## Groupes personnalisés

Vous pouvez également créer des groupes personnalisés en ajoutant un label à votre conteneur. Le label est `dev.dozzle.group` et sa valeur est le nom du groupe. Tous les conteneurs portant le même nom de groupe sont réunis dans l'interface. Par exemple, avec un groupe nommé `myapp`, tous les conteneurs ayant le label `dev.dozzle.group=myapp` seront regroupés.

Voici un exemple avec Docker Compose ou la CLI Docker :

::: code-group

```sh
docker run --label dev.dozzle.group=myapp hello-world
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: hello-world
    labels:
      - dev.dozzle.group=myapp
```

:::

## Kubernetes

En mode Kubernetes, la barre latérale regroupe les pods par namespace et par propriétaire. Un label ou une annotation `dev.dozzle.group` sur le modèle de pod ajoute en plus tous les conteneurs du pod à un groupe personnalisé. Un label et une annotation se comportent de la même façon, donc utilisez une annotation si le nom du groupe contient des espaces. Si les deux sont définis, l'annotation l'emporte.

```yaml [deployment.yaml]
spec:
  template:
    metadata:
      labels:
        dev.dozzle.group: myapp
```
