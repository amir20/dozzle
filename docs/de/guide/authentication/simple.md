---
title: Einfache Authentifizierung
sourceHash: 50dffd6cac64
---

# <Icon icon="mdi:account-cog-outline" inline /> Einfache Authentifizierung

Dozzles eigene Benutzerverwaltung. Die Benutzer stehen in einer Datei `users.yml`, die Dozzle selbst verwaltet, und Dozzle liefert seine eigene Anmeldeseite aus. Setze `--auth-provider` auf `simple`, um sie einzuschalten.

Passwörter sind ein Weg, um nachzuweisen, dass du einer dieser Benutzer bist. [Mit GitHub oder OIDC anmelden](/de/guide/authentication/oauth) ist der andere, und beide lesen dieselbe `users.yml`.

> [!TIP]
> Nutze den eingebauten [Befehl `generate`](/de/guide/authentication#users-yml-erzeugen), um `users.yml` zu erstellen, statt bcrypt-Hashes von Hand zu schreiben.

Dozzle unterstützt Authentifizierung mit mehreren Benutzern, wenn `--auth-provider` auf `simple` gesetzt ist. In diesem Modus versucht Dozzle, die Benutzerdatei aus `/data/` zu lesen, wobei `users.yml` Vorrang vor `users.yaml` hat, falls beide Dateien vorhanden sind. Existiert nur eine der Dateien, wird diese verwendet. Im Log steht, welche Datei gelesen wird (z. B. `Reading users.yml file`).

## Beispiele für Dateipfade:

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

## Lebensdauer des Authentifizierungs-Cookies verlängern

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

## Bestimmte Filter für Benutzer setzen

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

## Bestimmte Rollen für Benutzer setzen

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
> Dozzle Cloud gilt ebenfalls für die gesamte Instanz. Beim Verknüpfen wird ein einzelner API-Schlüssel gespeichert, der Alarmversand, Log-Streaming und Tool-Ausführung auf ein Cloud-Konto umleitet. Ein Benutzer mit der Rolle `cloud` kann die Instanz mit seinem eigenen Cloud-Konto verknüpfen und darüber jeden Container sehen oder eine bestehende Verbindung trennen. Vergib sie nur an Benutzer, denen du jeden Container der Instanz anvertraust.
>
> Die Rolle regelt das Verknüpfen, nicht das Lesen. Jeder angemeldete Benutzer kann Cloud-Logs durchsuchen und Cloud-Alarme sehen, begrenzt auf den eigenen Filter. Tool-Aufrufe, die die Cloud von sich aus startet, etwa eine Frage in Telegram oder Discord, laufen dagegen mit dem Filter der Instanz, weil kein Dozzle-Benutzer dahintersteht.

Jeder Rolle kann ein `^` vorangestellt werden, um sie auszuschließen. Ausschlüsse werden zuletzt angewendet, die Reihenfolge spielt also keine Rolle:

```yaml
roles: all,^shell # alles außer shell
```

`none` ist die einzige Rolle, die nicht negiert werden kann. `^none` wird ignoriert, und ein einfaches `none` an beliebiger Stelle in der Liste entfernt alle anderen Rollen.
