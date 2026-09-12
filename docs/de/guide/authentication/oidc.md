---
title: OpenID Connect
sourceHash: 9dcc3df4a715
---

# <Icon icon="mdi:shield-account" inline /> OpenID Connect

Mit `--auth-provider oidc` ist dein Identity Provider die Benutzerdatenbank. Dozzle liest `users.yml` in diesem Modus nie: Wer ein Benutzer ist, welche Rollen er hat und welche Container er sehen darf, kommt alles aus dem OpenID-Connect-Token. Lege in Keycloak, Authentik, Zitadel oder Pocket ID einen Benutzer an und er kann sich anmelden; entferne seine Rolle und er kann es nicht mehr.

Das ist etwas anderes als die [Anmeldung von `users.yml`-Benutzern über OIDC](/de/guide/authentication/oauth#mit-oidc-anmelden) beim Anbieter `simple`. Dort weist der Anbieter nur nach, wer du bist, und `users.yml` entscheidet weiterhin, was du bekommst. Nimm `simple`, wenn du jeden Benutzer von Hand auflisten willst, und `oidc`, wenn der Anbieter die Liste besitzen soll.

## Minimale Konfiguration

Registriere Dozzle bei deinem Anbieter als vertraulichen Client und setze die Redirect-URI auf:

```
https://your-dozzle-host/api/auth/callback
```

Nimm den Basispfad mit auf, falls Dozzle unter einem läuft, zum Beispiel `https://example.com/dozzle/api/auth/callback`. Zeig Dozzle dann den Issuer:

::: code-group

```sh [cli]
$ docker run -v /var/run/docker.sock:/var/run/docker.sock -v /path/to/dozzle/data:/data -p 8080:8080 amir20/dozzle --auth-provider oidc --auth-oidc-issuer https://keycloak.example.com/realms/main --auth-oidc-client-id dozzle --auth-oidc-client-secret secret
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
      DOZZLE_AUTH_PROVIDER: oidc
      DOZZLE_AUTH_OIDC_ISSUER: https://keycloak.example.com/realms/main
      DOZZLE_AUTH_OIDC_CLIENT_ID: dozzle
      DOZZLE_AUTH_OIDC_CLIENT_SECRET: secret
```

:::

Das ist schon alles. Für die gängigen Layouts müssen keine Claim-Pfade gesetzt werden, und `DOZZLE_AUTH_OIDC_NAME` ändert nur die Beschriftung des Anmeldebuttons. Das Client Secret akzeptiert ebenfalls ein Gegenstück mit `_FILE`, siehe [Docker Secrets verwenden](/de/guide/authentication/oauth#docker-secrets-fur-das-client-secret-verwenden).

Die Issuer-URL ist die, unter der `/.well-known/openid-configuration` ausgeliefert wird. Dozzle holt sich dieses Dokument, um die Endpunkte für Authorization, Token und Userinfo zu finden, und startet den Ablauf gar nicht erst, wenn das Dokument einen anderen Issuer nennt als den, den du konfiguriert hast.

## Rollen

Die Rollen eines Benutzers werden aus dem Token gelesen. Dozzle probiert diese Claims der Reihe nach und nimmt den ersten, der vorhanden ist:

1. `dozzle_roles`
2. `resource_access.<client-id>.roles`
3. `roles`

Die Client-ID ist bereits konfiguriert, mit `DOZZLE_AUTH_OIDC_CLIENT_ID=dozzle` lautet der zweite Pfad also `resource_access.dozzle.roles`, und genau dort legt Keycloak Client-Rollen ab. Für dieses Layout muss nichts weiter gesetzt werden.

Liegen deine Rollen woanders, ersetzt `DOZZLE_AUTH_OIDC_ROLES_CLAIM` die Suche durch den einen punktgetrennten Pfad, den du angibst:

```yaml
DOZZLE_AUTH_OIDC_ROLES_CLAIM: realm_access.roles
```

Jeder Claim wird zuerst im ID-Token und dann in der Userinfo-Antwort gesucht, es spielt also keine Rolle, in welchem von beiden dein Anbieter ihn unterbringt. Drei Formen werden akzeptiert: ein Array aus Strings, ein einzelner durch Kommas oder Leerzeichen getrennter String und ein Objekt, dessen Schlüssel die Rollen sind, denn so kodiert Zitadel Projektrollen.

Die Rollennamen sind dieselben wie in `users.yml`: `shell`, `actions`, `download`, `notifications`, `cloud` und `all`, mit `^` zum Ausschließen, `all,^shell` gewährt also alles außer Shell-Zugriff. Namen mit dem Präfix `dozzle_` werden ebenfalls akzeptiert, was hilft, wenn der Anbieter einen Rollen-Claim für mehrere Anwendungen gemeinsam nutzt. Unter [Rollen](/de/guide/authentication/simple#bestimmte-rollen-fur-benutzer-setzen) steht, was jede einzelne freischaltet.

> [!WARNING]
> `groups` steht bewusst nicht auf der Liste. Bei Authentik oder Google gehört jeder Benutzer zu mindestens einer Gruppe, eine Suche dort würde also aus „abgelehnt“ ein „angemeldet und darf jeden Container lesen“ machen. Wenn Gruppen das sind, was du hast, bilde sie beim Anbieter auf einen Claim `dozzle_roles` ab, siehe die Beispiele unten.

### Wenn eine Anmeldung abgelehnt wird

Die Anmeldung wird abgelehnt, wenn keiner der Claims vorhanden ist oder wenn der erste vorhandene leer ist. Das Log nennt die Pfade, die probiert wurden:

```
WRN OIDC login rejected: no roles claim found in the ID token or userinfo, or it was empty sub=... tried="dozzle_roles, resource_access.dozzle.roles, roles"
```

Ein Claim, der vorhanden ist, aber nichts enthält, was Dozzle als Rolle erkennt, ist etwas anderes. Dieser Benutzer meldet sich ohne Rechte an, genau wie mit `roles: none` in `users.yml`: Er kann die Logs der Container lesen, die sein Filter erlaubt, und sonst nichts. Keycloak-Realm-Rollen verhalten sich so, weil dort jeder Benutzer `offline_access` und `uma_authorization` trägt, weshalb die Beispiele unten stattdessen Client-Rollen verwenden.

## Filter

Container-Filter funktionieren genauso, gelesen aus dem ersten von `dozzle_filters`, `resource_access.<client-id>.filters` und `filters`, der vorhanden ist, oder aus dem einen Pfad in `DOZZLE_AUTH_OIDC_FILTERS_CLAIM`. Jeder Wert ist ein Filter in [derselben Syntax wie in `users.yml`](/de/guide/authentication/simple#bestimmte-filter-fur-benutzer-setzen), zum Beispiel `label=com.example.app` oder `name=web`:

```json
"resource_access": {
  "dozzle": {
    "roles": ["shell", "actions"],
    "filters": ["label=com.example.app"]
  }
}
```

Ein Benutzer ohne Filter-Claim sieht jeden Container, den die Dozzle-Instanz sieht. Ein Filter, der sich nicht parsen lässt, lässt die Anmeldung fehlschlagen, statt verworfen zu werden, ein Tippfehler beim Anbieter kann also nicht stillschweigend erweitern, was jemand sieht.

## Identität

Der Claim `sub` ist die stabile ID des Benutzers. Er ist der Schlüssel für das Profilverzeichnis unter `/data`, Einstellungen folgen der Person also auch dann, wenn sich ihr Benutzername oder ihre E-Mail-Adresse beim Anbieter ändert. Der im Menü angezeigte Name ist `name`, mit Rückfall auf `preferred_username`, dann `email`, dann `sub`. `email` und `picture` speisen den Avatar, und eine `picture`-URL wird direkt verwendet, wenn der Anbieter eine schickt.

Anders als beim Anbieter `simple` muss die E-Mail-Adresse hier nicht verifiziert sein. Sie wird nur angezeigt und nie mit etwas abgeglichen, ein Issuer, der keinen E-Mail-Scope gewährt, funktioniert also problemlos.

## Sitzungen

Nach der Anmeldung stellt Dozzle sein eigenes Sitzungs-Cookie aus, das die Rollen und Filter trägt, die es aus dem Token gelesen hat. Sie werden bei jeder Anfrage angewendet, aber nicht erneut abgerufen: Eine Rollenänderung beim Anbieter wirkt erst bei der nächsten Anmeldung des Benutzers. Wenn diese Lücke stört, setze [`--auth-ttl`](/de/guide/supported-env-vars) auf etwas wie `8h`, damit Sitzungen ablaufen und mit einem frischen Token neu aufgebaut werden.

## Abmelden

Beim Abmelden wird Dozzles Sitzung gelöscht. Ist `--auth-logout-url` gesetzt, wird der Browser anschließend dorthin geschickt. Zeig damit auf die End-Session-URL deines Anbieters, um den Benutzer auch beim Anbieter abzumelden:

```yaml
DOZZLE_AUTH_LOGOUT_URL: https://keycloak.example.com/realms/main/protocol/openid-connect/logout
```

## Was sich von `simple` unterscheidet

Beide Anbieter teilen sich die Flags `--auth-oidc-*`, der Unterschied zeigt sich also im Verhalten:

- `users.yml` wird nie gelesen. Liegt eine unter `/data`, protokolliert Dozzle, dass es sie ignoriert.
- Es gibt kein Passwortformular und keinen Endpunkt `/api/token`. Der Identity Provider ist der einzige Weg hinein, eine falsche Callback-URL oder ein abgelaufenes Client Secret sperrt also alle aus, bis es behoben ist.
- `--auth-github-*` ist ein Fehler beim Start. GitHub ist kein OpenID-Connect-Issuer und veröffentlicht keine Claims, aus denen sich Rollen lesen ließen.
- Beim Start werden der Issuer und die Claim-Pfade protokolliert, aus denen die Rollen gelesen werden.

## Beispiele für Anbieter

### Keycloak

Lege in deinem Realm einen Client `dozzle` mit eingeschalteter Client-Authentifizierung an und trage die Redirect-URI von oben ein. Erstelle dann im Tab **Roles** des Clients die Client-Rollen, die du vergeben willst: `shell`, `actions`, `download`, `notifications`, `cloud` oder `all`. Weise sie unter **Role mapping** Benutzern oder Gruppen zu.

Keycloak gibt Client-Rollen als `resource_access.<client-id>.roles` aus, wo Dozzle ohnehin sucht. Öffne unter den **Client scopes** des Clients den dedizierten Scope und prüfe, dass der Mapper **client roles** den Claim dem ID-Token oder der Userinfo hinzufügt; Dozzle liest beide, aber nicht das Access-Token.

Für Filter legst du ein Benutzerattribut `dozzle_filters` an sowie im dedizierten Scope einen Mapper **User Attribute** mit demselben Token-Claim-Namen und eingeschaltetem **Multivalued**. Jeder Wert ist ein Filter, etwa `label=com.example.app`.

### Authentik

Füge unter **Customization** → **Property Mappings** ein Scope Mapping hinzu, das die Rollen aus den Gruppen des Benutzers zurückgibt, und hänge es an den Dozzle-Provider:

```python
roles = []
if request.user.ak_groups.filter(name="dozzle-admins").exists():
    roles.append("all")
elif request.user.ak_groups.filter(name="dozzle-users").exists():
    roles.append("download")
return {"dozzle_roles": roles}
```

Ein Benutzer, der in keiner der beiden Gruppen ist, bekommt eine leere Liste und wird abgelehnt.

### Zitadel

Gib Benutzern Projektrollen, die nach den Dozzle-Rollen benannt sind, und aktiviere **Assert Roles on Authentication** an der Anwendung. Zitadels Rollen-Claim ist ein Objekt mit den Rollennamen als Schlüssel, was Dozzle akzeptiert, setze also:

```yaml
DOZZLE_AUTH_OIDC_ROLES_CLAIM: urn:zitadel:iam:org:project:roles
```

### Google

Googles Tokens enthalten keinen Rollen-Claim, `oidc` lässt sich also nicht damit verwenden. Nimm stattdessen den [Anbieter `simple` mit Google-Anmeldung](/de/guide/authentication/oauth#google), bei dem `users.yml` entscheidet, wer hereinkommt.
