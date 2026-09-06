---
title: Reverse Proxy & Basispfad
sourceHash: f344ea2a42cc
---

# Reverse Proxy & Basispfad

Dozzle wird häufig hinter einem Reverse Proxy betrieben, für TLS-Terminierung, Authentifizierung oder um sich einen Hostnamen mit anderen Diensten zu teilen. Diese Seite behandelt sowohl den Betrieb unter einem Unterpfad als auch die Proxy-Einstellungen, die das Streaming korrekt funktionieren lassen.

## Basispfad ändern

Dozzle wird standardmäßig unter `/` bereitgestellt. Das lässt sich mit der Option `--base` oder der Umgebungsvariable `DOZZLE_BASE` ändern. Zum Beispiel für `/foobar`:

::: code-group

```sh
docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --base /foobar
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
      DOZZLE_BASE: /foobar
```

:::

Dozzle ist dann unter `http://localhost:8080/foobar/` erreichbar. Diese Option schreibt alle Assets auf `/foobar/{file.path}` um und leitet `/foobar` automatisch auf `/foobar/` weiter.

## Anforderungen an den Proxy

Dozzle streamt Logs über **Server-Sent Events (SSE)** und nutzt **WebSocket** für Container-Shells. Reverse Proxies müssen daher:

1. **Response-Buffering deaktivieren** — SSE liefert Ereignisse, sobald sie auftreten. Jede Pufferung führt dazu, dass Logs schubweise oder gar nicht ankommen. Dozzle sendet `X-Accel-Buffering: no`, manche Proxies ignorieren das aber.
2. **WebSocket-Upgrade-Header weiterleiten** — nötig für die Shell- und Attach-Funktionen.
3. **`text/event-stream` nicht komprimieren** — Komprimierungs-Middleware bricht SSE oft.

## Nginx

```nginx
location ^~ /foobar/ {
    proxy_pass http://dozzle:8080;

    chunked_transfer_encoding off;
    proxy_buffering off;
    proxy_cache off;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
}
```

Lass das Präfix `^~ /foobar/` weg, wenn Dozzle im Wurzelpfad läuft. Siehe auch den FAQ-Eintrag zum [Deaktivieren des Bufferings](/de/guide/faq#disabling-buffering-in-nginx).

## Traefik

Traefik verarbeitet WebSocket-Upgrades automatisch, die Standard-Middleware `compress` bricht aber SSE. Schließe `text/event-stream` aus:

```yaml
http:
  middlewares:
    middlewares-compress:
      compress:
        excludedContentTypes:
          - text/event-stream
```

Ein typischer Labels-Block am Dozzle-Service sieht dann so aus:

```yaml
services:
  dozzle:
    image: amir20/dozzle:latest
    labels:
      - traefik.enable=true
      - traefik.http.routers.dozzle.rule=Host(`dozzle.example.com`)
      - traefik.http.routers.dozzle.entrypoints=websecure
      - traefik.http.routers.dozzle.tls.certresolver=letsencrypt
      - traefik.http.services.dozzle.loadbalancer.server.port=8080
```

## Caddy

```caddyfile
dozzle.example.com {
    reverse_proxy dozzle:8080 {
        flush_interval -1
    }
}
```

`flush_interval -1` deaktiviert das Response-Buffering für Streaming-Endpunkte.

## Häufige Stolpersteine

- **Leere Seite oder Assets mit 404 bei `--base`** — der Proxy entfernt das Pfadpräfix, bevor er weiterleitet. Konfiguriere ihn so, dass er den vollständigen Pfad an Dozzle durchreicht.
- **Logs hören nach ein paar Sekunden auf** — die Verbindungs-Timeouts am Proxy sind zu kurz. Erhöhe Lese- und Sende-Timeouts auf mindestens ein paar Minuten (z. B. Nginx `proxy_read_timeout 3600s`).
- **Shell trennt sofort die Verbindung** — die WebSocket-Upgrade-Header werden nicht weitergeleitet. Prüfe die Header `Upgrade` und `Connection`.
