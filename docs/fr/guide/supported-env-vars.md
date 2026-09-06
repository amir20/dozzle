---
title: Variables d'environnement et sous-commandes
sourceHash: 3930d18cbbb4
---

# Variables d'environnement globales

La configuration se fait avec des options en ligne de commande ou des variables d'environnement. Le tableau ci-dessous liste toutes les options prises en charge et les variables correspondantes.

| Option                 | Variable d'environnement    | Défaut            |
| ---------------------- | --------------------------- | ----------------- |
| `--addr`               | `DOZZLE_ADDR`               | `:8080`           |
| `--base`               | `DOZZLE_BASE`               | `/`               |
| `--hostname`           | `DOZZLE_HOSTNAME`           | `""`              |
| `--level`              | `DOZZLE_LEVEL`              | `info`            |
| `--auth-provider`      | `DOZZLE_AUTH_PROVIDER`      | `none`            |
| `--auth-header-user`   | `DOZZLE_AUTH_HEADER_USER`   | `Remote-User`     |
| `--auth-header-email`  | `DOZZLE_AUTH_HEADER_EMAIL`  | `Remote-Email`    |
| `--auth-header-name`   | `DOZZLE_AUTH_HEADER_NAME`   | `Remote-Name`     |
| `--auth-header-filter` | `DOZZLE_AUTH_HEADER_FILTER` | `Remote-Filter`   |
| `--auth-header-roles`  | `DOZZLE_AUTH_HEADER_ROLES`  | `Remote-Roles`    |
| `--auth-logout-url`    | `DOZZLE_AUTH_LOGOUT_URL`    | `""`              |
| `--auth-ttl`           | `DOZZLE_AUTH_TTL`           | `session`         |
| `--enable-actions`     | `DOZZLE_ENABLE_ACTIONS`     | `false`           |
| `--enable-shell`       | `DOZZLE_ENABLE_SHELL`       | `false`           |
| `--enable-mcp`         | `DOZZLE_ENABLE_MCP`         | `false`           |
| `--disable-avatars`    | `DOZZLE_DISABLE_AVATARS`    | `false`           |
| `--filter`             | `DOZZLE_FILTER`             | `""`              |
| `--no-analytics`       | `DOZZLE_NO_ANALYTICS`       | `false`           |
| `--mode`               | `DOZZLE_MODE`               | `server`          |
| `--release-check-mode` | `DOZZLE_RELEASE_CHECK_MODE` | `automatic`       |
| `--image-check-mode`   | `DOZZLE_IMAGE_CHECK_MODE`   | hérité            |
| `--remote-host`        | `DOZZLE_REMOTE_HOST`        |                   |
| `--remote-agent`       | `DOZZLE_REMOTE_AGENT`       |                   |
| `--timeout`            | `DOZZLE_TIMEOUT`            | `10s`             |
| `--namespace`          | `DOZZLE_NAMESPACE`          | `""`              |
| `--cert`               | `DOZZLE_CERT`               | `dozzle_cert.pem` |
| `--key`                | `DOZZLE_KEY`                | `dozzle_key.pem`  |

> [!TIP]
> Certaines options comme `--remote-host` ou `--remote-agent` peuvent être répétées. Par exemple, `--remote-agent 167.99.1.1:7007 --remote-agent 167.99.1.2:7007`, ou séparées par des virgules avec `DOZZLE_REMOTE_AGENT=167.99.1.1:7007,167.99.1.2:7007`.

## Générer users.yml

Dozzle peut générer un fichier `users.yml`. Ce fichier sert à authentifier les utilisateurs. Voici un exemple :

```sh
docker run -it --rm amir20/dozzle generate admin --password password --email test@email.net --name "John Doe" --user-filter name=foo --user-roles shell > users.yml
```

Dans cet exemple, `admin` est le nom d'utilisateur. L'e-mail et le nom sont facultatifs mais recommandés pour afficher les bons avatars. `docker run amir20/dozzle generate --help` affiche toutes les options.

| Option          | Description                   | Défaut |
| --------------- | ----------------------------- | ------ |
| `--password`    | Mot de passe de l'utilisateur |        |
| `--email`       | E-mail de l'utilisateur       |        |
| `--name`        | Nom complet de l'utilisateur  |        |
| `--user-filter` | Filtres de l'utilisateur      |        |
| `--user-roles`  | Rôles de l'utilisateur        |        |

Voir [authentification](/fr/guide/authentication) pour plus d'informations.

## Mode agent

Dozzle peut s'exécuter en mode agent. Le mode agent est utile quand Dozzle tourne sur un hôte distant et que vous voulez surveiller un autre hôte Docker. Le mode agent s'active avec l'option `--remote-agent`. Voici un exemple :

```sh
docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --remote-agent remote-ip:7007
```

| Option   | Variable d'environnement | Défaut  |
| -------- | ------------------------ | ------- |
| `--addr` | `DOZZLE_AGENT_ADDR`      | `:7007` |

Voir [agent](/fr/guide/agent) pour plus d'informations.

## Healthcheck

Dozzle prend en charge le healthcheck via la commande `dozzle healthcheck`. Il n'est pas activé par défaut car il consomme du CPU supplémentaire. Pour utiliser `healthcheck`, vous devez le configurer.

Voir [healthcheck](/fr/guide/healthcheck) pour plus d'informations.
