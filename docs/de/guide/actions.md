---
title: Container-Aktionen
sourceHash: 0b048a644851
---

# Container-Aktionen

<Badge type="warning" text="Docker Only" />

Dozzle unterstützt Container-Aktionen: Über das Dropdown-Menü rechts neben den Container-Statistiken kannst du Container `start`en, `stop`pen, neu starten (`restart`), entfernen (`remove`) und aktualisieren (`update`). Diese Funktion ist standardmäßig **deaktiviert** und lässt sich aktivieren, indem du die Umgebungsvariable `DOZZLE_ENABLE_ACTIONS` auf `true` setzt.

Die Aktion `update` lädt das neueste Image für den Container und erstellt ihn mit derselben Konfiguration neu — praktisch, um einen Container an Ort und Stelle zu aktualisieren, ohne seine Compose-Datei zu bearbeiten. `update` hat nur dann einen spürbaren Effekt, wenn das Image ein bewegliches Tag nutzt (z. B. `latest`, `stable`); bei einem fest gepinnten Tag wird schlicht dasselbe Image erneut geladen.

> [!WARNING]
> `remove` löscht den Container: Daten in seiner beschreibbaren Schicht gehen verloren, und seine anonymen Volumes bleiben losgelöst zurück. `update` erstellt den Container neu und behält jedes Volume, auch anonyme, sowie jeden Bind-Mount. Verloren gehen nur Daten, die in die beschreibbare Schicht des Containers geschrieben wurden.

::: code-group

```sh
docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --enable-actions
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    ports:
      - 8080:8080
    environment:
      DOZZLE_ENABLE_ACTIONS: true
```

:::

## Updates {#auto-updating-containers}

Jedes Update behält den alten Container, bis der neue stabil läuft, und stellt ihn sonst wieder her. Die Prüfung auf neuere Images, das Aktualisieren mehrerer Container auf einmal, der Zeitplan für automatische Updates, das Aufräumen und das Zurückrollen stehen unter [Container-Updates](/de/guide/updates).
