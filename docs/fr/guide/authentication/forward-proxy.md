---
title: Forward proxy
sourceHash: 37a5c3119fb2
---

# <Icon icon="mdi:swap-horizontal" inline /> Forward proxy

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

## Configurer Dozzle avec Authelia

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

Authelia envoie l'appartenance aux groupes dans `Remote-Groups`, et Dozzle ne lit pas cet en-tête par défaut. Pour associer les groupes Authelia aux [rôles](/fr/guide/authentication/simple#definir-des-roles-specifiques-pour-les-utilisateurs) Dozzle, définissez `DOZZLE_AUTH_HEADER_ROLES: Remote-Groups` sur le service Dozzle et nommez les groupes d'après les rôles. Les alias préfixés par `dozzle_` sont là pour ça : un groupe nommé `dozzle_shell` accorde le rôle `shell` et les autres noms de groupes sont ignorés. Sans cette correspondance, tout utilisateur authentifié obtient tous les rôles.

</details>

## Configurer Dozzle avec Cloudflare Zero Trust

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

## Configurer Dozzle avec Pocket ID

> [!TIP]
> Dozzle parle maintenant OpenID Connect nativement, vous pouvez donc pointer `--auth-oidc-issuer` directement sur Pocket ID et vous passer complètement d'oauth2-proxy. Voir [Se connecter avec GitHub et OIDC](/fr/guide/authentication/oauth#se-connecter-avec-oidc). La recette ci-dessous reste là pour les installations qui font déjà tourner oauth2-proxy ou qui veulent que le proxy protège autre chose que Dozzle.

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
