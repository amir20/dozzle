---
title: Umgebungsvariablen und Unterbefehle
sourceHash: 94da032e7db0
---

# Umgebungsvariablen

Jede Option lässt sich über eine Option oder eine Umgebungsvariable setzen. Optionen und Umgebungsvariablen haben immer Vorrang vor Einstellungen, die der [Einrichtungsassistent](/de/guide/setup-wizard) in `dozzle.yml` gespeichert hat.

Optionen, die eine Liste annehmen (`DOZZLE_FILTER`, `DOZZLE_REMOTE_HOST`, `DOZZLE_REMOTE_AGENT`, `DOZZLE_NAMESPACE`), akzeptieren in der Umgebungsvariable einen kommagetrennten Wert oder die Option einmal pro Eintrag wiederholt:

```sh
--remote-agent 167.99.1.1:7007 --remote-agent 167.99.1.2:7007
DOZZLE_REMOTE_AGENT=167.99.1.1:7007,167.99.1.2:7007
```

## Server

| Variable                                  | Beschreibung                                                                                                     | Werte                                                             | Standard   |
| ----------------------------------------- | ---------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------- | ---------- |
| `DOZZLE_ADDR`<br>`--addr`                 | Adresse und Port, auf denen der Webserver lauscht. In einem Container selten nötig.                              | `host:port`, z. B. `:9090`                                        | `:8080`    |
| `DOZZLE_BASE`<br>`--base`                 | Pfadpräfix, unter dem Dozzle ausgeliefert wird. Siehe [Basis-Pfad ändern](/de/guide/changing-base).              | ein Pfad, z. B. `/logs`                                           | `/`        |
| `DOZZLE_HOSTNAME`<br>`--hostname`         | Name dieser Instanz in der Oberfläche. Siehe [Hostname](/de/guide/hostname).                                     | beliebiger Text                                                   | keiner     |
| `DOZZLE_HOST_ID`<br>`--host-id`           | Überschreibt die ID, die Dozzle für diesen Host ableitet. Nur nötig, wenn sie mit einem anderen Host kollidiert. | Buchstaben, Ziffern, `_`, `.`, `-`                                | abgeleitet |
| `DOZZLE_LEVEL`<br>`--level`               | Log-Level von Dozzle selbst. Siehe [Debugging](/de/guide/debugging).                                             | `trace`, `debug`, `info`, `warn`, `error`                         | `info`     |
| `DOZZLE_MODE`<br>`--mode`                 | Betriebsmodus.                                                                                                   | `server`, [`swarm`](/de/guide/swarm-mode), [`k8s`](/de/guide/k8s) | `server`   |
| `DOZZLE_TIMEOUT`<br>`--timeout`           | Timeout für Aufrufe der Docker- oder Kubernetes-API.                                                             | eine Dauer, z. B. `30s`                                           | `10s`      |
| `DOZZLE_NO_ANALYTICS`<br>`--no-analytics` | Schaltet die anonyme [Analyse](/de/guide/analytics) ab.                                                          | `true`, `false`                                                   | `false`    |

## Container und Hosts

