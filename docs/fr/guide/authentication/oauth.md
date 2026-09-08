---
title: Se connecter avec GitHub et OIDC
sourceHash: cb9474f23fcc
---

# <Icon icon="mdi:shield-account" inline /> Se connecter avec GitHub et OIDC

Dozzle peut laisser les utilisateurs se connecter avec un compte externe au lieu de saisir un mot de passe. Cela fait partie du fournisseur [`simple`](/fr/guide/authentication/simple) et ce n'est pas un fournisseur à part entière : `users.yml` est toujours lu à chaque requête et c'est toujours lui qui décide qui entre.

Cela a une conséquence qui mérite d'être dite d'emblée : **`users.yml` est la liste d'autorisation.** Un compte externe auquel aucune entrée n'est liée ne peut pas se connecter, et aucun compte n'est jamais créé automatiquement.

La connexion par mot de passe continue de fonctionner à côté, ce qui compte quand une OAuth App casse et qu'il faut pouvoir entrer pour la réparer.

## Se connecter avec GitHub

Dozzle peut laisser les utilisateurs se connecter avec leur compte GitHub au lieu de saisir un mot de passe. Cela fait partie du fournisseur `simple` et ce n'est pas un fournisseur d'authentification distinct : `users.yml` est toujours lu à chaque requête et c'est toujours lui qui décide qui entre. Continuez d'utiliser `--auth-provider simple`. `github` est accepté comme alias si vous préférez expliciter ce que l'instance utilise.

Créez d'abord une OAuth App dans les [Developer settings](https://github.com/settings/developers) de GitHub et réglez l'**Authorization callback URL** sur :

```
https://your-dozzle-host/api/auth/callback
```

Si Dozzle est servi sous un [chemin de base](/fr/guide/changing-base), incluez-le, par exemple `https://example.com/dozzle/api/auth/callback`. Dozzle n'envoie pas de `redirect_uri` au démarrage du flux, GitHub redirige donc toujours vers l'URL de callback enregistrée sur l'OAuth App. Une différence à cet endroit est la cause la plus fréquente d'échec de connexion.

Copiez ensuite le client ID, générez un client secret, et passez les deux à Dozzle :

::: code-group

```sh [cli]
$ docker run -v /var/run/docker.sock:/var/run/docker.sock -v /path/to/dozzle/data:/data -p 8080:8080 amir20/dozzle --auth-provider simple --auth-github-client-id Iv1.0123456789abcdef --auth-github-client-secret 0123456789abcdef0123456789abcdef01234567
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
      DOZZLE_AUTH_GITHUB_CLIENT_ID: Iv1.0123456789abcdef
      DOZZLE_AUTH_GITHUB_CLIENT_SECRET: 0123456789abcdef0123456789abcdef01234567
```

:::

Liez un utilisateur à son compte GitHub avec une clé `github` dans `users.yml` :

```yaml
users:
  admin:
    email: me@email.net
    name: Admin
    password: $2a$11$9ho4vY2LdJ/WBopFcsAS0uORC0x2vuFHQgT/yBqZyzclhHsoaIkzK
    github: octocat

  guest:
    email: guest@email.net
    name: Guest
    github: hubot
    filter: "label=com.example.app"
    roles: none
```

`password` devient facultatif dès que `github` est défini, comme le montre `guest` ci-dessus. `admin` a les deux, il peut donc se connecter des deux façons. La connexion par mot de passe reste disponible en solution de repli pour tous ceux qui en ont encore un, et la page de connexion propose les deux options.

