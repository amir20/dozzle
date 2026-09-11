---
title: Neu in v11
sourceHash: 4b0ece212354
---

# <Icon icon="mdi:party-popper" inline /> Neu in v11

v11 ist die größte optische Veränderung, die Dozzle je hatte. Fast jede Oberfläche wurde nach einer gemeinsamen Designsprache neu gezeichnet: flach, ruhig, neutrale Flächen, Farbe nur für das, was wirklich Aufmerksamkeit braucht. Dazu kam die Anmeldung mit GitHub und OIDC, und der Log-Stream versteht ein paar neue Formate.

## Ein neuer Look

- **Seitenleiste** neu aufgebaut: einklappbare Gruppen mit Anzahl, App-Icons der Container mit dem Status als Eck-Badge und eine eingefärbte aktive Zeile. Das Zusammenführen einer ganzen Gruppe ist jetzt eine Schaltfläche an der Gruppe selbst.
- **Log-Stream** neu gestaltet. Zeitstempel stehen ruhig und ohne Rahmen, einzelne Zeilen bekommen einen Level-Punkt und gruppierte Einträge eine Leiste, und Zeilen mit `warn` oder `error` sind leicht eingefärbt, damit sie beim Scrollen auffallen.
- **Titelleiste des Containers** vereinfacht. Der Name steht vorne, das Image ist gedämpfter Text, den ein Klick kopiert, und das Anheften liegt jetzt auf einer Kartennadel, passend zum Bereich „Angeheftet“ in der Seitenleiste.
- **Startseite**, **Befehlspalette**, **Toasts**, **Attach- und Shell-Panels** sowie das **Container-Menü** wurden neu gezeichnet. Das Menü ist in benannte Abschnitte gegliedert statt einer langen Liste.
- **Live-Anzeige** und die **Scroll-Positionsanzeige** sind neu. Die Anzeige schwebt über dem Stream, sagt, wo in der Lebenszeit des Containers man sich befindet, und verschwindet wieder, sobald man aufhört zu scrollen.
- Alle Menüs nutzen jetzt die native Popover-API, dadurch werden sie nicht mehr abgeschnitten oder in einem scrollenden Bereich eingesperrt.

## Anmeldung mit GitHub und OIDC

Benutzer können sich mit einem GitHub-Konto oder einem beliebigen OIDC-Anbieter anmelden (Authentik, Keycloak, Pocket ID, Google). Das gehört zum Provider `simple`, also bleibt `users.yml` die Positivliste und entscheidet weiterhin, wer hineinkommt. Es wird nie automatisch ein Konto angelegt, und die Anmeldung per Passwort funktioniert daneben weiter.

```yaml
environment:
  DOZZLE_AUTH_PROVIDER: simple
  DOZZLE_AUTH_GITHUB_CLIENT_ID: Ov23liABCDEFGHIJKLMN
  DOZZLE_AUTH_GITHUB_CLIENT_SECRET: 0123456789abcdef0123456789abcdef01234567
```

Die vollständige Einrichtung steht unter [Anmeldung mit GitHub & OIDC](/de/guide/authentication/oauth).

## Logs

- Die OpenTelemetry-Felder `severityText` und `severityNumber` werden als Log-Level erkannt; `severityNumber` wird verwendet, wenn der Text kein Level-Name ist.
- Numerische Pino-Level (`30`, `40`, `50`) werden ausgewertet.
- Die Feld-Schalter wirken auch in Service- und Stack-Ansichten, nicht mehr nur bei einzelnen Containern.
- Angeheftete Spalten stehen in der URL, eine Nebeneinander-Ansicht ist damit ein Link, den man verschicken kann.

## Benachrichtigungen und Dozzle Cloud

- Alarme werden gespeichert. Sie erscheinen an der Regel, die sie ausgelöst hat, und als Punkt in der Container-Zeile, und sie überstehen ein Neuladen.
- Neben dem Stream liegt eine neue Cloud-Leiste mit drei Bereichen: eine Frage zu dem stellen, was gerade zu sehen ist, die Metriken hinter dem Live-Diagramm und die Alarme der Container in der aktuellen Ansicht. Sie erscheint nur, wenn Cloud verbunden ist, und klappt zu einem Reiter am Rand zusammen.
- Cloud-Werkzeuge laufen im Kontext der fragenden Person, der Assistent sieht also genau das, was dieses Konto sehen darf.
- `min_level=disabled` stoppt nicht länger auch die Metriken.

## Performance und Fehlerbehebungen

- Die ersten Log-Zeilen erscheinen deutlich früher.
- Log-Streams, die der Browser still aufgegeben hat, verbinden sich wieder.
- Die Statistikpunkte in der Nutzlast bei Container-Änderungen sind deutlich kleiner.
- Zusammengeführte Log-Streams blockieren den Container-Store nicht mehr.

## Aktualisieren

Sitzungs-Tokens werden jetzt mit einem zufälligen Schlüssel signiert, der als `session_secret` im Datenverzeichnis neben `users.yml` gespeichert wird. **Nach dem Update werden alle einmal abgemeldet.** Ist `/data` nicht beschreibbar, startet Dozzle trotzdem, verwendet einen Schlüssel nur im Arbeitsspeicher und warnt davor, was bedeutet, dass Sitzungen bei jedem Neustart verloren gehen.

Der alte Schlüssel wurde allein aus `users.yml` abgeleitet, und das war zu wenig Entropie, sobald ein Konto per OAuth nachgewiesen wird und gar kein Passwort-Hash mehr enthält.
