---
title: Container-Links
sourceHash: d39d7f2119c5
---

# Container-Links

Die meisten Container, die man im Blick behalten will, bringen auch eine Weboberfläche mit. Setze das Label `dev.dozzle.url`, dann zeigt Dozzle neben dem Container-Namen einen Link darauf an, und du kommst von den Logs direkt zur Anwendung.

::: code-group

```sh
docker run --label dev.dozzle.url=https://grafana.example.com grafana/grafana
```

```yaml [docker-compose.yml]
services:
  grafana:
    image: grafana/grafana
    labels:
      - dev.dozzle.url=https://grafana.example.com
```

:::

Der Link taucht an drei Stellen auf: in der Seitenleiste, in der Container-Tabelle und in der Titelleiste der Container-Seite. Er öffnet immer einen neuen Tab, ein Klick führt dich also nie von den Logs weg.

## Was das Label akzeptiert

Der Wert muss eine absolute `http`- oder `https`-URL sein. Alles andere, also ein relativer Pfad, ein bloßer Hostname oder ein anderes Schema, wird ignoriert und es wird kein Link angezeigt.

Dozzle prüft nicht, ob die URL auflösbar ist, und schreibt sie auch nicht pro Host um. Es wird genau das geöffnet, was du hinterlegt hast, nimm also eine Adresse, die aus dem Browser funktioniert, in dem du Dozzle ansiehst.

## Warum es keine automatische Erkennung gibt

Dozzle weiß, welche Ports ein Container veröffentlicht, aber ein veröffentlichter Port ist noch keine erreichbare URL. Reverse Proxies, eigene Pfade, TLS und getrennte Netzwerke sorgen oft genug für eine falsche Vermutung, dass es lästig wird. Das Label macht es eindeutig: Dozzle zeigt nur den Link, den du hinterlegt hast.

Bei Containern ohne Label zeigt Dozzle ein blasses Link-Symbol neben dem Namen an, sowohl im Dashboard als auch auf der Container-Seite. Darüber öffnet sich ein Snippet, das du in deine Compose-Datei kopieren kannst, vorausgefüllt mit einer Vermutung. Wenn du den Hinweis wegklickst, verschwindet er überall.

Die Vermutung stammt aus zwei Quellen. Zuerst werden die Router-Labels von Traefik gelesen, denn eine Router-Regel nennt eine Adresse, unter der der Container aus dem Browser tatsächlich erreichbar ist:

```yaml
labels:
  - traefik.http.routers.grafana.rule=Host(`grafana.example.com`)
  - traefik.http.routers.grafana.tls=true
```

Pfad-Präfixe werden angehängt, das Schema richtet sich nach den TLS-Einstellungen und Entrypoints des Routers, und `traefik.enable=false` schaltet das Ganze ab. Andernfalls fällt Dozzle auf einen veröffentlichten Host-Port in Kombination mit dem Hostnamen zurück, unter dem du Dozzle gerade ansiehst. Beides wird nur in das Snippet vorausgefüllt. Ein Link wird daraus erst, wenn du es selbst in `dev.dozzle.url` einträgst.

Container hinter einem Reverse Proxy veröffentlichen gar keinen Host-Port, die Traefik-Labels sind dort also meist das einzige Signal. Nutzt du einen anderen Proxy, bleibt der Hinweis aus und du setzt das Label von Hand.

## Swarm

In Swarm setzt `deploy.labels` Labels auf dem Service und der Schlüssel `labels` auf oberster Ebene setzt sie auf dem Container. Der Swarm-Provider von Traefik liest Service-Labels, deshalb landen sie üblicherweise dort:

```yaml
services:
  ui:
    image: my/ui
    deploy:
      labels:
        - traefik.http.routers.ui.rule=Host(`app.example.com`)
        - dev.dozzle.url=https://app.example.com
```

Dozzle überträgt Service-Labels zurück auf jeden Task-Container, damit funktionieren `dev.dozzle.url` und der Traefik-Hinweis auch aus `deploy.labels`. Dasselbe gilt für `dev.dozzle.name`, `dev.dozzle.group` und `dev.dozzle.icon`. Ein direkt am Container gesetztes Label sticht das des Service.

Services aufzulisten ist eine API nur für Manager. In einem Swarm mit mehreren Knoten können die Agents auf Worker-Knoten keine Service-Labels lesen, dort geplante Container sehen also nur ihre eigenen.

## Verwandte Labels

- [`dev.dozzle.name`](/de/guide/container-names) legt einen eigenen Anzeigenamen fest
- [`dev.dozzle.group`](/de/guide/container-groups) fasst Container zu Gruppen zusammen
- [`dev.dozzle.icon`](/de/guide/app-icons) wählt das App-Icon