La valeur est le **login** GitHub (l'identifiant dans `github.com/octocat`), pas l'adresse email. Les logins sont stables et toujours visibles, alors que l'email d'un compte peut être privé ou changer à tout moment.

`users.yml` est la liste d'autorisation. Un compte GitHub qui n'y figure pas ne peut pas se connecter, quelle que soit l'organisation à laquelle il appartient. Il n'y a pas de création automatique de compte : ajouter quelqu'un veut dire l'ajouter au fichier. Les filtres et les rôles sont résolus depuis `users.yml` à chaque requête, exactement comme pour les utilisateurs avec mot de passe, donc un utilisateur GitHub avec `roles: none` est restreint de la même manière.

> [!NOTE]
> Dozzle ne permet volontairement pas d'autoriser toute une organisation GitHub ou tout un domaine email. Chaque utilisateur est listé individuellement. Si vous avez besoin d'un accès par groupe ou par domaine, utilisez `forward-proxy` avec [Authelia](/fr/guide/authentication/forward-proxy#configurer-dozzle-avec-authelia) ou Authentik, qui sont faits pour ça.

> [!WARNING]
> Modifier `users.yml` fait tourner la clé de signature JWT et déconnecte tous les utilisateurs. C'est déjà le cas aujourd'hui quand vous ajoutez ou supprimez un utilisateur, et cela vaut aussi quand vous ajoutez une clé `github`.

## Se connecter avec OIDC

N'importe quel fournisseur qui publie un document de découverte OpenID Connect fonctionne avec le même callback : Google, Keycloak, Pocket ID, Zitadel, Authentik et d'autres. Pointez Dozzle sur l'URL de l'émetteur et donnez-lui un client id et un client secret.

Enregistrez Dozzle comme client confidentiel auprès de votre fournisseur et réglez l'URI de redirection sur :

```
https://your-dozzle-host/api/auth/callback
```

Incluez le chemin de base si Dozzle tourne sous un chemin de base, par exemple `https://example.com/dozzle/api/auth/callback`. Contrairement à GitHub, OIDC oblige Dozzle à envoyer `redirect_uri`, cette valeur doit donc correspondre exactement à ce que vous avez enregistré.

::: code-group

```sh [cli]
$ docker run -v /var/run/docker.sock:/var/run/docker.sock -v /path/to/dozzle/data:/data -p 8080:8080 amir20/dozzle --auth-provider simple --auth-oidc-issuer https://id.example.com --auth-oidc-client-id dozzle --auth-oidc-client-secret secret --auth-oidc-name "Pocket ID"
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
      DOZZLE_AUTH_OIDC_ISSUER: https://id.example.com
      DOZZLE_AUTH_OIDC_CLIENT_ID: dozzle
      DOZZLE_AUTH_OIDC_CLIENT_SECRET: secret
      DOZZLE_AUTH_OIDC_NAME: Pocket ID
```

:::

`DOZZLE_AUTH_OIDC_NAME` n'est que le libellé du bouton de connexion. Sa valeur par défaut est `SSO`.

L'URL de l'émetteur est celle qui sert `/.well-known/openid-configuration`. Dozzle récupère ce document pour trouver les endpoints d'autorisation, de token et de userinfo, et refuse de démarrer le flux si le document déclare un émetteur différent de celui que vous avez configuré.

### Lier les utilisateurs

OIDC fait la correspondance sur l'**email vérifié**, pas sur le login. Définissez `email` sur l'utilisateur dans `users.yml` :

```yaml
users:
  admin:
    email: me@email.net
    name: Admin
    # le mot de passe est facultatif une fois le compte lié
```

L'email doit être marqué comme vérifié par votre fournisseur. Dozzle refuse la connexion quand `email_verified` est faux, parce que faire correspondre une adresse non vérifiée permettrait à quiconque peut s'inscrire chez un fournisseur permissif de s'approprier un compte en saisissant l'email de quelqu'un d'autre.

> [!NOTE]
> GitHub fait la correspondance sur le login et OIDC sur l'email, et cette différence est voulue. Un login GitHub est stable et toujours présent, alors qu'un email GitHub peut être privé ou changer. OIDC n'a pas d'équivalent stable et lisible par un humain, l'email vérifié est donc l'attribut que les administrateurs connaissent réellement.

### Google

Google est un fournisseur OIDC comme un autre. Créez un client OAuth dans la console Google Cloud et utilisez :

```
DOZZLE_AUTH_OIDC_ISSUER: https://accounts.google.com
DOZZLE_AUTH_OIDC_NAME: Google
```

`--auth-provider google` est accepté comme alias de `simple`, les deux écritures fonctionnent donc.

## Utiliser les secrets Docker pour le client secret

Mettre un client secret directement dans `environment:` veut dire qu'il apparaît dans `docker inspect`, dans votre fichier compose, et dans l'historique du shell de quiconque a lancé le conteneur à la main. Les deux client secrets acceptent donc un équivalent `_FILE` qui nomme un fichier depuis lequel lire la valeur, la convention qu'utilisent les images officielles de Docker :

| Au lieu de                         | Utilisez                                |
| ---------------------------------- | --------------------------------------- |
| `DOZZLE_AUTH_GITHUB_CLIENT_SECRET` | `DOZZLE_AUTH_GITHUB_CLIENT_SECRET_FILE` |
| `DOZZLE_AUTH_OIDC_CLIENT_SECRET`   | `DOZZLE_AUTH_OIDC_CLIENT_SECRET_FILE`   |

Dozzle lit le fichier au démarrage et supprime les espaces autour, un retour à la ligne final laissé par `echo secret > file` ne pose donc pas de problème. Définir à la fois une variable et son équivalent `_FILE` est une erreur plutôt qu'une préférence silencieuse pour l'une des deux, et pointer `_FILE` sur un fichier manquant ou vide arrête Dozzle au démarrage au lieu de désactiver discrètement le bouton de connexion.

### Docker Compose

Hors Swarm, `docker secret create` n'existe pas : un secret Compose est donc soit un fichier sur le disque, soit une variable d'environnement. La forme environnement est en général celle qu'il vous faut, elle va de pair avec un `.env` ignoré par git et aucun fichier en clair ne traîne à côté de votre fichier compose en attendant d'être commité.

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    ports:
      - 8080:8080
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./data:/data
    environment:
      DOZZLE_AUTH_PROVIDER: simple
      DOZZLE_AUTH_GITHUB_CLIENT_ID: Iv1.0123456789abcdef
      DOZZLE_AUTH_GITHUB_CLIENT_SECRET_FILE: /run/secrets/dozzle_github_secret
    secrets:
      - dozzle_github_secret

secrets:
  dozzle_github_secret:
    environment: GITHUB_CLIENT_SECRET
```

```ini [.env]
GITHUB_CLIENT_SECRET=your-github-client-secret
```

Compose lit lui-même la variable et monte la valeur sur `/run/secrets/dozzle_github_secret`. Elle ne fait jamais partie de l'environnement du conteneur, elle reste donc hors de `docker inspect`, exactement comme un secret adossé à un fichier.

Utilisez plutôt la forme fichier quand le secret existe déjà sous forme de fichier, par exemple écrit par un gestionnaire de secrets :

```yaml [docker-compose.yml]
secrets:
  dozzle_github_secret:
    file: /run/secrets/github_client_secret
```

Dans les deux cas, Dozzle supprime les espaces autour de la valeur, un retour à la ligne final dans le fichier n'a donc aucune importance.

### Docker Swarm

En Swarm le secret est géré par le cluster plutôt que par un fichier sur le disque, déclarez-le donc comme `external` et créez-le avec `docker secret create` :

```sh
printf '%s' 'your-oidc-client-secret' | docker secret create dozzle_oidc_secret -
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      DOZZLE_MODE: swarm
      DOZZLE_AUTH_PROVIDER: simple
      DOZZLE_AUTH_OIDC_ISSUER: https://id.example.com
      DOZZLE_AUTH_OIDC_CLIENT_ID: dozzle
      DOZZLE_AUTH_OIDC_CLIENT_SECRET_FILE: /run/secrets/dozzle_oidc_secret
      DOZZLE_AUTH_OIDC_NAME: Pocket ID
    secrets:
      - dozzle_oidc_secret
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    deploy:
      mode: global

secrets:
  dozzle_oidc_secret:
    external: true
```

> [!NOTE]
> Un secret est monté par défaut sur `/run/secrets/<name>`, c'est pour cela que `_FILE` pointe là. Si vous définissez un `target:` explicite, faites pointer `_FILE` sur ce chemin.

### Vérifier que ça a marché

Dozzle journalise au démarrage les fournisseurs qu'il a activés. Lancez-le avec `--level debug` et cherchez la ligne qui nomme le fournisseur :

```sh
$ docker compose logs dozzle | grep -i 'sign in'
DBG Enabling Sign in with GitHub
```

Si le fichier du secret est manquant ou vide, Dozzle s'arrête au démarrage avec un message qui nomme la variable, un montage cassé échoue donc bruyamment au lieu de faire disparaître silencieusement le bouton de connexion.
