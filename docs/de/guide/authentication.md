---
title: Authentifizierung
sourceHash: c0e3f963afbe
---

# Authentifizierung

Dozzle unterstützt zwei Konfigurationen für die Authentifizierung. In der ersten bringst du deine eigene Authentifizierungsmethode mit, indem du Dozzle hinter einen Proxy stellst. Dozzle kann die passenden Header ohne weitere Einrichtung lesen.

Wenn du keine Authentifizierungslösung hast, bietet Dozzle eine einfache dateibasierte Benutzerverwaltung. Authentifizierungsanbieter werden über das Flag `--auth-provider` eingerichtet. In beiden Konfigurationen versucht Dozzle, die Benutzereinstellungen auf die Festplatte zu schreiben. Diese Daten landen in `/data`.

## <Icon icon="mdi:shield-alert-outline" inline /> Sicherheitshinweise

Dozzle hat Zugriff auf `docker.sock`, was — sofern nicht eingeschränkt — **Root-Rechten auf dem Host** entspricht. Bevor du Dozzle über dein privates Netzwerk hinaus erreichbar machst, geh Folgendes durch:

- **Stelle Dozzle immer hinter eine Authentifizierung**, wenn es aus dem öffentlichen Internet erreichbar ist. Nutze `--auth-provider=simple` oder einen Forward-Proxy wie Authelia / Authentik / Cloudflare Access.
- **Lass [Aktionen](/de/guide/actions) und [Shell-Zugriff](/de/guide/shell) deaktiviert**, solange du sie nicht brauchst. Damit lassen sich Container starten, stoppen, neu erstellen und beliebige Befehle darin ausführen.
- **Schränke Benutzer im Mehrbenutzermodus über [Rollen](/de/guide/authentication/simple#bestimmte-rollen-fur-benutzer-setzen) und [Filter](/de/guide/authentication/simple#bestimmte-filter-fur-benutzer-setzen) ein.** Ohne explizite Rollen sieht ein Benutzer jeden Container, den die Dozzle-Instanz sieht.
- **Gib den Port von Dozzle im Forward-Proxy-Modus niemals direkt frei.** Dozzle vertraut `Remote-User` bei jeder Anfrage, und wenn kein Rollen-Header vorhanden ist, bekommt der Benutzer alle Rollen. Wer den Container erreicht, ohne über den Proxy zu gehen, authentifiziert sich mit einem einzigen Header als beliebiger Benutzer. Veröffentliche nur den Proxy und halte Dozzle mit `expose` statt `ports` in einem internen Netzwerk.
- **Terminiere TLS am Reverse Proxy**. Beispiele für Nginx / Traefik / Caddy findest du unter [Reverse Proxy & Basispfad](/de/guide/changing-base).
- **Schränke den Zugriff auf `docker.sock` mit einem Proxy ein**, wenn du keine Aktionen brauchst. Beachte, dass ein schreibgeschützter Mount (`/var/run/docker.sock:/var/run/docker.sock:ro`) die API _nicht_ einschränkt: Das Flag `:ro` markiert nur die Socket-Datei auf der Festplatte als schreibgeschützt, API-Aufrufe laufen weiterhin ganz normal über den Socket, sodass Erstellen, Löschen und Ändern weiterhin möglich sind. Um Operationen wirklich einzuschränken, setze einen Socket-Proxy wie [`tecnativa/docker-socket-proxy`](https://github.com/Tecnativa/docker-socket-proxy) vor den Daemon.

## <Icon icon="mdi:key-outline" inline /> Eine Methode wählen

| Methode                                                 | Wem die Benutzer gehören | Wann du sie nimmst                                                                                                                                           |
| ------------------------------------------------------- | ------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| [Simple](/de/guide/authentication/simple)               | Dozzle, in `users.yml`   | Du hast keine Authentifizierungslösung und möchtest, dass Dozzle die Anmeldungen übernimmt.                                                                  |
| [GitHub & OIDC](/de/guide/authentication/oauth)         | Dozzle, in `users.yml`   | Du möchtest, dass sich dieselben Benutzer aus `users.yml` mit GitHub, Google, Keycloak, Pocket ID, Zitadel oder Authentik anmelden statt mit einem Passwort. |
| [Forward Proxy](/de/guide/authentication/forward-proxy) | Dein Proxy               | Du betreibst bereits Authelia, Authentik, Cloudflare Access oder Ähnliches und möchtest, dass es die Authentifizierung vollständig übernimmt.                |

Simple und OAuth sind derselbe Anbieter: `users.yml` ist in beiden Fällen die Benutzerliste, und OAuth ergänzt nur einen zweiten Weg, um nachzuweisen, dass du einer der Benutzer darin bist. Der Forward Proxy ist der eigenständige Weg, und er ist die richtige Wahl, wenn du organisations- oder domainweite Zugriffsregeln brauchst, was `users.yml` bewusst nicht kann.

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
