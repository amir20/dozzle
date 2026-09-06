---
title: Authentifizierung
sourceHash: 4ce1bd2c3816
---

# Authentifizierung

Dozzle unterstützt zwei Konfigurationen für die Authentifizierung. In der ersten bringst du deine eigene Authentifizierungsmethode mit, indem du Dozzle hinter einen Proxy stellst. Dozzle kann die passenden Header ohne weitere Einrichtung lesen.

Wenn du keine Authentifizierungslösung hast, bietet Dozzle eine einfache dateibasierte Benutzerverwaltung. Authentifizierungsanbieter werden über das Flag `--auth-provider` eingerichtet. In beiden Konfigurationen versucht Dozzle, die Benutzereinstellungen auf die Festplatte zu schreiben. Diese Daten landen in `/data`.

## <Icon icon="mdi:shield-alert-outline" inline /> Sicherheitshinweise

Dozzle hat Zugriff auf `docker.sock`, was — sofern nicht eingeschränkt — **Root-Rechten auf dem Host** entspricht. Bevor du Dozzle über dein privates Netzwerk hinaus erreichbar machst, geh Folgendes durch:

- **Stelle Dozzle immer hinter eine Authentifizierung**, wenn es aus dem öffentlichen Internet erreichbar ist. Nutze `--auth-provider=simple` oder einen Forward-Proxy wie Authelia / Authentik / Cloudflare Access.
- **Lass [Aktionen](/de/guide/actions) und [Shell-Zugriff](/de/guide/shell) deaktiviert**, solange du sie nicht brauchst. Damit lassen sich Container starten, stoppen, neu erstellen und beliebige Befehle darin ausführen.
- **Schränke Benutzer im Mehrbenutzermodus über [Rollen](#setting-specific-roles-for-users) und [Filter](#setting-specific-filters-for-users) ein.** Ohne explizite Rollen sieht ein Benutzer jeden Container, den die Dozzle-Instanz sieht.
- **Gib den Port von Dozzle im Forward-Proxy-Modus niemals direkt frei.** Dozzle vertraut `Remote-User` bei jeder Anfrage, und wenn kein Rollen-Header vorhanden ist, bekommt der Benutzer alle Rollen. Wer den Container erreicht, ohne über den Proxy zu gehen, authentifiziert sich mit einem einzigen Header als beliebiger Benutzer. Veröffentliche nur den Proxy und halte Dozzle mit `expose` statt `ports` in einem internen Netzwerk.
- **Terminiere TLS am Reverse Proxy**. Beispiele für Nginx / Traefik / Caddy findest du unter [Reverse Proxy & Basispfad](/de/guide/changing-base).
- **Schränke den Zugriff auf `docker.sock` mit einem Proxy ein**, wenn du keine Aktionen brauchst. Beachte, dass ein schreibgeschützter Mount (`/var/run/docker.sock:/var/run/docker.sock:ro`) die API _nicht_ einschränkt: Das Flag `:ro` markiert nur die Socket-Datei auf der Festplatte als schreibgeschützt, API-Aufrufe laufen weiterhin ganz normal über den Socket, sodass Erstellen, Löschen und Ändern weiterhin möglich sind. Um Operationen wirklich einzuschränken, setze einen Socket-Proxy wie [`tecnativa/docker-socket-proxy`](https://github.com/Tecnativa/docker-socket-proxy) vor den Daemon.

## <Icon icon="mdi:account-cog-outline" inline /> Dateibasierte Benutzerverwaltung

Dozzle unterstützt Authentifizierung mit mehreren Benutzern, wenn `--auth-provider` auf `simple` gesetzt ist. In diesem Modus versucht Dozzle, die Benutzerdatei aus `/data/` zu lesen, wobei `users.yml` Vorrang vor `users.yaml` hat, falls beide Dateien vorhanden sind. Existiert nur eine der Dateien, wird diese verwendet. Im Log steht, welche Datei gelesen wird (z. B. `Reading users.yml file`).

### Beispiele für Dateipfade:

- `/data/users.yml`
- `/data/users.yaml`

Der Inhalt der Datei sieht so aus:

```yaml
users:
  # "admin" ist hier der Benutzername
  admin:
    email: me@email.net
    name: Admin
    # Mit docker run -it --rm amir20/dozzle generate admin --password password --email me@email.net --name "Admin" erzeugen
    password: $2a$11$9ho4vY2LdJ/WBopFcsAS0uORC0x2vuFHQgT/yBqZyzclhHsoaIkzK
    filter:
    roles:
```

Dozzle nutzt `email`, um Avatare über [Gravatar](https://gravatar.com/) zu erzeugen. Das Feld ist optional. Das Passwort wird mit `bcrypt` gehasht und kann mit `docker run amir20/dozzle generate` erzeugt werden.

> [!WARNING]
> Passwort-Hashes mit SHA-256 werden nicht mehr unterstützt. Ältere Dozzle-Versionen haben Passwörter mit SHA-256 gehasht, und eine `users.yml`, die noch so einen Hash enthält, wird kommentarlos geladen, doch der Prozess beendet sich, sobald sich dieser Benutzer anmelden will. Erzeuge vor dem Upgrade jedes Passwort mit `generate` neu. Mehr Details findest du in [diesem Advisory](https://github.com/amir20/dozzle/security/advisories/GHSA-w7qr-q9fh-fj35).

Du musst diese Datei einhängen, damit Dozzle sie findet. Hier ein Beispiel:

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

Oder mit Docker Secrets:

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

### Lebensdauer des Authentifizierungs-Cookies verlängern

Standardmäßig verwendet Dozzle Session-Cookies, die beim Schließen des Browsers ablaufen. Du kannst die Lebensdauer des Cookies verlängern, indem du `--auth-ttl` auf eine Dauer setzt. Hier ein Beispiel:

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

Beachte, dass nur eine Dauer unterstützt wird. Du kannst nur `s`, `m` und `h` für Sekunden, Minuten und Stunden verwenden.

### Bestimmte Filter für Benutzer setzen

Dozzle unterstützt Filter für einzelne Benutzer. Filter schränken ein, welche Container ein Benutzer sehen kann. Sie werden in der Datei `users.yml` gesetzt. Hier ein Beispiel:

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

In diesem Beispiel hat der Benutzer `admin` keinen Filter und sieht damit alle Container. Der Benutzer `guest` sieht nur Container mit dem Label `com.example.app`. Das ist nützlich, um den Zugriff auf bestimmte Container einzuschränken.

> [!NOTE]
> Filter können auch [global](/de/guide/filters) mit dem Flag `--filter` gesetzt werden. Dieses Flag gilt für alle Benutzer. Hat ein Benutzer einen eigenen Filter, überschreibt dieser den globalen Filter.

### Bestimmte Rollen für Benutzer setzen

Dozzle erlaubt es, Benutzern Rollen zuzuweisen. Rollen legen fest, welche Aktionen ein Benutzer an Containern ausführen darf. Rollen werden in der Datei users.yml konfiguriert.

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

In diesem Beispiel hat der Benutzer `admin` keine Rollen angegeben und damit vollen Zugriff auf alle Container-Aktionen. Der Benutzer `guest` hat die Rolle shell und kann damit nur eine Shell in den Containern öffnen. Rollen machen es einfach, zu steuern und einzuschränken, was Benutzer in Dozzle tun dürfen.

Dozzle unterstützt die folgenden Rollen:

| Rolle           | Ebenfalls akzeptiert   | Erlaubt                                                                                                    |
| --------------- | ---------------------- | ---------------------------------------------------------------------------------------------------------- |
| `shell`         | `dozzle_shell`         | An einen Container andocken und eine Exec-Sitzung öffnen. Die Instanz braucht zusätzlich `--enable-shell`. |
| `actions`       | `dozzle_actions`       | Container starten, stoppen und neu starten. Die Instanz braucht zusätzlich `--enable-actions`.             |
| `download`      | `dozzle_download`      | Container-Logs als Datei herunterladen.                                                                    |
| `notifications` | `dozzle_notifications` | Benachrichtigungsregeln und Ziele erstellen und bearbeiten.                                                |
| `cloud`         | `dozzle_cloud`         | Dozzle Cloud verknüpfen, trennen und konfigurieren.                                                        |
| `all`           | `dozzle_all`           | Alle Rollen oben. Das ist die Voreinstellung, wenn `roles` leer ist.                                       |
| `none`          | `dozzle_none`          | Keine Rollen. Logs bleiben sichtbar, im Rahmen des Benutzerfilters. Überschreibt alles andere.             |

Rollen werden durch Kommas oder Pipes getrennt (`shell,actions` oder `shell|actions`), ein JSON-Array funktioniert ebenfalls (`["shell", "actions"]`). Groß- und Kleinschreibung spielt keine Rolle. Die Aliase mit dem Präfix `dozzle_` gibt es, damit Gruppennamen aus einem Identity Provider im Forward-Proxy-Modus unverändert durchgereicht werden können.

> [!WARNING]
> Benachrichtigungsregeln gelten für die gesamte Instanz. Eine Regel wählt Container über einen Ausdruck aus, nicht über den Filter des Benutzers. Ein Benutzer mit der Rolle `notifications` kann also eine Regel für Container anlegen, die sein Filter sonst verbirgt, und diese Logzeilen an ein Ziel schicken, das er selbst kontrolliert. Vergib sie nur an Benutzer, denen du jeden Container der Instanz anvertraust.

> [!WARNING]
> Dozzle Cloud gilt ebenfalls für die gesamte Instanz. Beim Verknüpfen wird ein einzelner API-Schlüssel gespeichert, der Alarmversand, Log-Streaming und Tool-Ausführung auf ein Cloud-Konto umleitet, und Cloud-Tools laufen mit dem Filter der Instanz statt mit dem des verknüpfenden Benutzers. Ein Benutzer mit der Rolle `cloud` kann die Instanz mit seinem eigenen Cloud-Konto verknüpfen und darüber jeden Container sehen oder eine bestehende Verbindung trennen. Vergib sie nur an Benutzer, denen du jeden Container der Instanz anvertraust.

Jeder Rolle kann ein `^` vorangestellt werden, um sie auszuschließen. Ausschlüsse werden zuletzt angewendet, die Reihenfolge spielt also keine Rolle:

```yaml
roles: all,^shell # alles außer shell
```

`none` ist die einzige Rolle, die nicht negiert werden kann. `^none` wird ignoriert, und ein einfaches `none` an beliebiger Stelle in der Liste entfernt alle anderen Rollen.

## <Icon icon="mdi:file-document-edit-outline" inline /> users.yml erzeugen

Dozzle hat einen eingebauten Befehl `generate`, um `users.yml` zu erzeugen. Hier ein Beispiel:

```sh
docker run -it --rm amir20/dozzle generate admin --password password --email test@email.net --name "John Doe" --user-filter name=foo --user-roles shell > users.yml
```

In diesem Beispiel ist `admin` der Benutzername. E-Mail und Name sind optional, aber empfohlen, damit die Avatare stimmen. `docker run -it --rm amir20/dozzle generate --help` zeigt alle Optionen. Das Flag `--user-filter` erwartet eine kommagetrennte Liste von Filtern. Das Flag `--user-roles` erwartet eine kommagetrennte Liste von Rollen.

Wenn du `--password` weglässt, fragt Dozzle das Passwort auf stdin ab, sodass es nie in deiner Shell-History landet. Dafür ist ein interaktives Terminal nötig, behalte also die Flags `-it`:

```sh
docker run -it --rm amir20/dozzle generate admin --email test@email.net --name "John Doe" > users.yml
```

Die Eingabeaufforderung wird nach stderr geschrieben, die Umleitung von stdout nach `users.yml` funktioniert also weiterhin. Du kannst das Passwort auch hineinpipen, zum Beispiel `echo "$PASSWORD" | docker run -i --rm amir20/dozzle generate admin > users.yml`.

## <Icon icon="mdi:swap-horizontal" inline /> Forward Proxy

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

### Dozzle mit Authelia einrichten

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

Authelia schickt die Gruppenzugehörigkeit in `Remote-Groups`, und Dozzle liest diesen Header nicht standardmäßig. Um Authelia-Gruppen auf Dozzle-[Rollen](#setting-specific-roles-for-users) abzubilden, setze `DOZZLE_AUTH_HEADER_ROLES: Remote-Groups` beim Dozzle-Service und benenne die Gruppen nach den Rollen. Genau dafür gibt es die Aliase mit dem Präfix `dozzle_`: Eine Gruppe namens `dozzle_shell` gewährt die Rolle `shell`, andere Gruppennamen werden ignoriert. Ohne diese Zuordnung bekommt jeder authentifizierte Benutzer alle Rollen.

</details>

### Dozzle mit Cloudflare Zero Trust einrichten

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

### Dozzle mit Pocket ID einrichten

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
