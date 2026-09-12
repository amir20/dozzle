---
title: Mit GitHub & OIDC anmelden
sourceHash: cfb7126acb65
---

# <Icon icon="mdi:shield-account" inline /> Mit GitHub & OIDC anmelden

Dozzle kann Benutzer sich mit einem externen Konto anmelden lassen, statt ein Passwort einzutippen. Das gehört zum Anbieter [`simple`](/de/guide/authentication/simple) und ist kein eigener Anbieter, `users.yml` wird also weiterhin bei jeder Anfrage gelesen und entscheidet weiterhin, wer hereinkommt.

Das hat eine Konsequenz, die gleich vorweg gesagt sein will: **`users.yml` ist die Zugriffsliste.** Ein externes Konto, mit dem kein Eintrag verknüpft ist, kann sich nicht anmelden, und es wird nie automatisch ein Konto angelegt.

Die Anmeldung mit Passwort funktioniert weiterhin daneben, was wichtig ist, wenn eine OAuth-App kaputtgeht und du hereinkommen musst, um sie zu reparieren.

> [!TIP]
> Wenn du lieber den Identity Provider die Benutzerliste besitzen lässt, mit Rollen und Filtern aus dem Token und ganz ohne `users.yml`, ist das ein eigener Anbieter: siehe [OpenID Connect](/de/guide/authentication/oidc).

## Mit GitHub anmelden

Dozzle kann Benutzer sich mit ihrem GitHub-Konto anmelden lassen, statt ein Passwort einzutippen. Das gehört zum Anbieter `simple` und ist kein eigener Authentifizierungsanbieter, `users.yml` wird also weiterhin bei jeder Anfrage gelesen und entscheidet weiterhin, wer hereinkommt. Bleib bei `--auth-provider simple`. `github` wird als Alias akzeptiert, falls du lieber ausschreiben willst, was die Instanz verwendet.

