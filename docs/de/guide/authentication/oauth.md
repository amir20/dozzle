---
title: Mit GitHub & OIDC anmelden
sourceHash: eb2ef5b9def7
---

# <Icon icon="mdi:shield-account" inline /> Mit GitHub & OIDC anmelden

Dozzle kann Benutzer sich mit einem externen Konto anmelden lassen, statt ein Passwort einzutippen. Das gehört zum Anbieter [`simple`](/de/guide/authentication/simple) und ist kein eigener Anbieter, `users.yml` wird also weiterhin bei jeder Anfrage gelesen und entscheidet weiterhin, wer hereinkommt.

Das hat eine Konsequenz, die gleich vorweg gesagt sein will: **`users.yml` ist die Zugriffsliste.** Ein externes Konto, mit dem kein Eintrag verknüpft ist, kann sich nicht anmelden, und es wird nie automatisch ein Konto angelegt.

Die Anmeldung mit Passwort funktioniert weiterhin daneben, was wichtig ist, wenn eine OAuth-App kaputtgeht und du hereinkommen musst, um sie zu reparieren.

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
$ docker run -v /var/run/docker.sock:/var/run/docker.sock -v /path/to/dozzle/data:/data -p 8080:8080 amir20/dozzle --auth-provider simple --auth-github-client-id Iv1.0123456789abcdef --auth-github-client-secret 0123456789abcdef0123456789abcdef01234567
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
      DOZZLE_AUTH_GITHUB_CLIENT_ID: Iv1.0123456789abcdef
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

Der Wert ist der GitHub-**Login** (das Kürzel in `github.com/octocat`), nicht die E-Mail-Adresse. Logins sind stabil und immer sichtbar, während die E-Mail-Adresse eines Kontos privat sein oder sich jederzeit ändern kann.

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

### Google

Google ist ein ganz normaler OIDC-Anbieter. Lege in der Google Cloud Console einen OAuth-Client an und verwende:

```
DOZZLE_AUTH_OIDC_ISSUER: https://accounts.google.com
DOZZLE_AUTH_OIDC_NAME: Google
```

`--auth-provider google` wird als Alias für `simple` akzeptiert, beide Schreibweisen funktionieren also.

## Docker Secrets für das Client Secret verwenden

Ein Client Secret direkt unter `environment:` einzutragen bedeutet, dass es in `docker inspect`, in deiner Compose-Datei und in der Shell-History von allen auftaucht, die den Container von Hand gestartet haben. Beide Client Secrets akzeptieren deshalb ein Gegenstück mit `_FILE`, das eine Datei benennt, aus der der Wert gelesen wird. Das ist dieselbe Konvention, die Dockers eigene Images verwenden:

| Statt                              | Nimm                                    |
| ---------------------------------- | --------------------------------------- |
| `DOZZLE_AUTH_GITHUB_CLIENT_SECRET` | `DOZZLE_AUTH_GITHUB_CLIENT_SECRET_FILE` |
| `DOZZLE_AUTH_OIDC_CLIENT_SECRET`   | `DOZZLE_AUTH_OIDC_CLIENT_SECRET_FILE`   |

Dozzle liest die Datei beim Start und entfernt umgebende Leerzeichen, ein abschließender Zeilenumbruch von `echo secret > file` ist also kein Problem. Eine Variable und ihr `_FILE`-Gegenstück gleichzeitig zu setzen ist ein Fehler und keine stille Bevorzugung einer von beiden, und wenn `_FILE` auf eine fehlende oder leere Datei zeigt, stoppt Dozzle beim Start, statt den Anmeldebutton klammheimlich abzuschalten.

### Docker Compose

Ein vollständiges Beispiel. `users.yml` wird ebenfalls als Secret eingehängt, damit nichts Sensibles in der Compose-Datei steht:

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
      DOZZLE_AUTH_GITHUB_CLIENT_ID: Iv1.0123456789abcdef
      DOZZLE_AUTH_GITHUB_CLIENT_SECRET_FILE: /run/secrets/dozzle_github_secret
    secrets:
      - source: dozzle_github_secret
      - source: dozzle_users
        target: /data/users.yml

secrets:
  dozzle_github_secret:
    file: ./secrets/github_client_secret.txt
  dozzle_users:
    file: ./secrets/users.yml
```

Lege zuerst die Secret-Datei an, ohne einen abschließenden Zeilenumbruch in deiner Shell-History:

```sh
mkdir -p secrets
printf '%s' 'your-github-client-secret' > secrets/github_client_secret.txt
chmod 600 secrets/github_client_secret.txt
```

### Docker Swarm

Im Swarm wird das Secret vom Cluster verwaltet und liegt nicht als Datei auf der Platte. Deklariere es deshalb als `external` und lege es mit `docker secret create` an:

```sh
printf '%s' 'your-oidc-client-secret' | docker secret create dozzle_oidc_secret -
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
```

> [!NOTE]
> Ein Secret wird standardmäßig unter `/run/secrets/<name>` eingehängt, deshalb zeigt `_FILE` dorthin. Wenn du ein explizites `target:` setzt, lass `_FILE` stattdessen auf diesen Pfad zeigen.

### Überprüfen, ob es funktioniert hat

Dozzle protokolliert beim Start, welche Anbieter es aktiviert hat. Starte mit `--level debug` und such nach der Zeile, die den Anbieter nennt:

```sh
$ docker compose logs dozzle | grep -i 'sign in'
DBG Enabling Sign in with GitHub
```

Wenn die Secret-Datei fehlt oder leer ist, beendet sich Dozzle beim Start mit einer Meldung, die die Variable nennt. Eine kaputte Einbindung schlägt also laut fehl, statt den Anmeldebutton stillschweigend verschwinden zu lassen.
