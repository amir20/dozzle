---
title: Accès shell aux conteneurs
description: Comment activer l'accès shell dans Dozzle pour s'attacher aux conteneurs Docker et Kubernetes ou y exécuter des commandes depuis le navigateur, et comment le sécuriser.
sourceHash: 6617c04a0ef1
---

# Attacher un terminal et exécuter des commandes

<Badge type="tip" text="Docker" />
<Badge type="tip" text="K8s" />

Dozzle permet de s'attacher à un conteneur ou d'y exécuter des commandes. Il fournit une interface web pour interagir avec les conteneurs Docker, ce qui permet de s'attacher aux conteneurs en cours d'exécution et d'exécuter des commandes directement depuis le navigateur. C'est particulièrement utile pour déboguer et diagnostiquer des applications conteneurisées. Cette fonctionnalité est **désactivée** par défaut car elle peut présenter des risques de sécurité. Pour l'activer, définissez la variable d'environnement `DOZZLE_ENABLE_SHELL` à `true`.

::: code-group

```sh
docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --enable-shell
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
      DOZZLE_ENABLE_SHELL: true
```

:::

> [!NOTE]
> L'accès shell fonctionne avec tous les types de conteneurs, y compris Docker, Kubernetes et les autres plateformes d'orchestration.

## <Icon icon="mdi:shield-lock-outline" inline /> Sécurité

Toute personne qui peut atteindre l'interface de Dozzle pourra ouvrir un shell dans vos conteneurs, l'équivalent de `docker exec`. Avant d'activer `--enable-shell` sur un Dozzle accessible publiquement, placez-le derrière une [authentification](/fr/guide/authentication). Les permissions par rôle permettent de restreindre l'accès shell à certains utilisateurs.

Sans authentification, le shell ne fonctionne que si vous ouvrez Dozzle par adresse IP ou par `localhost` (par exemple `http://192.168.1.10:8080`). Ouvert par un nom d'hôte comme `dozzle.lan` ou via un reverse proxy, il est refusé, car une page web malveillante peut utiliser le DNS rebinding pour atteindre Dozzle sous son propre nom d'hôte. Activez l'[authentification](/fr/guide/authentication) pour utiliser le shell depuis un nom d'hôte.

## <Icon icon="mdi:kubernetes" inline /> Kubernetes

En mode k8s, l'accès shell passe par l'API Kubernetes plutôt que par `docker exec`. Le pod ciblé doit contenir un shell exécutable (`/bin/sh`, `/bin/bash`, etc.). Les images minimales construites `FROM scratch` ou les images distroless sans shell ne peuvent pas être attachées.
