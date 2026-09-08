---
title: Forward Proxy
sourceHash: 37a5c3119fb2
---

# <Icon icon="mdi:swap-horizontal" inline /> Forward Proxy

Dozzle kann Proxy-Header lesen, wenn `--auth-provider` auf `forward-proxy` gesetzt ist.

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

Hänge `/data` auch hier ein. Benutzereinstellungen werden auch im Forward-Proxy-Modus auf die Festplatte geschrieben und gehen ohne das Volume bei jedem Neuerstellen des Containers verloren.

In diesem Modus erwartet Dozzle die folgenden Header:

- `Remote-User` als Benutzername, z. B. `johndoe`
- `Remote-Email` als E-Mail-Adresse des Benutzers. Über diese E-Mail wird auch der passende [Gravatar](https://gravatar.com/) gefunden.
- `Remote-Name` als Anzeigename wie `John Doe`
- `Remote-Filter` als kommagetrennte Liste der für den Benutzer erlaubten Filter.
- `Remote-Roles` als kommagetrennte Liste der für den Benutzer erlaubten Rollen.

Zusätzlich kannst du eine Logout-URL konfigurieren:

```yaml
DOZZLE_AUTH_LOGOUT_URL: http://oauth2.example.ru/oauth2/sign_out
```

## Dozzle mit Authelia einrichten

[Authelia](https://www.authelia.com/) ist ein quelloffener Authentifizierungs- und Autorisierungsserver samt Portal für Identitäts- und Zugriffsverwaltung. Die Einrichtung von Authelia selbst geht über diesen Abschnitt hinaus, aber die Konfiguration lässt sich als Beispiel für die Einrichtung von Dozzle mit Authelia teilen.

<details>
<summary>➡️ Zum Aufklappen des Authelia-Beispiels klicken</summary>

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
    - domain: example.com # Sollte zu deiner geschützten Root-Domain passen
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

Gültige SSL-Schlüssel sind erforderlich, da Authelia nur SSL unterstützt.

Authelia schickt die Gruppenzugehörigkeit in `Remote-Groups`, und Dozzle liest diesen Header nicht standardmäßig. Um Authelia-Gruppen auf Dozzle-[Rollen](/de/guide/authentication/simple#bestimmte-rollen-fur-benutzer-setzen) abzubilden, setze `DOZZLE_AUTH_HEADER_ROLES: Remote-Groups` beim Dozzle-Service und benenne die Gruppen nach den Rollen. Genau dafür gibt es die Aliase mit dem Präfix `dozzle_`: Eine Gruppe namens `dozzle_shell` gewährt die Rolle `shell`, andere Gruppennamen werden ignoriert. Ohne diese Zuordnung bekommt jeder authentifizierte Benutzer alle Rollen.

</details>

## Dozzle mit Cloudflare Zero Trust einrichten

Cloudflare Zero Trust ist ein Dienst für authentifizierten Zugriff auf selbst gehostete Software. Dieser Abschnitt beschreibt, wie du Dozzle für die Authentifizierung über Cloudflare Zero Trust einrichtest.

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

`expose` hält Port 8080 vom Host fern, der einzige Weg hinein führt also über den Tunnel. Wird er mit `ports` veröffentlicht, könnte jeder auf dem Host `Cf-Access-Authenticated-User-Email` selbst setzen und Cloudflare komplett umgehen.

Nachdem der Dozzle-Container läuft, konfigurierst du die Anwendung im Cloudflare-Zero-Trust-Dashboard nach dieser [Anleitung](https://developers.cloudflare.com/cloudflare-one/applications/configure-apps/self-hosted-apps/).

## Dozzle mit Pocket ID einrichten

> [!TIP]
> Dozzle spricht inzwischen selbst OpenID Connect, du kannst `--auth-oidc-issuer` also direkt auf Pocket ID zeigen lassen und oauth2-proxy komplett weglassen. Siehe [Mit GitHub & OIDC anmelden](/de/guide/authentication/oauth#mit-oidc-anmelden). Das Rezept unten bleibt für Setups, die schon oauth2-proxy betreiben oder den Proxy mehr als nur Dozzle absichern lassen wollen.

Du musst zuerst einen Container einrichten, der die OpenID-Connect-Authentifizierung durch deinen Reverse Proxy reicht.

Unten ein Beispiel mit [oauth2-proxy](https://github.com/oauth2-proxy/oauth2-proxy).

<details>
<summary>➡️ Zum Aufklappen des oauth2-proxy-Beispiels klicken</summary>

1. Lege in Pocket ID einen neuen OIDC-Client für Dozzle an:
   - **Name:** `Dozzle`
   - **Callback-URLs:** `https://dozzle.example.com/oauth2/callback`
   - **PKCE:** `Enabled`

   Kopiere die Werte für **Client ID** und **Client Secret** für später.

2. Ergänze deine bestehende Dozzle-Compose-Datei um Folgendes:

   ```yml
   environment:
     DOZZLE_AUTH_PROVIDER: forward-proxy
     DOZZLE_AUTH_HEADER_USER: X-Forwarded-User
     DOZZLE_AUTH_HEADER_EMAIL: X-Forwarded-Email
     DOZZLE_AUTH_HEADER_NAME: X-Forwarded-Preferred-Username
   ```

   Kommentiere die Ports von Dozzle aus, da wir sie über den neuen Authentifizierungscontainer umleiten.

   Diese Methode sollte keine Änderungen an der Konfiguration deines Reverse Proxy erfordern.

   ```yml
   # ports:
   #   - 8080:8080
   ```

3. Füge deiner bestehenden Dozzle-Compose-Datei einen neuen oauth2-proxy-Service hinzu:

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

4. Lege die Konfigurationsdatei für oauth2-proxy an.

   Erstelle im Verzeichnis neben deiner Compose-Datei die Datei `oauth2-proxy.cfg` :

   ```toml
    client_id = "xxx"                            # aus Pocket ID
    client_secret = "xxx"                        # aus Pocket ID
    cookie_secret = "xxx"                        # mit openssl rand -base64 32 | tr -- '+/' '-_' erzeugen
    upstreams = "http://dozzle:8080"             # Upstream zum internen Port der Dozzle-Container
    code_challenge_method = "S256"               # PKCE-Challenges plain oder S256
    cookie_expire = "0"                          # Sekunden, 0 für Session
    cookie_name = "__Host-oauth2-proxy"          # oder __Secure-oauth2-proxy (weniger sicher)
    cookie_secure = true                         # nutzt das sichere HTTPS-Cookie
    email_domains = ["*"]                        # erlaubt die Anmeldung mit jeder E-Mail-Domain
    http_address = "0.0.0.0:4180"                # Port, auf dem oauth2-proxy lauscht
    oidc_issuer_url = "https://id.example.com"   # deine Pocket-Basis-URL
    provider_display_name = "Pocket ID"          # Anzeigename für den OIDC-Login
    provider = "oidc"                            # OpenID Connect verwenden
    reverse_proxy = true                         # Traffic über den Reverse Proxy leiten
    scope = "openid email profile groups"        # diese OIDC-Scopes durchreichen
   ```

   Fülle die Variablen entsprechend den Kommentaren aus.

5. Zum Schluss: Starte deinen Docker-Compose-Stack neu.

   Dein Reverse Proxy sollte dich jetzt über oauth2-proxy bei Dozzle authentifizieren.

   Bei Problemen hilft ein Blick in die Logs.

</details>
