---
title: Umgebungsvariablen und Unterbefehle
sourceHash: 3930d18cbbb4
---

# Globale Umgebungsvariablen

Die Konfiguration erfolgt über Optionen oder Umgebungsvariablen. Die Tabelle unten listet alle unterstützten Optionen und die zugehörigen Umgebungsvariablen auf.

| Option                 | Umgebungsvariable           | Standard          |
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
| `--image-check-mode`   | `DOZZLE_IMAGE_CHECK_MODE`   | geerbt            |
| `--remote-host`        | `DOZZLE_REMOTE_HOST`        |                   |
| `--remote-agent`       | `DOZZLE_REMOTE_AGENT`       |                   |
| `--timeout`            | `DOZZLE_TIMEOUT`            | `10s`             |
| `--namespace`          | `DOZZLE_NAMESPACE`          | `""`              |
| `--cert`               | `DOZZLE_CERT`               | `dozzle_cert.pem` |
| `--key`                | `DOZZLE_KEY`                | `dozzle_key.pem`  |

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
