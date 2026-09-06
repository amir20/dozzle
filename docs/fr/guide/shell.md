---
title: Accès shell aux conteneurs
sourceHash: 267b2a6665a0
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

## <Icon icon="mdi:kubernetes" inline /> Kubernetes

En mode k8s, l'accès shell passe par l'API Kubernetes plutôt que par `docker exec`. Le pod ciblé doit contenir un shell exécutable (`/bin/sh`, `/bin/bash`, etc.). Les images minimales construites `FROM scratch` ou les images distroless sans shell ne peuvent pas être attachées.
