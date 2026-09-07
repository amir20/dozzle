---
title: Authentification simple
sourceHash: deea96688436
---

# <Icon icon="mdi:account-cog-outline" inline /> Authentification simple

La gestion d'utilisateurs propre à Dozzle. Les utilisateurs vivent dans un fichier `users.yml` que Dozzle gère lui-même, et Dozzle sert sa propre page de connexion. Réglez `--auth-provider` sur `simple` pour l'activer.

Les mots de passe sont une façon de prouver que vous êtes l'un de ces utilisateurs. [Se connecter avec GitHub ou OIDC](/fr/guide/authentication/oauth) en est une autre, et les deux lisent le même `users.yml`.

> [!TIP]
> Utilisez la [commande `generate`](/fr/guide/authentication#generer-users-yml) intégrée pour créer `users.yml` plutôt que d'écrire des hachages bcrypt à la main.

Dozzle prend en charge l'authentification multi-utilisateur en réglant `--auth-provider` sur `simple`. Dans ce mode, Dozzle tente de lire le fichier des utilisateurs depuis `/data/`, en donnant la priorité à `users.yml` sur `users.yaml` si les deux fichiers sont présents. Si un seul existe, c'est celui-là qui est utilisé. Le log indique quel fichier est lu (par exemple `Reading users.yml file`).

## Exemples de chemins de fichiers :

- `/data/users.yml`
- `/data/users.yaml`

Le contenu du fichier ressemble à ceci :

```yaml
users:
  # "admin" est ici le nom d'utilisateur
  admin:
    email: me@email.net
    name: Admin
    # Générez avec docker run -it --rm amir20/dozzle generate admin --password password --email me@email.net --name "Admin"
    password: $2a$11$9ho4vY2LdJ/WBopFcsAS0uORC0x2vuFHQgT/yBqZyzclhHsoaIkzK
    filter:
    roles:
```

Dozzle utilise `email` pour générer les avatars via [Gravatar](https://gravatar.com/). C'est optionnel. Le mot de passe est haché avec `bcrypt`, ce qui peut être fait avec `docker run amir20/dozzle generate`.

Vous devrez monter ce fichier pour que Dozzle le trouve. Voici un exemple :

::: code-group

```sh [cli]
$ docker run -v /var/run/docker.sock:/var/run/docker.sock -v /path/to/dozzle/data:/data -p 8080:8080 amir20/dozzle --auth-provider simple
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /path/to/dozzle/data:/data
    ports:
      - 8080:8080
    environment:
      DOZZLE_AUTH_PROVIDER: simple
```

```yaml [users.yml]
users:
  admin:
    email: me@email.net
    name: Admin
    password: $2a$11$9ho4vY2LdJ/WBopFcsAS0uORC0x2vuFHQgT/yBqZyzclhHsoaIkzK
```

:::

Ou en utilisant les secrets Docker :

```yaml
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      - DOZZLE_AUTH_PROVIDER=simple
    secrets:
      - source: users
        target: /data/users.yml
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - dozzle:/data
secrets:
  users:
    file: users.yml
volumes:
  dozzle:
```

## Prolonger la durée de vie du cookie d'authentification

Par défaut, Dozzle utilise des cookies de session qui expirent à la fermeture du navigateur. Vous pouvez prolonger la durée de vie du cookie en réglant `--auth-ttl` sur une durée. Voici un exemple :

::: code-group

```sh [cli]
$ docker run -v /var/run/docker.sock:/var/run/docker.sock -v /path/to/dozzle/data:/data -p 8080:8080 amir20/dozzle --auth-provider simple --auth-ttl 48h
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /path/to/dozzle/data:/data
    ports:
      - 8080:8080
    environment:
      DOZZLE_AUTH_PROVIDER: simple
      DOZZLE_AUTH_TTL: 48h
```

:::

Notez que seule une durée est acceptée. Vous ne pouvez utiliser que `s`, `m` et `h` pour les secondes, minutes et heures.

## Définir des filtres spécifiques pour les utilisateurs

Dozzle permet de définir des filtres par utilisateur. Les filtres servent à restreindre les conteneurs qu'un utilisateur peut voir. Ils se définissent dans le fichier `users.yml`. Voici un exemple :

```yaml
users:
  admin:
    email:
    name: Admin
    password: $2a$11$9ho4vY2LdJ/WBopFcsAS0uORC0x2vuFHQgT/yBqZyzclhHsoaIkzK
    filter:

  guest:
    email:
    name: Guest
    password: $2a$11$9ho4vY2LdJ/WBopFcsAS0uORC0x2vuFHQgT/yBqZyzclhHsoaIkzK
    filter: "label=com.example.app"
```

Dans cet exemple, l'utilisateur `admin` n'a aucun filtre, il voit donc tous les conteneurs. L'utilisateur `guest` ne voit que les conteneurs portant le label `com.example.app`. C'est pratique pour restreindre l'accès à certains conteneurs.

> [!NOTE]
> Les filtres peuvent aussi être définis [globalement](/fr/guide/filters) avec le flag `--filter`. Ce flag s'applique à tous les utilisateurs. Si un utilisateur a un filtre défini, celui-ci remplace le filtre global.

## Définir des rôles spécifiques pour les utilisateurs

Dozzle permet d'attribuer des rôles aux utilisateurs. Les rôles définissent les actions qu'un utilisateur peut effectuer sur les conteneurs. Ils se configurent dans le fichier users.yml.

```yaml
users:
  admin:
    email:
    name: Admin
    password: $2a$11$9ho4vY2LdJ/WBopFcsAS0uORC0x2vuFHQgT/yBqZyzclhHsoaIkzK
    roles:

  guest:
    email:
    name: Guest
    password: $2a$11$9ho4vY2LdJ/WBopFcsAS0uORC0x2vuFHQgT/yBqZyzclhHsoaIkzK
    roles: shell
```

Dans cet exemple, l'utilisateur `admin` n'a aucun rôle indiqué, il a donc un accès complet à toutes les actions sur les conteneurs. L'utilisateur `guest` a le rôle shell, ce qui veut dire qu'il peut seulement ouvrir un shell dans les conteneurs. Les rôles facilitent le contrôle et la restriction de ce que les utilisateurs peuvent faire dans Dozzle.

Dozzle prend en charge les rôles suivants :

| Rôle            | Aussi accepté          | Accorde                                                                                               |
| --------------- | ---------------------- | ----------------------------------------------------------------------------------------------------- |
| `shell`         | `dozzle_shell`         | S'attacher à un conteneur et ouvrir une session exec. L'instance a aussi besoin de `--enable-shell`.  |
| `actions`       | `dozzle_actions`       | Démarrer, arrêter et redémarrer les conteneurs. L'instance a aussi besoin de `--enable-actions`.      |
| `download`      | `dozzle_download`      | Télécharger les logs d'un conteneur sous forme de fichier.                                            |
| `notifications` | `dozzle_notifications` | Créer et modifier les règles de notification et les destinations.                                     |
| `cloud`         | `dozzle_cloud`         | Lier, délier et configurer Dozzle Cloud.                                                              |
| `all`           | `dozzle_all`           | Tous les rôles ci-dessus. C'est la valeur par défaut quand `roles` est vide.                          |
| `none`          | `dozzle_none`          | Aucun rôle. Les logs restent consultables, selon le filtre de l'utilisateur. Prime sur tout le reste. |

Les rôles se séparent par des virgules ou des barres verticales (`shell,actions` ou `shell|actions`), et un tableau JSON fonctionne aussi (`["shell", "actions"]`). Les noms sont insensibles à la casse. Les alias préfixés par `dozzle_` existent pour que les noms de groupes d'un fournisseur d'identité puissent être transmis tels quels en mode forward proxy.

> [!WARNING]
> Les règles de notification s'appliquent à toute l'instance. Une règle sélectionne les conteneurs par expression, pas par le filtre de l'utilisateur, donc un utilisateur ayant le rôle `notifications` peut créer une règle pour des conteneurs que son filtre masque par ailleurs et recevoir ces lignes de log sur une destination qu'il contrôle. Ne l'accordez qu'aux utilisateurs à qui vous confiez tous les conteneurs de l'instance.

> [!WARNING]
> Dozzle Cloud s'applique aussi à toute l'instance. La liaison enregistre une seule clé d'API qui redirige l'envoi des alertes, le streaming des logs et l'exécution des outils vers un seul compte cloud, et les outils cloud s'exécutent avec le filtre de l'instance plutôt qu'avec celui de l'utilisateur qui a fait la liaison. Un utilisateur ayant le rôle `cloud` peut lier l'instance à son propre compte cloud et voir tous les conteneurs à travers lui, ou supprimer une connexion existante. Ne l'accordez qu'aux utilisateurs à qui vous confiez tous les conteneurs de l'instance.

Tout rôle peut être préfixé par `^` pour être exclu. Les exclusions sont appliquées en dernier, l'ordre n'a donc pas d'importance :

```yaml
roles: all,^shell # tout sauf shell
```

`none` est le seul rôle qui ne peut pas être nié. `^none` est ignoré, et un simple `none` n'importe où dans la liste supprime tous les autres rôles.