| Variable                                  | Beschreibung                                                                                                     | Werte                              | Standard          |
| ----------------------------------------- | ---------------------------------------------------------------------------------------------------------------- | ---------------------------------- | ----------------- |
| `DOZZLE_FILTER`<br>`--filter`             | Zeigt nur Container, die zu einem Docker-Filter passen. Siehe [Filter](/de/guide/filters).                       | `key=value`, z. B. `label=app=web` | keiner            |
| `DOZZLE_REMOTE_AGENT`<br>`--remote-agent` | [Agents](/de/guide/agent), mit denen verbunden wird, optional mit Anzeigename und Gruppe.                        | `host:port[\|name[\|group]]`       | keiner            |
| `DOZZLE_REMOTE_HOST`<br>`--remote-host`   | Docker-Hosts, mit denen über TCP verbunden wird. Siehe [Remote-Hosts](/de/guide/remote-hosts).                   | `tcp://host:port[\|label]`         | keiner            |
| `DOZZLE_NAMESPACE`<br>`--namespace`       | Kubernetes-Namespaces, die beobachtet werden. Nur mit `DOZZLE_MODE=k8s` verwendet.                               | Namespace-Namen                    | alle              |
| `DOZZLE_CERT`<br>`--cert`                 | TLS-Zertifikat für die Kommunikation mit Agents. Siehe [eigene Zertifikate](/de/guide/agent#eigene-zertifikate). | ein Dateipfad                      | `dozzle_cert.pem` |
| `DOZZLE_KEY`<br>`--key`                   | Privater TLS-Schlüssel für die Kommunikation mit Agents.                                                         | ein Dateipfad                      | `dozzle_key.pem`  |

## Funktionen

| Variable                                              | Beschreibung                                                                                                                              | Werte                        | Standard                        |
| ----------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------- | ------------------------------- |
| `DOZZLE_ENABLE_ACTIONS`<br>`--enable-actions`         | Erlaubt Starten, Stoppen, Neustarten, Entfernen und Aktualisieren von Containern aus der Oberfläche. Siehe [Aktionen](/de/guide/actions). | `true`, `false`              | `false`                         |
| `DOZZLE_ENABLE_SHELL`<br>`--enable-shell`             | Erlaubt, sich aus der Oberfläche an Container anzuhängen und darin eine Shell zu starten. Siehe [Shell](/de/guide/shell).                 | `true`, `false`              | `false`                         |
| `DOZZLE_ENABLE_MCP`<br>`--enable-mcp`                 | Stellt den [MCP](/de/guide/mcp)-Endpunkt für LLM-Clients bereit.                                                                          | `true`, `false`              | `false`                         |
| `DOZZLE_DISABLE_AVATARS`<br>`--disable-avatars`       | Blendet Benutzeravatare aus, wenn die Authentifizierung aktiv ist.                                                                        | `true`, `false`              | `false`                         |
| `DOZZLE_RELEASE_CHECK_MODE`<br>`--release-check-mode` | Ob Dozzle nach neuen eigenen Releases sucht. `manual` prüft nur, wenn du danach fragst.                                                   | `automatic`, `manual`        | `automatic`                     |
| `DOZZLE_IMAGE_CHECK_MODE`<br>`--image-check-mode`     | Ob Dozzle in Registries nach neueren Container-Images sucht. Siehe [Update-Prüfung](/de/guide/actions#update-prufung).                    | `automatic`, `manual`, `off` | wie `DOZZLE_RELEASE_CHECK_MODE` |
| `DOZZLE_AUTO_UPDATE`<br>`--auto-update`               | Aktualisiert den eigenen Container von Dozzle nach Zeitplan. `weekly` läuft am Sonntag. Setzt `DOZZLE_ENABLE_ACTIONS` voraus.             | `off`, `daily`, `weekly`     | `off`                           |
| `DOZZLE_AUTO_UPDATE_TIME`<br>`--auto-update-time`     | Uhrzeit, zu der das automatische Update läuft, in der lokalen Zeit des Servers.                                                           | `HH:MM`, z. B. `04:30`       | `03:00`                         |

## Authentifizierung

Wie die Anbieter funktionieren, steht unter [Authentifizierung](/de/guide/authentication).

| Variable                                        | Beschreibung                                                                                                                      | Werte                                                            | Standard  |
| ----------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------- | --------- |
| `DOZZLE_AUTH_PROVIDER`<br>`--auth-provider`     | Welcher Authentifizierungsanbieter verwendet wird. `github` und `google` sind Aliase für `simple`.                                | `none`, `simple`, `oidc`, `forward-proxy`                        | `none`    |
| `DOZZLE_AUTH_TTL`<br>`--auth-ttl`               | Wie lange eine Anmeldung gilt. `session` meldet ab, wenn der Browser geschlossen wird.                                            | `session` oder eine Dauer, z. B. `48h` (Einheiten `s`, `m`, `h`) | `session` |
| `DOZZLE_AUTH_LOGOUT_URL`<br>`--auth-logout-url` | Wohin der Benutzer beim Abmelden mit `forward-proxy` geschickt wird. Mit `oidc` überschreibt sie den Logout-Endpunkt des Issuers. | eine URL                                                         | keiner    |

### GitHub OAuth

Fügt `simple`-Authentifizierung ein „Mit GitHub anmelden" hinzu. Siehe [OAuth](/de/guide/authentication/oauth).

| Variable                                                            | Beschreibung                        | Werte      | Standard |
| ------------------------------------------------------------------- | ----------------------------------- | ---------- | -------- |
| `DOZZLE_AUTH_GITHUB_CLIENT_ID`<br>`--auth-github-client-id`         | Client-ID der GitHub-OAuth-App.     | ein String | keiner   |
| `DOZZLE_AUTH_GITHUB_CLIENT_SECRET`<br>`--auth-github-client-secret` | Client Secret der GitHub-OAuth-App. | ein String | keiner   |

### OpenID Connect

Wird mit `DOZZLE_AUTH_PROVIDER=oidc` verwendet. Siehe [OIDC](/de/guide/authentication/oidc).

| Variable                                                        | Beschreibung                                                                                                                                                                                                                     | Werte                          | Standard |
| --------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------ | -------- |
| `DOZZLE_AUTH_OIDC_ISSUER`<br>`--auth-oidc-issuer`               | Issuer-URL des Identitätsanbieters.                                                                                                                                                                                              | eine URL                       | keiner   |
| `DOZZLE_AUTH_OIDC_CLIENT_ID`<br>`--auth-oidc-client-id`         | Beim Anbieter registrierte Client-ID.                                                                                                                                                                                            | ein String                     | keiner   |
| `DOZZLE_AUTH_OIDC_CLIENT_SECRET`<br>`--auth-oidc-client-secret` | Beim Anbieter registriertes Client Secret.                                                                                                                                                                                       | ein String                     | keiner   |
| `DOZZLE_AUTH_OIDC_NAME`<br>`--auth-oidc-name`                   | Beschriftung des Anmeldebuttons.                                                                                                                                                                                                 | beliebiger Text                | `SSO`    |
| `DOZZLE_AUTH_OIDC_ROLES_CLAIM`<br>`--auth-oidc-roles-claim`     | Claim, aus dem die Rollen gelesen werden. Nur nötig, wenn die Standardsuche (`dozzle_roles`, `resource_access.<client-id>.roles`, `roles`) ihn nicht findet.                                                                     | ein punktgetrennter Claim-Pfad | keiner   |
| `DOZZLE_AUTH_OIDC_FILTERS_CLAIM`<br>`--auth-oidc-filters-claim` | Claim, aus dem die Container-Filter gelesen werden. Nur nötig, wenn die Standardsuche (`dozzle_filters`, `resource_access.<client-id>.filters`, `filters`) ihn nicht findet.                                                     | ein punktgetrennter Claim-Pfad | keiner   |
| `DOZZLE_AUTH_OIDC_SCOPES`<br>`--auth-oidc-scopes`               | Zusätzliche Scopes, die neben `openid`, `profile` und `email` angefordert werden, für Anbieter, die einen Claim nur herausgeben, [wenn sein Scope angefordert wird](/de/guide/authentication/oidc#zusatzliche-scopes-anfordern). | kommagetrennte Scopes          | keiner   |

> [!TIP]
> `DOZZLE_AUTH_GITHUB_CLIENT_SECRET` und `DOZZLE_AUTH_OIDC_CLIENT_SECRET` akzeptieren auch ein Gegenstück mit `_FILE`, das eine Datei benennt, aus der der Wert gelesen wird, zur Nutzung mit [Docker Secrets](/de/guide/authentication/oauth#docker-secrets-fur-das-client-secret-verwenden).

### Forward Proxy

Wird mit `DOZZLE_AUTH_PROVIDER=forward-proxy` verwendet. Jede Variable benennt den HTTP-Header, den der Proxy setzt. Siehe [Forward Proxy](/de/guide/authentication/forward-proxy).

| Variable                                              | Beschreibung                                   | Werte           | Standard        |
| ----------------------------------------------------- | ---------------------------------------------- | --------------- | --------------- |
| `DOZZLE_AUTH_HEADER_USER`<br>`--auth-header-user`     | Header mit dem Benutzernamen.                  | ein Header-Name | `Remote-User`   |
| `DOZZLE_AUTH_HEADER_EMAIL`<br>`--auth-header-email`   | Header mit der E-Mail.                         | ein Header-Name | `Remote-Email`  |
| `DOZZLE_AUTH_HEADER_NAME`<br>`--auth-header-name`     | Header mit dem Anzeigenamen.                   | ein Header-Name | `Remote-Name`   |
| `DOZZLE_AUTH_HEADER_FILTER`<br>`--auth-header-filter` | Header mit dem Container-Filter des Benutzers. | ein Header-Name | `Remote-Filter` |
| `DOZZLE_AUTH_HEADER_ROLES`<br>`--auth-header-roles`   | Header mit den Rollen des Benutzers.           | ein Header-Name | `Remote-Roles`  |

## Unterbefehle

### generate

Erzeugt einen `users.yml`-Eintrag für die [einfache Authentifizierung](/de/guide/authentication/simple). Der Benutzername ist das erste Argument.

```sh
docker run -it --rm amir20/dozzle generate admin --password password --email test@email.net --name "John Doe" --user-filter name=foo --user-roles shell > users.yml
```

| Option             | Beschreibung                                                                            | Werte                                                                   |
| ------------------ | --------------------------------------------------------------------------------------- | ----------------------------------------------------------------------- |
| `--password`, `-p` | Passwort des Benutzers.                                                                 | ein String                                                              |
| `--email`, `-e`    | E-Mail des Benutzers, wird für den Avatar verwendet.                                    | eine E-Mail-Adresse                                                     |
| `--name`, `-n`     | Anzeigename des Benutzers.                                                              | ein String                                                              |
| `--user-filter`    | Container, die der Benutzer sehen darf.                                                 | kommagetrennte `key=value`-Filter                                       |
| `--user-roles`     | Was der Benutzer darf. Ein vorangestelltes `^` entfernt eine Rolle, z. B. `all,^shell`. | `all`, `none`, `shell`, `actions`, `download`, `notifications`, `cloud` |

### agent

Startet Dozzle als [Agent](/de/guide/agent), mit dem sich eine andere Dozzle-Instanz verbindet.

```sh
docker run -v /var/run/docker.sock:/var/run/docker.sock -p 7007:7007 amir20/dozzle agent
```

| Variable                              | Beschreibung                                   | Werte       | Standard |
| ------------------------------------- | ---------------------------------------------- | ----------- | -------- |
| `DOZZLE_AGENT_ADDR`<br>`--agent-addr` | Adresse und Port, auf denen der Agent lauscht. | `host:port` | `:7007`  |

### generate-certs

Erzeugt ein eigenes Zertifikat samt Schlüssel für Agent-Verbindungen, anstelle des gemeinsamen, das Dozzle mitbringt. Siehe [eigene Zertifikate](/de/guide/agent#eigene-zertifikate).

| Option       | Beschreibung                                  | Standard          |
| ------------ | --------------------------------------------- | ----------------- |
| `--cert-out` | Wohin das Zertifikat geschrieben wird.        | `dozzle_cert.pem` |
| `--key-out`  | Wohin der private Schlüssel geschrieben wird. | `dozzle_key.pem`  |
| `--force`    | Überschreibt vorhandene Dateien.              | `false`           |

### healthcheck

Prüft, ob Server oder Agent laufen. Er ist im Image standardmäßig nicht eingebunden, weil er etwas zusätzliche CPU-Last erzeugt. Siehe [Healthcheck](/de/guide/healthcheck).
