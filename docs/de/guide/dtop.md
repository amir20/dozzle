---
title: dtop im Überblick
sourceHash: 3137db243510
---

# Was ist dtop?

`dtop` ist ein Kommandozeilen-Begleiter für Dozzle und zeigt dir die auf deinem System laufenden Docker-Container live im Terminal. Stell es dir als reichhaltigeres `docker ps` vor, das du in einem tmux-Pane offen lassen kannst. Und wenn du die vollständige Log-Historie, die Suche oder die Diagramme brauchst, springst du mit `dtop` direkt nach Dozzle.

Es verbindet sich über `ssh`, `tcp` oder einen lokalen `unix socket` mit Docker-Hosts und passt damit gut zu denselben Multi-Host-Setups, die auch Dozzle unterstützt.

![dtop Screenshot](https://github.com/amir20/dtop/raw/master/demo.gif)

## Installation

Installation mit Homebrew:

```bash
brew install dtop
```

Oder ohne Installation direkt über Docker ausführen:

```bash
docker run -v /var/run/docker.sock:/var/run/docker.sock -it ghcr.io/amir20/dtop:latest
```

Die vollständige Installationsanleitung findest du unter [https://github.com/amir20/dtop](https://github.com/amir20/dtop?tab=readme-ov-file#installation).

## Projektstatus

`dtop` ist ein neues Projekt und noch nicht so umfangreich wie Dozzle. Ich arbeite aber aktiv daran, weitere Funktionen zu ergänzen. Ich selbst nutze es, um alle meine Container über mehrere Hosts hinweg auf der Kommandozeile im Blick zu behalten. Wenn du Vorschläge hast, erstelle gerne ein Issue unter [https://github.com/amir20/dtop/issues](https://github.com/amir20/dtop/issues).
