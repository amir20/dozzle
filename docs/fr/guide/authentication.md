---
title: Authentification
sourceHash: 4ce1bd2c3816
---

# Authentification

Dozzle prend en charge deux configurations pour l'authentification. Dans la première, vous apportez votre propre méthode d'authentification en protégeant Dozzle derrière un proxy. Dozzle sait lire les en-têtes appropriés sans configuration supplémentaire.

Si vous n'avez pas de solution d'authentification, Dozzle propose une gestion d'utilisateurs simple, basée sur un fichier. Les fournisseurs d'authentification se configurent avec le flag `--auth-provider`. Dans les deux configurations, Dozzle essaie d'enregistrer les paramètres utilisateur sur le disque. Ces données sont écrites dans `/data`.

## <Icon icon="mdi:shield-alert-outline" inline /> Considérations de sécurité

Dozzle a accès à `docker.sock`, ce qui équivaut, sauf restriction, à un accès **root sur l'hôte**. Avant d'exposer Dozzle en dehors de votre réseau privé, passez en revue les points suivants :

- **Placez toujours Dozzle derrière une authentification** s'il est joignable depuis Internet. Utilisez `--auth-provider=simple` ou un forward proxy comme Authelia / Authentik / Cloudflare Access.
- **Laissez les [actions](/fr/guide/actions) et l'[accès shell](/fr/guide/shell) désactivés** sauf si vous en avez besoin. Ils permettent de démarrer, arrêter, recréer des conteneurs et d'y exécuter des commandes arbitraires.
- **Restreignez les utilisateurs avec les [rôles](#setting-specific-roles-for-users) et les [filtres](#setting-specific-filters-for-users)** en mode multi-utilisateur. Sans rôles explicites, un utilisateur voit tous les conteneurs auxquels l'instance Dozzle a accès.
- **N'exposez jamais directement le port de Dozzle en mode forward proxy.** Dozzle fait confiance à `Remote-User` sur chaque requête, et quand aucun en-tête de rôles n'est présent, l'utilisateur reçoit tous les rôles. Quiconque peut atteindre le conteneur sans passer par le proxy s'authentifie comme il veut en positionnant un seul en-tête. Publiez uniquement le proxy et gardez Dozzle sur un réseau interne avec `expose` plutôt que `ports`.
- **Terminez le TLS au niveau du reverse proxy**. Voir [Reverse proxy et chemin de base](/fr/guide/changing-base) pour des exemples Nginx / Traefik / Caddy.
- **Restreignez l'accès à `docker.sock` avec un proxy** si vous n'avez pas besoin des actions. Notez qu'un montage en lecture seule (`/var/run/docker.sock:/var/run/docker.sock:ro`) ne limite _pas_ l'API : le flag `:ro` marque seulement le fichier socket en lecture seule sur le disque, alors que les appels API passent normalement par le socket, donc les opérations de création, suppression et mise à jour restent possibles. Pour réellement restreindre les opérations, placez un proxy de socket comme [`tecnativa/docker-socket-proxy`](https://github.com/Tecnativa/docker-socket-proxy) devant le démon.

## <Icon icon="mdi:account-cog-outline" inline /> Gestion des utilisateurs par fichier

Dozzle prend en charge l'authentification multi-utilisateur en réglant `--auth-provider` sur `simple`. Dans ce mode, Dozzle tente de lire le fichier des utilisateurs depuis `/data/`, en donnant la priorité à `users.yml` sur `users.yaml` si les deux fichiers sont présents. Si un seul existe, c'est celui-là qui est utilisé. Le log indique quel fichier est lu (par exemple `Reading users.yml file`).

### Exemples de chemins de fichiers :

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

> [!WARNING]
> Les hachages de mots de passe SHA-256 ne sont plus pris en charge. Les anciennes versions de Dozzle hachaient les mots de passe avec SHA-256, et un `users.yml` qui en contient encore un se charge sans erreur, mais le processus s'arrête dès que cet utilisateur tente de se connecter. Régénérez tous les mots de passe avec `generate` avant la mise à jour. Pour plus de détails, voir [cet avis de sécurité](https://github.com/amir20/dozzle/security/advisories/GHSA-w7qr-q9fh-fj35).

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

### Prolonger la durée de vie du cookie d'authentification

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

### Définir des filtres spécifiques pour les utilisateurs

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

### Définir des rôles spécifiques pour les utilisateurs

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

## <Icon icon="mdi:file-document-edit-outline" inline /> Générer users.yml

Dozzle intègre une commande `generate` pour produire `users.yml`. Voici un exemple :

```sh
docker run -it --rm amir20/dozzle generate admin --password password --email test@email.net --name "John Doe" --user-filter name=foo --user-roles shell > users.yml
```

Dans cet exemple, `admin` est le nom d'utilisateur. L'email et le nom sont optionnels mais recommandés pour afficher les bons avatars. `docker run -it --rm amir20/dozzle generate --help` affiche toutes les options. Le flag `--user-filter` prend une liste de filtres séparés par des virgules. Le flag `--user-roles` prend une liste de rôles séparés par des virgules.

Si vous omettez `--password`, Dozzle le demande sur stdin, pour que le mot de passe ne finisse pas dans l'historique de votre shell. Cela nécessite un terminal interactif, gardez donc les flags `-it` :

```sh
docker run -it --rm amir20/dozzle generate admin --email test@email.net --name "John Doe" > users.yml
```

L'invite est écrite sur stderr, la redirection de stdout vers `users.yml` fonctionne donc toujours. Vous pouvez aussi transmettre le mot de passe par un tube, par exemple `echo "$PASSWORD" | docker run -i --rm amir20/dozzle generate admin > users.yml`.

## <Icon icon="mdi:swap-horizontal" inline /> Forward proxy

Dozzle peut être configuré pour lire les en-têtes du proxy en réglant `--auth-provider` sur `forward-proxy`.

::: code-group

```sh [cli]
$ docker run -v /var/run/docker.sock:/var/run/docker.sock -v /path/to/dozzle/data:/data -p 8080:8080 amir20/dozzle --auth-provider forward-proxy
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
      DOZZLE_AUTH_PROVIDER: forward-proxy
```

:::

Montez `/data` ici aussi. Les paramètres par utilisateur sont également écrits sur le disque en mode forward proxy, et sans ce volume ils sont perdus à chaque recréation du conteneur.

Dans ce mode, Dozzle attend les en-têtes suivants :

- `Remote-User` correspond au nom d'utilisateur, par exemple `johndoe`
- `Remote-Email` correspond à l'adresse email de l'utilisateur. Cet email sert aussi à trouver le bon [Gravatar](https://gravatar.com/) pour l'utilisateur.
- `Remote-Name` est un nom d'affichage comme `John Doe`
- `Remote-Filter` est une liste de filtres autorisés pour l'utilisateur, séparés par des virgules.
- `Remote-Roles` est une liste de rôles autorisés pour l'utilisateur, séparés par des virgules.

Vous pouvez également configurer une URL de déconnexion avec :

```yaml
DOZZLE_AUTH_LOGOUT_URL: http://oauth2.example.ru/oauth2/sign_out
```

### Configurer Dozzle avec Authelia

[Authelia](https://www.authelia.com/) est un serveur et portail open source d'authentification et d'autorisation qui assure la gestion des identités et des accès. La configuration d'Authelia elle-même sort du cadre de cette section, mais sa configuration peut servir d'exemple pour mettre en place Dozzle avec Authelia.

<details>
<summary>➡️ Cliquez pour dérouler l'exemple Authelia</summary>

::: code-group

```yaml [docker-compose.yml]
networks:
  net:
    driver: bridge

services:
  authelia:
    image: authelia/authelia
    container_name: authelia
    volumes:
      - ./authelia:/config
    networks:
      - net
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.authelia.rule=Host(`authelia.example.com`)"
      - "traefik.http.routers.authelia.entrypoints=https"
      - "traefik.http.routers.authelia.tls=true"
      - "traefik.http.routers.authelia.tls.options=default"
      - "traefik.http.middlewares.authelia.forwardAuth.address=http://authelia:9091/api/authz/forward-auth"
      - "traefik.http.middlewares.authelia.forwardAuth.trustForwardHeader=true"
      - "traefik.http.middlewares.authelia.forwardAuth.authResponseHeaders=Remote-User,Remote-Groups,Remote-Name,Remote-Email"
    expose:
      - 9091
    restart: unless-stopped

  traefik:
    image: traefik:v3.5
    container_name: traefik
    volumes:
      - ./traefik:/etc/traefik
      - /var/run/docker.sock:/var/run/docker.sock
    networks:
      - net
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.api.rule=Host(`traefik.example.com`)"
      - "traefik.http.routers.api.entrypoints=https"
      - "traefik.http.routers.api.service=api@internal"
      - "traefik.http.routers.api.tls=true"
      - "traefik.http.routers.api.tls.options=default"
      - "traefik.http.routers.api.middlewares=authelia@docker"
    ports:
      - "80:80"
      - "443:443"
    command:
      - "--api"
      - "--providers.docker=true"
      - "--providers.docker.exposedByDefault=false"
      - "--providers.file.filename=/etc/traefik/certificates.yml"
      - "--entrypoints.http=true"
      - "--entrypoints.http.address=:80"
      - "--entrypoints.http.http.redirections.entrypoint.to=https"
      - "--entrypoints.http.http.redirections.entrypoint.scheme=https"
      - "--entrypoints.https=true"
      - "--entrypoints.https.address=:443"
      - "--log=true"
      - "--log.level=DEBUG"

  dozzle:
    image: amir20/dozzle:latest
    networks:
      - net
    environment:
      DOZZLE_AUTH_PROVIDER: forward-proxy
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - dozzle:/data
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.dozzle.rule=Host(`dozzle.example.com`)"
      - "traefik.http.routers.dozzle.entrypoints=https"
      - "traefik.http.routers.dozzle.tls=true"
      - "traefik.http.routers.dozzle.tls.options=default"
      - "traefik.http.routers.dozzle.middlewares=authelia@docker"
    expose:
      - 8080
    restart: unless-stopped

volumes:
  dozzle:
```

```yaml [configuration.yml]
###############################################################
#                   Authelia configuration                      #
###############################################################

server:
  address: tcp://0.0.0.0:9091

log:
  level: info

totp:
  issuer: authelia.com

identity_validation:
  reset_password:
    jwt_secret: a_very_important_secret

authentication_backend:
  file:
    path: /config/users_database.yml

access_control:
  default_policy: deny
  rules:
    - domain: traefik.example.com
      policy: one_factor
    - domain: dozzle.example.com
      policy: one_factor

session:
  secret: unsecure_session_secret
  cookies:
    - domain: example.com # Doit correspondre au domaine racine que vous protégez
      authelia_url: https://authelia.example.com
      default_redirection_url: https://public.example.com

regulation:
  max_retries: 3
  find_time: 120
  ban_time: 300

storage:
  encryption_key: you_must_generate_a_random_string_of_more_than_twenty_chars_and_configure_this
  local:
    path: /config/db.sqlite3

notifier:
  filesystem:
    filename: /config/notification.txt
```

:::

Des clés SSL valides sont nécessaires, car Authelia ne fonctionne qu'en SSL.

Authelia envoie l'appartenance aux groupes dans `Remote-Groups`, et Dozzle ne lit pas cet en-tête par défaut. Pour associer les groupes Authelia aux [rôles](#setting-specific-roles-for-users) Dozzle, définissez `DOZZLE_AUTH_HEADER_ROLES: Remote-Groups` sur le service Dozzle et nommez les groupes d'après les rôles. Les alias préfixés par `dozzle_` sont là pour ça : un groupe nommé `dozzle_shell` accorde le rôle `shell` et les autres noms de groupes sont ignorés. Sans cette correspondance, tout utilisateur authentifié obtient tous les rôles.

</details>

### Configurer Dozzle avec Cloudflare Zero Trust

Cloudflare Zero Trust est un service d'accès authentifié à des logiciels auto-hébergés. Cette section explique comment configurer Dozzle pour utiliser Cloudflare Zero Trust comme authentification.

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      DOZZLE_AUTH_PROVIDER: forward-proxy
      DOZZLE_AUTH_HEADER_USER: Cf-Access-Authenticated-User-Email
      DOZZLE_AUTH_HEADER_EMAIL: Cf-Access-Authenticated-User-Email
      DOZZLE_AUTH_HEADER_NAME: Cf-Access-Authenticated-User-Email
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - dozzle:/data
    expose:
      - 8080
    restart: unless-stopped

volumes:
  dozzle:
```

`expose` garde le port 8080 hors de l'hôte, le seul chemin d'entrée est donc le tunnel. Le publier avec `ports` permettrait à n'importe qui sur l'hôte de définir lui-même `Cf-Access-Authenticated-User-Email` et de contourner complètement Cloudflare.

Après avoir lancé le conteneur Dozzle, configurez l'application dans le tableau de bord Cloudflare Zero Trust en suivant le [guide](https://developers.cloudflare.com/cloudflare-one/applications/configure-apps/self-hosted-apps/).

### Configurer Dozzle avec Pocket ID

Vous devez d'abord mettre en place un conteneur qui fait transiter l'authentification OpenID Connect par votre reverse proxy.

Voici un exemple utilisant [oauth2-proxy](https://github.com/oauth2-proxy/oauth2-proxy).

<details>
<summary>➡️ Cliquez pour dérouler l'exemple oauth2-proxy</summary>

1. Créez un nouveau client OIDC dans Pocket ID pour Dozzle :
   - **Nom :** `Dozzle`
   - **URL de callback :** `https://dozzle.example.com/oauth2/callback`
   - **PKCE :** `Enabled`

   Copiez les valeurs **Client ID** et **Client Secret** pour plus tard.

2. Ajoutez ceci à votre fichier compose Dozzle existant :

   ```yml
   environment:
     DOZZLE_AUTH_PROVIDER: forward-proxy
     DOZZLE_AUTH_HEADER_USER: X-Forwarded-User
     DOZZLE_AUTH_HEADER_EMAIL: X-Forwarded-Email
     DOZZLE_AUTH_HEADER_NAME: X-Forwarded-Preferred-Username
   ```

   Commentez les ports de Dozzle, puisqu'ils vont être redirigés à travers le nouveau conteneur d'authentification.

   Cette méthode ne devrait demander aucune modification de la configuration de votre reverse proxy.

   ```yml
   # ports:
   #   - 8080:8080
   ```

3. Ajoutez un nouveau service oauth2-proxy à votre fichier compose Dozzle existant :

   ```yml
   services:
     # ...
     oauth2-proxy:
       image: quay.io/oauth2-proxy/oauth2-proxy:latest
       restart: unless-stopped
       container_name: dozzle-oidc
       command: --config /oauth2-proxy.cfg
       volumes:
         - "./oauth2-proxy.cfg:/oauth2-proxy.cfg"
       ports:
         - 8080:4180
   ```

4. Créez le fichier de configuration d'oauth2-proxy.

   Dans le dossier de votre fichier compose, créez `oauth2-proxy.cfg` :

   ```toml
    client_id = "xxx"                            # depuis Pocket ID
    client_secret = "xxx"                        # depuis Pocket ID
    cookie_secret = "xxx"                        # générez avec openssl rand -base64 32 | tr -- '+/' '-_'
    upstreams = "http://dozzle:8080"             # upstream vers le port interne du conteneur Dozzle
    code_challenge_method = "S256"               # défis PKCE plain ou S256
    cookie_expire = "0"                          # secondes, 0 pour la session
    cookie_name = "__Host-oauth2-proxy"          # ou __Secure-oauth2-proxy (moins sûr)
    cookie_secure = true                         # utilise le cookie sécurisé HTTPS
    email_domains = ["*"]                        # autorise n'importe quel domaine email à s'authentifier
    http_address = "0.0.0.0:4180"                # port sur lequel oauth2-proxy écoute
    oidc_issuer_url = "https://id.example.com"   # l'URL de base de votre Pocket
    provider_display_name = "Pocket ID"          # nom affiché pour la connexion OIDC
    provider = "oidc"                            # utilise OpenID Connect
    reverse_proxy = true                         # met le trafic derrière un reverse proxy
    scope = "openid email profile groups"        # transmet ces portées OIDC
   ```

   Remplissez les variables en suivant les commentaires.

5. Enfin, redémarrez votre stack Docker Compose.

   Votre reverse proxy devrait maintenant vous authentifier auprès de Dozzle via oauth2-proxy.

   En cas de problème, consultez les logs.

</details>
