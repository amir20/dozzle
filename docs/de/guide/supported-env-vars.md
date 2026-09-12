---
title: Umgebungsvariablen und Unterbefehle
sourceHash: a9818d761eb6
---

# Globale Umgebungsvariablen

Die Konfiguration erfolgt über Optionen oder Umgebungsvariablen. Die Tabelle unten listet alle unterstützten Optionen und die zugehörigen Umgebungsvariablen auf.

| Option                        | Umgebungsvariable                  | Standard          |
| ----------------------------- | ---------------------------------- | ----------------- |
| `--addr`                      | `DOZZLE_ADDR`                      | `:8080`           |
| `--base`                      | `DOZZLE_BASE`                      | `/`               |
| `--hostname`                  | `DOZZLE_HOSTNAME`                  | `""`              |
| `--host-id`                   | `DOZZLE_HOST_ID`                   | `""`              |
| `--level`                     | `DOZZLE_LEVEL`                     | `info`            |
| `--auth-provider`             | `DOZZLE_AUTH_PROVIDER`             | `none`            |
| `--auth-header-user`          | `DOZZLE_AUTH_HEADER_USER`          | `Remote-User`     |
| `--auth-header-email`         | `DOZZLE_AUTH_HEADER_EMAIL`         | `Remote-Email`    |
| `--auth-header-name`          | `DOZZLE_AUTH_HEADER_NAME`          | `Remote-Name`     |
| `--auth-header-filter`        | `DOZZLE_AUTH_HEADER_FILTER`        | `Remote-Filter`   |
| `--auth-header-roles`         | `DOZZLE_AUTH_HEADER_ROLES`         | `Remote-Roles`    |
| `--auth-logout-url`           | `DOZZLE_AUTH_LOGOUT_URL`           | `""`              |
| `--auth-ttl`                  | `DOZZLE_AUTH_TTL`                  | `session`         |
| `--auth-github-client-id`     | `DOZZLE_AUTH_GITHUB_CLIENT_ID`     | `""`              |
| `--auth-github-client-secret` | `DOZZLE_AUTH_GITHUB_CLIENT_SECRET` | `""`              |
| `--auth-oidc-issuer`          | `DOZZLE_AUTH_OIDC_ISSUER`          | `""`              |
| `--auth-oidc-client-id`       | `DOZZLE_AUTH_OIDC_CLIENT_ID`       | `""`              |
| `--auth-oidc-client-secret`   | `DOZZLE_AUTH_OIDC_CLIENT_SECRET`   | `""`              |
| `--auth-oidc-name`            | `DOZZLE_AUTH_OIDC_NAME`            | `SSO`             |
| `--auth-oidc-roles-claim`     | `DOZZLE_AUTH_OIDC_ROLES_CLAIM`     | `""`              |
| `--auth-oidc-filters-claim`   | `DOZZLE_AUTH_OIDC_FILTERS_CLAIM`   | `""`              |
| `--enable-actions`            | `DOZZLE_ENABLE_ACTIONS`            | `false`           |
| `--enable-shell`              | `DOZZLE_ENABLE_SHELL`              | `false`           |
| `--enable-mcp`                | `DOZZLE_ENABLE_MCP`                | `false`           |
| `--disable-avatars`           | `DOZZLE_DISABLE_AVATARS`           | `false`           |
| `--filter`                    | `DOZZLE_FILTER`                    | `""`              |
| `--no-analytics`              | `DOZZLE_NO_ANALYTICS`              | `false`           |
| `--mode`                      | `DOZZLE_MODE`                      | `server`          |
| `--release-check-mode`        | `DOZZLE_RELEASE_CHECK_MODE`        | `automatic`       |
| `--image-check-mode`          | `DOZZLE_IMAGE_CHECK_MODE`          | geerbt            |
| `--remote-host`               | `DOZZLE_REMOTE_HOST`               |                   |
| `--remote-agent`              | `DOZZLE_REMOTE_AGENT`              |                   |
| `--timeout`                   | `DOZZLE_TIMEOUT`                   | `10s`             |
| `--namespace`                 | `DOZZLE_NAMESPACE`                 | `""`              |
| `--cert`                      | `DOZZLE_CERT`                      | `dozzle_cert.pem` |
| `--key`                       | `DOZZLE_KEY`                       | `dozzle_key.pem`  |

> [!TIP]
> `DOZZLE_AUTH_GITHUB_CLIENT_SECRET` und `DOZZLE_AUTH_OIDC_CLIENT_SECRET` akzeptieren auch ein Gegenstück mit `_FILE`, das eine Datei benennt, aus der der Wert gelesen wird, zur Nutzung mit [Docker Secrets](/de/guide/authentication/oauth#docker-secrets-fur-das-client-secret-verwenden).

> [!TIP]
> `DOZZLE_AUTH_OIDC_ROLES_CLAIM` und `DOZZLE_AUTH_OIDC_FILTERS_CLAIM` gelten nur für [`--auth-provider oidc`](/de/guide/authentication/oidc) und werden nur gebraucht, wenn die Claims an einer Stelle liegen, an der die Standardsuche nicht nachsieht.

> [!TIP]
> Manche Optionen wie `--remote-host` oder `--remote-agent` lassen sich mehrfach angeben. Zum Beispiel `--remote-agent 167.99.1.1:7007 --remote-agent 167.99.1.2:7007` oder kommagetrennt `DOZZLE_REMOTE_AGENT=167.99.1.1:7007,167.99.1.2:7007`.

## users.yml erzeugen

Dozzle kann eine `users.yml`-Datei erzeugen. Diese Datei wird zur Authentifizierung von Benutzern verwendet. Hier ein Beispiel:

```sh
docker run -it --rm amir20/dozzle generate admin --password password --email test@email.net --name "John Doe" --user-filter name=foo --user-roles shell > users.yml
```

In diesem Beispiel ist `admin` der Benutzername. E-Mail und Name sind optional, aber empfehlenswert, damit die Avatare korrekt angezeigt werden. `docker run amir20/dozzle generate --help` zeigt alle Optionen.

| Option          | Beschreibung           | Standard |
| --------------- | ---------------------- | -------- |
| `--password`    | Passwort des Benutzers |          |
| `--email`       | E-Mail des Benutzers   |          |
| `--name`        | Vollständiger Name     |          |
| `--user-filter` | Filter des Benutzers   |          |
| `--user-roles`  | Rollen des Benutzers   |          |

Mehr dazu unter [Authentifizierung](/de/guide/authentication).

## Agent-Modus

Dozzle kann im Agent-Modus laufen. Der Agent-Modus ist nützlich, wenn Dozzle auf einem entfernten Host läuft und du einen anderen Docker-Host überwachen willst. Der Agent-Modus wird über die Option `--remote-agent` aktiviert. Hier ein Beispiel:

```sh
docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --remote-agent remote-ip:7007
```

| Option   | Umgebungsvariable   | Standard |
| -------- | ------------------- | -------- |
| `--addr` | `DOZZLE_AGENT_ADDR` | `:7007`  |

Mehr dazu unter [Agent](/de/guide/agent).

## Healthcheck

Dozzle unterstützt Healthchecks über den Befehl `dozzle healthcheck`. Er ist standardmäßig nicht aktiv, da er zusätzliche CPU-Last erzeugt. Um `healthcheck` zu nutzen, musst du ihn konfigurieren.

Mehr dazu unter [Healthcheck](/de/guide/healthcheck).
