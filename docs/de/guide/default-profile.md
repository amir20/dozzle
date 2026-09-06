---
title: Standardprofil
sourceHash: 1ef0edd24fb4
---

# Standardprofil

Dozzle speichert die UI-Einstellungen pro Benutzer (Theme, Sprache, angeheftete Container, eingeklappte Gruppen, sichtbare JSON-Schlüssel usw.) unter `/data/<username>/profile.json` auf der Festplatte. Wenn die [Authentifizierung](/de/guide/authentication) deaktiviert ist, oder für jeden Benutzer, der sich noch nicht angemeldet und seine Einstellungen angepasst hat, greift Dozzle auf ein spezielles Profil namens `__default__` zurück.

Du kannst ein vorkonfiguriertes Profil ausliefern, indem du die Datei `/data/__default__/profile.json` anlegst. Anonyme Besucher und alle neuen Benutzer ohne gespeichertes Profil laden diese Einstellungen beim ersten Besuch.

## Speicherort der Datei

```
/data/__default__/profile.json
```

Existiert die Datei nicht, startet Dozzle mit den eingebauten Standardwerten. Du musst sie nur anlegen, wenn du diese überschreiben willst.

## Beispiel

```json
{
  "settings": {
    "showTimestamp": true,
    "showStd": false,
    "showAllContainers": false,
    "softWrap": true,
    "collapseNav": false,
    "smallerScrollbars": false,
    "search": false,
    "compact": false,
    "menuWidth": 15,
    "size": "medium",
    "lightTheme": "auto",
    "hourStyle": "auto",
    "dateLocale": "auto",
    "locale": "en",
    "groupContainers": "at-least-2",
    "automaticRedirect": "delayed"
  },
  "pinned": [],
  "visibleKeys": [],
  "collapsedGroups": []
}
```

Alle Felder sind optional — nimm nur die auf, die du überschreiben willst.

## Verfügbare Einstellungen

| Feld                | Typ     | Beschreibung                                                         |
| ------------------- | ------- | -------------------------------------------------------------------- |
| `showTimestamp`     | boolean | Zeitstempel neben jeder Logzeile anzeigen                            |
| `showStd`           | boolean | Anzeige des stdout/stderr-Streams einblenden                         |
| `showAllContainers` | boolean | Gestoppte Container in der Seitenleiste einschließen                 |
| `softWrap`          | boolean | Lange Logzeilen umbrechen statt horizontal zu scrollen               |
| `collapseNav`       | boolean | Mit eingeklappter Seitenleiste starten                               |
| `smallerScrollbars` | boolean | Schmalere Scrollbalken verwenden                                     |
| `search`            | boolean | Inline-Suche standardmäßig aktivieren                                |
| `compact`           | boolean | Kompakter Zeilenabstand im Log                                       |
| `menuWidth`         | number  | Breite der Seitenleiste in Prozent des Fensters. Maximal `50`.       |
| `size`              | string  | Schriftgröße: `small`, `medium`, `large`                             |
| `lightTheme`        | string  | Theme-Voreinstellung: `auto`, `light`, `dark`                        |
| `hourStyle`         | string  | Zeitformat: `auto`, `12`, `24`                                       |
| `dateLocale`        | string  | Datums-/Zeitformat: `auto`, `en-US`, `en-GB`, `de-DE`, `en-CA`       |
| `locale`            | string  | Sprache der Oberfläche (z. B. `en`, `fr`, `de`)                      |
| `groupContainers`   | string  | Gruppierung in der Seitenleiste: `always`, `at-least-2`, `never`     |
| `automaticRedirect` | string  | Weiterleitung zu einem neuen Container: `instant`, `delayed`, `none` |

Werte außerhalb dieser Mengen werden nicht akzeptiert, `groupContainers: "stack"` oder ein `dateLocale` von `fr-FR` tun also nicht das, was du erwartest.

Die Felder `pinned`, `visibleKeys` und `collapsedGroups` auf oberster Ebene nehmen Arrays entgegen und erlauben es dir, für Erstbesucher Container vorab anzuheften oder Gruppen vorab einzuklappen. Dozzle schreibt auf oberster Ebene außerdem `releaseSeen`, `dismissedImageUpdates` und `dismissedLinkHint`, um sich zu merken, was ein Benutzer bereits weggeklickt hat. Wenn du `dismissedLinkHint: true` vorbelegst, wird der Link-Hinweis beim ersten Start für alle unterdrückt.

## Wie es funktioniert

- Beim Laden der Seite liest Dozzle `/data/<username>/profile.json` für den angemeldeten Benutzer oder `/data/__default__/profile.json`, wenn niemand authentifiziert ist.
- Ändert ein Benutzer eine Einstellung in der Oberfläche, wird der neue Wert unter seinem eigenen Benutzernamen gespeichert (oder wieder in `__default__`, wenn die Authentifizierung deaktiviert ist).
- Das Profil `__default__` ist damit sowohl die **Vorlage für neue Besucher** als auch das **aktive Profil des anonymen Benutzers** in Deployments ohne Authentifizierung.

::: tip
Wenn du nur Standardwerte vorgeben, dem anonymen Benutzer aber weiterhin Anpassungen zur Laufzeit erlauben willst, binde die Datei schreibgeschützt ein — Dozzle kann Änderungen dann nicht speichern, die Oberfläche funktioniert aber weiter.
:::