Lege zuerst unter [Developer settings](https://github.com/settings/developers) auf GitHub eine OAuth App an und setze die **Authorization callback URL** auf:

```
https://your-dozzle-host/api/auth/callback
```

Wenn Dozzle unter einem [Basispfad](/de/guide/changing-base) ausgeliefert wird, nimm ihn mit auf, zum Beispiel `https://example.com/dozzle/api/auth/callback`. Dozzle schickt beim Start des Ablaufs kein `redirect_uri` mit, GitHub leitet also immer auf die Callback-URL um, die in der OAuth App hinterlegt ist. Eine Abweichung hier ist der häufigste Grund, warum die Anmeldung fehlschlägt.

Kopiere dann die Client-ID, erzeuge ein Client Secret und gib Dozzle beides mit:

::: code-group

```sh [cli]
$ docker run -v /var/run/docker.sock:/var/run/docker.sock -v /path/to/dozzle/data:/data -p 8080:8080 amir20/dozzle --auth-provider simple --auth-github-client-id Ov23liABCDEFGHIJKLMN --auth-github-client-secret 0123456789abcdef0123456789abcdef01234567
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
      DOZZLE_AUTH_GITHUB_CLIENT_ID: Ov23liABCDEFGHIJKLMN
      DOZZLE_AUTH_GITHUB_CLIENT_SECRET: 0123456789abcdef0123456789abcdef01234567
```

:::

Verknüpfe einen Benutzer über den Schlüssel `github` in `users.yml` mit seinem GitHub-Konto:

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

`password` ist optional, sobald `github` gesetzt ist, wie `guest` oben zeigt. `admin` hat beides und kann sich daher auf beiden Wegen anmelden. Die Anmeldung mit Passwort bleibt als Rückfalloption für alle erhalten, die noch ein Passwort haben, und die Anmeldeseite zeigt beide Möglichkeiten.

> [!WARNING]
> Behalte bei mindestens einem Konto ein Passwort. Wenn kein Benutzer in `users.yml` ein `password` hat, verschwindet das Anmeldeformular vollständig und der externe Anbieter ist der einzige Weg hinein. Eine falsche Callback-URL, eine widerrufene OAuth-App oder ein abgelaufenes Client Secret sperrt dann alle aus der Weboberfläche aus. Zur Wiederherstellung musst du `users.yml` auf dem Host bearbeiten und dort wieder ein Passwort eintragen, wofür du Shell-Zugriff dorthin brauchst, wo Dozzles `/data` liegt.

Der Wert ist der GitHub-**Login** (das Kürzel in `github.com/octocat`), nicht die E-Mail-Adresse. Ein Login ist immer vorhanden und sichtbar, während die E-Mail-Adresse eines Kontos privat sein oder sich jederzeit ändern kann.

> [!WARNING]
> Ein GitHub-Login ist nicht dauerhaft. Wenn jemand sein GitHub-Konto umbenennt, wird das alte Kürzel freigegeben und kann von beliebigen Personen neu registriert werden. Wer das tut, übernimmt bei der nächsten Anmeldung den zugehörigen Eintrag in deiner `users.yml`. Behandle eine Umbenennung als Änderung der Zugriffsrechte: Passe `users.yml` zur selben Zeit an und entferne Einträge von Personen, die nicht mehr dabei sind, statt ein veraltetes Kürzel stehen zu lassen.

`users.yml` ist die Zugriffsliste. Ein GitHub-Konto, das nicht in `users.yml` steht, kann sich nicht anmelden, egal zu welcher Organisation es gehört. Es gibt kein automatisches Anlegen von Benutzern: Jemanden hinzuzufügen heißt, ihn in die Datei einzutragen. Filter und Rollen werden bei jeder Anfrage aus `users.yml` aufgelöst, genau wie bei Benutzern mit Passwort. Ein GitHub-Benutzer mit `roles: none` ist also genauso eingeschränkt.

> [!NOTE]
> Dozzle unterstützt bewusst nicht, eine ganze GitHub-Organisation oder eine komplette E-Mail-Domain freizugeben. Jeder Benutzer wird einzeln eingetragen. Wenn du Zugriff auf Gruppen- oder Domain-Basis brauchst, nutze `forward-proxy` mit [Authelia](/de/guide/authentication/forward-proxy#dozzle-mit-authelia-einrichten) oder Authentik, die genau dafür gemacht sind.

> [!WARNING]
> Beim Bearbeiten von `users.yml` wird der JWT-Signaturschlüssel neu erzeugt und alle Benutzer werden abgemeldet. Das ist schon heute so, wenn du einen Benutzer hinzufügst oder entfernst, und gilt genauso, wenn du einen `github`-Schlüssel ergänzt.

## Mit OIDC anmelden

Jeder Anbieter, der ein OpenID-Connect-Discovery-Dokument veröffentlicht, funktioniert über denselben Callback: Google, Keycloak, Pocket ID, Zitadel, Authentik und weitere. Zeig Dozzle die Issuer-URL und gib ihm eine Client-ID und ein Secret mit.

Registriere Dozzle bei deinem Anbieter als vertraulichen Client und setze die Redirect-URI auf:

```
https://your-dozzle-host/api/auth/callback
```

Nimm den Basispfad mit auf, falls Dozzle unter einem läuft, zum Beispiel `https://example.com/dozzle/api/auth/callback`. Anders als bei GitHub muss Dozzle bei OIDC ein `redirect_uri` mitschicken, dieser Wert muss also exakt zu dem passen, was du registriert hast.

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

`DOZZLE_AUTH_OIDC_NAME` ist nur die Beschriftung des Anmeldebuttons. Der Standardwert ist `SSO`.

Die Issuer-URL ist die, unter der `/.well-known/openid-configuration` ausgeliefert wird. Dozzle holt sich dieses Dokument, um die Endpunkte für Authorization, Token und Userinfo zu finden, und startet den Ablauf gar nicht erst, wenn das Dokument einen anderen Issuer nennt als den, den du konfiguriert hast.

### Benutzer verknüpfen

OIDC gleicht über die **verifizierte E-Mail-Adresse** ab, nicht über den Login. Setze `email` beim Benutzer in `users.yml`:

```yaml
users:
  admin:
    email: me@email.net
    name: Admin
    # password ist optional, sobald das Konto verknüpft ist
```

Die E-Mail-Adresse muss von deinem Anbieter als verifiziert markiert sein. Dozzle lehnt eine Anmeldung ab, wenn `email_verified` false ist, denn beim Abgleich einer nicht verifizierten Adresse könnte sich jeder, der sich bei einem großzügigen Anbieter registrieren kann, ein Konto aneignen, indem er die E-Mail-Adresse einer anderen Person einträgt.

> [!NOTE]
> GitHub gleicht über den Login ab und OIDC über die E-Mail-Adresse, und dieser Unterschied ist Absicht. Ein GitHub-Login ist stabil und immer vorhanden, während eine GitHub-E-Mail-Adresse privat sein oder geändert werden kann. Bei OIDC gibt es kein stabiles, menschenlesbares Gegenstück, also ist die verifizierte E-Mail-Adresse das Merkmal, das Betreiber tatsächlich kennen.

In diesem Modus weist der Anbieter nur nach, wer du bist. Rollen und Filter kommen weiterhin aus `users.yml`, und ein Benutzer, den die Datei nicht auflistet, kann sich nicht anmelden, egal wie viele Rollen das Token trägt. Damit der Anbieter beides entscheidet, nimm stattdessen [`--auth-provider oidc`](/de/guide/authentication/oidc).

### Google

Google ist ein ganz normaler OIDC-Anbieter. Lege in der Google Cloud Console einen OAuth-Client an und verwende:

```
DOZZLE_AUTH_OIDC_ISSUER: https://accounts.google.com
DOZZLE_AUTH_OIDC_NAME: Google
```

`--auth-provider google` wird als Alias für `simple` akzeptiert, beide Schreibweisen funktionieren also.

## Hinter einem Reverse Proxy

Dozzle entscheidet anhand des Headers `X-Forwarded-Proto`, ob die ursprüngliche Anfrage HTTPS verwendet hat, und nimmt den Hostnamen aus `X-Forwarded-Host`, sofern er vorhanden ist. Die meisten Reverse Proxys setzen beide Header standardmäßig, aber wenn deiner das nicht tut, geht die Anmeldung auf zwei Arten kaputt.

Bei OIDC scheitert die Anmeldung komplett. Das `redirect_uri`, das Dozzle mitschickt, wird aus diesen Headern gebaut. Ein Proxy, der `X-Forwarded-Proto: https` nicht setzt, bringt Dozzle also dazu, `http://your-host/api/auth/callback` zu schicken. Das passt nicht zu der bei deinem Anbieter registrierten `https://`-URI, und der Anbieter lehnt die Anfrage ab, statt irgendwohin Sinnvolles umzuleiten.

Bei GitHub bleibt die URL davon unberührt, weil Dozzle `redirect_uri` weglässt und GitHub auf die in der OAuth App hinterlegte Callback-URL zurückfällt. Der Header entscheidet aber trotzdem, ob das Sitzungs-Cookie als `Secure` markiert wird, es lohnt sich also so oder so, ihn richtig zu setzen.

Was dein Proxy schickt, kannst du an dem Cookie ablesen, das Dozzle beim Start einer Anmeldung setzt:

```sh
$ curl -sI 'https://your-dozzle-host/api/auth/login?provider=github' | grep -i set-cookie
set-cookie: dozzle_oauth_state=...; Path=/; Max-Age=600; HttpOnly; Secure; SameSite=Lax
```

`Secure` in dieser Antwort bedeutet, dass der Header ankommt. Fehlt es, repariere erst den Proxy, bevor du weitermachst. Beispiele für Nginx, Traefik und Caddy findest du unter [Reverse Proxy & Basispfad](/de/guide/changing-base).

## Docker Secrets für das Client Secret verwenden

Ein Client Secret direkt unter `environment:` einzutragen bedeutet, dass es in `docker inspect`, in deiner Compose-Datei und in der Shell-History von allen auftaucht, die den Container von Hand gestartet haben. Beide Client Secrets akzeptieren deshalb ein Gegenstück mit `_FILE`, das eine Datei benennt, aus der der Wert gelesen wird. Das ist dieselbe Konvention, die Dockers eigene Images verwenden:

| Statt                              | Nimm                                    |
| ---------------------------------- | --------------------------------------- |
| `DOZZLE_AUTH_GITHUB_CLIENT_SECRET` | `DOZZLE_AUTH_GITHUB_CLIENT_SECRET_FILE` |
| `DOZZLE_AUTH_OIDC_CLIENT_SECRET`   | `DOZZLE_AUTH_OIDC_CLIENT_SECRET_FILE`   |

Dozzle liest die Datei beim Start und entfernt umgebende Leerzeichen, ein abschließender Zeilenumbruch von `echo secret > file` ist also kein Problem. Eine Variable und ihr `_FILE`-Gegenstück gleichzeitig zu setzen ist ein Fehler und keine stille Bevorzugung einer von beiden, und wenn `_FILE` auf eine fehlende oder leere Datei zeigt, stoppt Dozzle beim Start, statt den Anmeldebutton klammheimlich abzuschalten.

### Docker Compose

Außerhalb von Swarm gibt es kein `docker secret create`, ein Compose-Secret ist also entweder eine Datei auf der Platte oder eine Umgebungsvariable. Die Variante mit der Umgebungsvariable ist meistens die, die du willst: Sie passt zu einer per gitignore ausgeschlossenen `.env`, und es liegt keine Klartextdatei neben deiner Compose-Datei, die nur darauf wartet, eingecheckt zu werden.

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
      DOZZLE_AUTH_GITHUB_CLIENT_ID: Ov23liABCDEFGHIJKLMN
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

Compose liest die Variable selbst und hängt den Wert unter `/run/secrets/dozzle_github_secret` ein. Sie wird nie Teil der Umgebung des Containers und bleibt damit genauso aus `docker inspect` heraus wie ein dateibasiertes Secret.

Nimm stattdessen die Datei-Variante, wenn das Secret bereits als Datei vorliegt, zum Beispiel von einem Secret-Manager geschrieben:

```yaml [docker-compose.yml]
secrets:
  dozzle_github_secret:
    file: /run/secrets/github_client_secret
```

So oder so entfernt Dozzle umgebende Leerzeichen, ein abschließender Zeilenumbruch in der Datei spielt also keine Rolle.

### Docker Swarm

Im Swarm wird das Secret vom Cluster verwaltet und liegt nicht als Datei auf der Platte. Lege es deshalb mit `docker secret create` an und deklariere es als `external`:

```sh
printf '%s' 'your-oidc-client-secret' | docker secret create dozzle_oidc_secret_v1 -
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
    name: dozzle_oidc_secret_v1
```

Beachte die Trennung der beiden Namen. `dozzle_oidc_secret` ist der Alias, den diese Compose-Datei verwendet, und er bestimmt den Einhängepfad: Das Secret landet unter `/run/secrets/dozzle_oidc_secret`, passend zu `_FILE`. `name:` ist das tatsächliche Objekt im Swarm und die einzige Stelle, an der die Version auftaucht.

Diese Trennung gibt es, weil Swarm-Secrets unveränderlich sind. Den Wert eines bestehenden Secrets zu ändern geht nicht, ein geleaktes oder abgelaufenes Client Secret zu rotieren heißt also, die nächste Version anzulegen und den Stack darauf zeigen zu lassen. Wenn die Version nicht im Alias steht, ist das eine einzige Zeile statt drei Änderungen, die über `_FILE`, die `secrets:`-Liste des Service und die Deklaration auf oberster Ebene synchron gehalten werden müssen:

```sh
printf '%s' 'your-new-client-secret' | docker secret create dozzle_oidc_secret_v2 -
```

```yaml [docker-compose.yml]
secrets:
  dozzle_oidc_secret:
    external: true
    name: dozzle_oidc_secret_v2 # was _v1
```

Deploye den Stack neu und entferne danach das alte Secret mit `docker secret rm dozzle_oidc_secret_v1`. Die Umgebungsvariable und der Einhängepfad haben sich nie verschoben.

> [!NOTE]
> Ohne `name:` wird ein Secret unter `/run/secrets/<alias>` eingehängt und der Alias muss dem tatsächlichen Objekt im Swarm entsprechen. Mit `name:` sind die beiden entkoppelt, und genau das macht die Rotation oben zu einer einzigen Änderung. In beiden Fällen zeigt `_FILE` auf den Alias, nie auf `name:`.

### Überprüfen, ob es funktioniert hat

Dozzle protokolliert beim Start, welche Anbieter es aktiviert hat. Starte mit `--level debug` und such nach der Zeile, die den Anbieter nennt:

```sh
$ docker compose logs dozzle | grep -i 'sign in'
DBG Enabling Sign in with GitHub
```

Wenn die Secret-Datei fehlt oder leer ist, beendet sich Dozzle beim Start mit einer Meldung, die die Variable nennt. Eine kaputte Einbindung schlägt also laut fehl, statt den Anmeldebutton stillschweigend verschwinden zu lassen.
