---
title: App-Icons
sourceHash: 2b128908b958
---

# App-Icons

Dozzle ordnet bekannte Container-Images ihrem Projektlogo zu und zeigt es neben dem Container-Namen in der Seitenleiste, in der Container-Tabelle und in der Befehlspalette an. Wenn du einen \*arr-Stack, Plex oder Home Assistant betreibst, lässt sich die Liste deutlich schneller erfassen.

Die Icons sind in Dozzle enthalten. Sie werden nie von einem CDN geladen, es verlässt also nichts über deine Container dein Netzwerk und alles funktioniert auch ohne Internetzugang.

## Abschalten

Der Schalter liegt unter **Einstellungen → Optionen → App-Icons anzeigen**. Es ist eine Einstellung pro Profil und gilt damit nur für deinen Browser.

## Wie die Zuordnung funktioniert

Dozzle schaut sich den Image-Namen an und ignoriert dabei Registry, Tag und Digest. Das letzte Pfadsegment entscheidet, all diese Varianten landen also bei Sonarr:

- `sonarr`
- `linuxserver/sonarr:latest`
- `lscr.io/linuxserver/sonarr`
- `ghcr.io/hotio/sonarr@sha256:...`

Ist der Image-Name zu allgemein, greift Dozzle auf den Namespace zurück. So landet `ghcr.io/goauthentik/server` bei Authentik.

## Das Icon überschreiben

Manche Images passen zu keinem Eintrag, und ein Fork kann beim falschen Logo landen. Setze das Label `dev.dozzle.icon`, um selbst ein Icon zu wählen, oder auf `none`, um es für diesen Container auszublenden.

::: code-group

```sh
docker run --label dev.dozzle.icon=plex my-custom-media-server
```

```yaml [docker-compose.yml]
services:
  media:
    image: my-custom-media-server
    labels:
      - dev.dozzle.icon=plex

  scratch:
    image: alpine
    labels:
      - dev.dozzle.icon=none
```

:::

Der Wert ist ein Icon-Name aus [dashboard-icons](https://github.com/homarr-labs/dashboard-icons). Verfügbar sind nur die Icons, die Dozzle mitliefert. Bei einem unbekannten Namen wird kein Icon angezeigt.

## Ein eigenes Icon verwenden

Für ein Image, das Dozzle nie mitliefern wird, etwa dein eigenes Projekt, kann das Label das Icon selbst als Data-URI enthalten. Es wird nichts hochgeladen oder gemountet, und nichts wird aus dem Netzwerk geladen.

```yaml [docker-compose.yml]
services:
  app:
    image: my-company/internal-thing
    labels:
      - dev.dozzle.icon=data:image/svg+xml;base64,PHN2ZyB4bWxucz0i...
```

Den Wert erzeugst du mit `base64`:

```sh
echo "data:image/svg+xml;base64,$(base64 < icon.svg | tr -d '\n')"
```

Akzeptiert werden SVG, PNG und WebP, und der gesamte Wert darf höchstens 16 KB groß sein. Alles andere führt dazu, dass kein Icon angezeigt wird. Icons werden mit etwa 20 px dargestellt, ein optimiertes SVG oder ein WebP mit 64 px reicht also völlig.

Wenn du ein Image veröffentlichst, setze das Label im Dockerfile, dann bekommen alle, die es ausführen, das Icon ohne jede Konfiguration.

```dockerfile
LABEL dev.dozzle.icon="data:image/svg+xml;base64,PHN2ZyB4bWxucz0i..."
```

## Fehlt ein Icon?

Dozzle liefert eine kuratierte Auswahl statt des kompletten Satzes von 3.000 Icons, damit das Image klein bleibt. Wenn etwas Verbreitetes fehlt, [erstelle ein Issue](https://github.com/amir20/dozzle/issues) mit dem Image-Namen, dann kann es ergänzt werden.
