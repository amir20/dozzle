---
title: Variables d'environnement et sous-commandes
sourceHash: 94da032e7db0
---

# Variables d'environnement

Chaque option peut être définie par une option en ligne de commande ou par une variable d'environnement. Les options et les variables d'environnement l'emportent toujours sur les réglages enregistrés dans `dozzle.yml` par l'[assistant de configuration](/fr/guide/setup-wizard).

Les options qui prennent une liste (`DOZZLE_FILTER`, `DOZZLE_REMOTE_HOST`, `DOZZLE_REMOTE_AGENT`, `DOZZLE_NAMESPACE`) acceptent une valeur séparée par des virgules dans la variable d'environnement, ou l'option répétée une fois par élément :

```sh
--remote-agent 167.99.1.1:7007 --remote-agent 167.99.1.2:7007
DOZZLE_REMOTE_AGENT=167.99.1.1:7007,167.99.1.2:7007
```

## Serveur

| Variable                                  | Description                                                                                                      | Valeurs                                                           | Défaut   |
| ----------------------------------------- | ---------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------- | -------- |
| `DOZZLE_ADDR`<br>`--addr`                 | Adresse et port sur lesquels le serveur web écoute. Rarement utile dans un conteneur.                            | `host:port`, par ex. `:9090`                                      | `:8080`  |
| `DOZZLE_BASE`<br>`--base`                 | Préfixe de chemin sous lequel servir Dozzle. Voir [changer la base](/fr/guide/changing-base).                    | un chemin, par ex. `/logs`                                        | `/`      |
| `DOZZLE_HOSTNAME`<br>`--hostname`         | Nom affiché dans l'interface pour cette instance. Voir [hostname](/fr/guide/hostname).                           | n'importe quelle chaîne                                           | aucun    |
| `DOZZLE_HOST_ID`<br>`--host-id`           | Remplace l'identifiant que Dozzle calcule pour cet hôte. Utile seulement en cas de collision avec un autre hôte. | lettres, chiffres, `_`, `.`, `-`                                  | calculé  |
| `DOZZLE_LEVEL`<br>`--level`               | Niveau de log de Dozzle lui-même. Voir [débogage](/fr/guide/debugging).                                          | `trace`, `debug`, `info`, `warn`, `error`                         | `info`   |
| `DOZZLE_MODE`<br>`--mode`                 | Mode de déploiement.                                                                                             | `server`, [`swarm`](/fr/guide/swarm-mode), [`k8s`](/fr/guide/k8s) | `server` |
| `DOZZLE_TIMEOUT`<br>`--timeout`           | Délai d'expiration des appels à l'API Docker ou Kubernetes.                                                      | une durée, par ex. `30s`                                          | `10s`    |
| `DOZZLE_NO_ANALYTICS`<br>`--no-analytics` | Désactive les [statistiques d'utilisation](/fr/guide/analytics) anonymes.                                        | `true`, `false`                                                   | `false`  |

## Conteneurs et hôtes

| Variable                                  | Description                                                                                                                           | Valeurs                              | Défaut            |
| ----------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------ | ----------------- |
| `DOZZLE_FILTER`<br>`--filter`             | N'affiche que les conteneurs correspondant à un filtre Docker. Voir [filtres](/fr/guide/filters).                                     | `key=value`, par ex. `label=app=web` | aucun             |
| `DOZZLE_REMOTE_AGENT`<br>`--remote-agent` | [Agents](/fr/guide/agent) auxquels se connecter, avec un nom d'affichage et un groupe facultatifs.                                    | `host:port[\|name[\|group]]`         | aucun             |
| `DOZZLE_REMOTE_HOST`<br>`--remote-host`   | Hôtes Docker auxquels se connecter en TCP. Voir [hôtes distants](/fr/guide/remote-hosts).                                             | `tcp://host:port[\|label]`           | aucun             |
| `DOZZLE_NAMESPACE`<br>`--namespace`       | Namespaces Kubernetes à surveiller. Utilisé uniquement avec `DOZZLE_MODE=k8s`.                                                        | noms de namespaces                   | tous              |
| `DOZZLE_CERT`<br>`--cert`                 | Certificat TLS utilisé pour communiquer avec les agents. Voir [certificats personnalisés](/fr/guide/agent#certificats-personnalises). | un chemin de fichier                 | `dozzle_cert.pem` |
| `DOZZLE_KEY`<br>`--key`                   | Clé privée TLS utilisée pour communiquer avec les agents.                                                                             | un chemin de fichier                 | `dozzle_key.pem`  |

## Fonctionnalités

| Variable                                              | Description                                                                                                                                                                           | Valeurs                      | Défaut                                  |
| ----------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------- | --------------------------------------- |
| `DOZZLE_ENABLE_ACTIONS`<br>`--enable-actions`         | Permet de démarrer, arrêter, redémarrer, supprimer et mettre à jour les conteneurs depuis l'interface. Voir [actions](/fr/guide/actions).                                             | `true`, `false`              | `false`                                 |
| `DOZZLE_ENABLE_SHELL`<br>`--enable-shell`             | Permet de s'attacher aux conteneurs et d'y lancer un shell depuis l'interface. Voir [shell](/fr/guide/shell).                                                                         | `true`, `false`              | `false`                                 |
| `DOZZLE_ENABLE_MCP`<br>`--enable-mcp`                 | Expose l'endpoint [MCP](/fr/guide/mcp) pour les clients LLM.                                                                                                                          | `true`, `false`              | `false`                                 |
| `DOZZLE_DISABLE_AVATARS`<br>`--disable-avatars`       | Masque les avatars des utilisateurs quand l'authentification est activée.                                                                                                             | `true`, `false`              | `false`                                 |
| `DOZZLE_RELEASE_CHECK_MODE`<br>`--release-check-mode` | Indique si Dozzle vérifie ses propres nouvelles versions. `manual` ne vérifie que lorsque vous le demandez.                                                                           | `automatic`, `manual`        | `automatic`                             |
| `DOZZLE_IMAGE_CHECK_MODE`<br>`--image-check-mode`     | Indique si Dozzle interroge les registres pour trouver des images de conteneurs plus récentes. Voir [vérification des mises à jour](/fr/guide/actions#verification-des-mises-a-jour). | `automatic`, `manual`, `off` | identique à `DOZZLE_RELEASE_CHECK_MODE` |
| `DOZZLE_AUTO_UPDATE`<br>`--auto-update`               | Met à jour le conteneur de Dozzle lui-même selon un planning. `weekly` s'exécute le dimanche. Nécessite `DOZZLE_ENABLE_ACTIONS`.                                                      | `off`, `daily`, `weekly`     | `off`                                   |
| `DOZZLE_AUTO_UPDATE_TIME`<br>`--auto-update-time`     | Heure de la journée à laquelle la mise à jour automatique s'exécute, dans l'heure locale du serveur.                                                                                  | `HH:MM`, par ex. `04:30`     | `03:00`                                 |

## Authentification

Voir [authentification](/fr/guide/authentication) pour le fonctionnement des fournisseurs.

| Variable                                        | Description                                                                                                                      | Valeurs                                                      | Défaut    |
| ----------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------ | --------- |
| `DOZZLE_AUTH_PROVIDER`<br>`--auth-provider`     | Fournisseur d'authentification à utiliser. `github` et `google` sont des alias de `simple`.                                      | `none`, `simple`, `oidc`, `forward-proxy`                    | `none`    |
| `DOZZLE_AUTH_TTL`<br>`--auth-ttl`               | Durée de validité d'une connexion. `session` déconnecte à la fermeture du navigateur.                                            | `session` ou une durée, par ex. `48h` (unités `s`, `m`, `h`) | `session` |
| `DOZZLE_AUTH_LOGOUT_URL`<br>`--auth-logout-url` | Où rediriger l'utilisateur à la déconnexion avec `forward-proxy`. Avec `oidc`, remplace l'endpoint de déconnexion de l'émetteur. | une URL                                                      | aucun     |

### GitHub OAuth

Ajoute « Se connecter avec GitHub » à l'authentification `simple`. Voir [OAuth](/fr/guide/authentication/oauth).

| Variable                                                            | Description                                  | Valeurs    | Défaut |
| ------------------------------------------------------------------- | -------------------------------------------- | ---------- | ------ |
| `DOZZLE_AUTH_GITHUB_CLIENT_ID`<br>`--auth-github-client-id`         | Client id de l'application OAuth GitHub.     | une chaîne | aucun  |
| `DOZZLE_AUTH_GITHUB_CLIENT_SECRET`<br>`--auth-github-client-secret` | Client secret de l'application OAuth GitHub. | une chaîne | aucun  |

### OpenID Connect

Utilisé avec `DOZZLE_AUTH_PROVIDER=oidc`. Voir [OIDC](/fr/guide/authentication/oidc).

| Variable                                                        | Description                                                                                                                                                                                                                          | Valeurs                                  | Défaut |
| --------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ---------------------------------------- | ------ |
| `DOZZLE_AUTH_OIDC_ISSUER`<br>`--auth-oidc-issuer`               | URL de l'émetteur du fournisseur d'identité.                                                                                                                                                                                         | une URL                                  | aucun  |
| `DOZZLE_AUTH_OIDC_CLIENT_ID`<br>`--auth-oidc-client-id`         | Client id enregistré auprès du fournisseur.                                                                                                                                                                                          | une chaîne                               | aucun  |
| `DOZZLE_AUTH_OIDC_CLIENT_SECRET`<br>`--auth-oidc-client-secret` | Client secret enregistré auprès du fournisseur.                                                                                                                                                                                      | une chaîne                               | aucun  |
| `DOZZLE_AUTH_OIDC_NAME`<br>`--auth-oidc-name`                   | Libellé du bouton de connexion.                                                                                                                                                                                                      | n'importe quelle chaîne                  | `SSO`  |
| `DOZZLE_AUTH_OIDC_ROLES_CLAIM`<br>`--auth-oidc-roles-claim`     | Claim depuis lequel lire les rôles. Nécessaire seulement quand la recherche par défaut (`dozzle_roles`, `resource_access.<client-id>.roles`, `roles`) ne le trouve pas.                                                              | un chemin de claim séparé par des points | aucun  |
| `DOZZLE_AUTH_OIDC_FILTERS_CLAIM`<br>`--auth-oidc-filters-claim` | Claim depuis lequel lire les filtres de conteneurs. Nécessaire seulement quand la recherche par défaut (`dozzle_filters`, `resource_access.<client-id>.filters`, `filters`) ne le trouve pas.                                        | un chemin de claim séparé par des points | aucun  |
| `DOZZLE_AUTH_OIDC_SCOPES`<br>`--auth-oidc-scopes`               | Scopes supplémentaires à demander en plus de `openid`, `profile` et `email`, pour les fournisseurs qui ne délivrent un claim que [lorsque son scope est demandé](/fr/guide/authentication/oidc#demander-des-scopes-supplementaires). | scopes séparés par des virgules          | aucun  |

> [!TIP]
> `DOZZLE_AUTH_GITHUB_CLIENT_SECRET` et `DOZZLE_AUTH_OIDC_CLIENT_SECRET` acceptent aussi un équivalent `_FILE` qui nomme un fichier depuis lequel lire la valeur, à utiliser avec les [secrets Docker](/fr/guide/authentication/oauth#utiliser-les-secrets-docker-pour-le-client-secret).

### Forward Proxy

Utilisé avec `DOZZLE_AUTH_PROVIDER=forward-proxy`. Chaque variable nomme l'en-tête HTTP défini par le proxy. Voir [forward proxy](/fr/guide/authentication/forward-proxy).

| Variable                                              | Description                                                 | Valeurs          | Défaut          |
| ----------------------------------------------------- | ----------------------------------------------------------- | ---------------- | --------------- |
| `DOZZLE_AUTH_HEADER_USER`<br>`--auth-header-user`     | En-tête contenant le nom d'utilisateur.                     | un nom d'en-tête | `Remote-User`   |
| `DOZZLE_AUTH_HEADER_EMAIL`<br>`--auth-header-email`   | En-tête contenant l'e-mail.                                 | un nom d'en-tête | `Remote-Email`  |
| `DOZZLE_AUTH_HEADER_NAME`<br>`--auth-header-name`     | En-tête contenant le nom d'affichage.                       | un nom d'en-tête | `Remote-Name`   |
| `DOZZLE_AUTH_HEADER_FILTER`<br>`--auth-header-filter` | En-tête contenant le filtre de conteneurs de l'utilisateur. | un nom d'en-tête | `Remote-Filter` |
| `DOZZLE_AUTH_HEADER_ROLES`<br>`--auth-header-roles`   | En-tête contenant les rôles de l'utilisateur.               | un nom d'en-tête | `Remote-Roles`  |

## Sous-commandes

### generate

Génère une entrée `users.yml` pour l'[authentification simple](/fr/guide/authentication/simple). Le nom d'utilisateur est le premier argument.

```sh
docker run -it --rm amir20/dozzle generate admin --password password --email test@email.net --name "John Doe" --user-filter name=foo --user-roles shell > users.yml
```

| Option             | Description                                                                                      | Valeurs                                                                 |
| ------------------ | ------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------- |
| `--password`, `-p` | Mot de passe de l'utilisateur.                                                                   | une chaîne                                                              |
| `--email`, `-e`    | E-mail de l'utilisateur, utilisé pour l'avatar.                                                  | un e-mail                                                               |
| `--name`, `-n`     | Nom d'affichage de l'utilisateur.                                                                | une chaîne                                                              |
| `--user-filter`    | Conteneurs que l'utilisateur peut voir.                                                          | filtres `key=value` séparés par des virgules                            |
| `--user-roles`     | Ce que l'utilisateur peut faire. Préfixez un rôle par `^` pour le retirer, par ex. `all,^shell`. | `all`, `none`, `shell`, `actions`, `download`, `notifications`, `cloud` |

### agent

Exécute Dozzle en tant qu'[agent](/fr/guide/agent) auquel une autre instance de Dozzle se connecte.

```sh
docker run -v /var/run/docker.sock:/var/run/docker.sock -p 7007:7007 amir20/dozzle agent
```

| Variable                              | Description                                  | Valeurs     | Défaut  |
| ------------------------------------- | -------------------------------------------- | ----------- | ------- |
| `DOZZLE_AGENT_ADDR`<br>`--agent-addr` | Adresse et port sur lesquels l'agent écoute. | `host:port` | `:7007` |

### generate-certs

Génère un certificat et une clé uniques pour les connexions aux agents, à la place de ceux partagés fournis avec Dozzle. Voir [certificats personnalisés](/fr/guide/agent#certificats-personnalises).

| Option       | Description                    | Défaut            |
| ------------ | ------------------------------ | ----------------- |
| `--cert-out` | Où écrire le certificat.       | `dozzle_cert.pem` |
| `--key-out`  | Où écrire la clé privée.       | `dozzle_key.pem`  |
| `--force`    | Écrase les fichiers existants. | `false`           |

### healthcheck

Vérifie que le serveur ou l'agent est opérationnel. Elle n'est pas branchée dans l'image par défaut car elle ajoute un peu de charge CPU. Voir [healthcheck](/fr/guide/healthcheck).
