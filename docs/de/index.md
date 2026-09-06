---
# https://vitepress.dev/reference/default-theme-home-page
layout: home

titleTemplate: Docker-Logs in Echtzeit
description: Dozzle ist ein schlanker, quelloffener Log-Viewer für Docker, Swarm und Kubernetes. Streame Logs, verfolge Live-Statistiken und debugge Container direkt im Browser.

hero:
  name: "Dozzle"
  text: "Sieh, was deine Container gerade tun"
  tagline: Docker-Logs, Statistiken und Fehlersuche in Echtzeit — direkt im Browser.
  actions:
    - theme: brand
      text: Loslegen
      link: /de/guide/getting-started
    - theme: alt
      text: Auf GitHub ansehen
      link: https://github.com/amir20/dozzle

features:
  - title: Logs in Echtzeit
    details: Streame Container-Logs, während sie entstehen. Suche, filtere und verfolge sie über Container hinweg, ohne den Host anzufassen.
    icon:
      src: /icons/document.svg
      width: 36
      height: 36
    link: /de/guide/what-is-dozzle#advanced-log-handling
    linkText: Mehr erfahren
  - title: Live-Statistiken und Metriken
    details: Verfolge CPU-, Speicher- und Netzwerkauslastung in Echtzeit, mit fortlaufenden Verlaufsdiagrammen für jeden Container.
    icon:
      src: /icons/chart-line-data.svg
      width: 36
      height: 36
    link: /de/guide/what-is-dozzle#real-time-monitoring
    linkText: Mehr erfahren
  - title: Log-Analyse mit SQL
    details: Frage deine Logs mit DuckDB und WebAssembly ab — vollständiges SQL, komplett im Browser.
    icon:
      src: /icons/sql.svg
      width: 36
      height: 36
    link: /de/guide/sql-engine
    linkText: Mehr erfahren
  - title: Alarme und Webhooks
    details: Erkenne Log-Muster mit mächtigen Ausdrücken und benachrichtige Slack, Discord, ntfy oder jeden Webhook.
    icon:
      src: /icons/notification-new.svg
      width: 36
      height: 36
    link: /de/guide/alerts-and-webhooks
    linkText: Mehr erfahren
  - title: Multi-Host und Swarm
    details: Verbinde dich aus einer einzigen Oberfläche mit mehreren Docker-Hosts und Swarm-Clustern, abgesichert über TLS-Agents.
    icon:
      src: /icons/network-3.svg
      width: 36
      height: 36
    link: /de/guide/remote-hosts
    linkText: Mehr erfahren
  - title: Shell- und Exec-Zugriff
    details: Hänge dich an laufende Container an oder führe Befehle direkt aus dem Browser aus, wenn du tiefer graben musst.
    icon:
      src: /icons/terminal.svg
      width: 36
      height: 36
    link: /de/guide/shell
    linkText: Mehr erfahren
  - title: dtop in deinem Terminal
    details: Ein Kommandozeilen-Begleiter, der dir deine Container live anzeigt und von dort direkt nach Dozzle springt.
    icon:
      src: /icons/terminal-command.svg
      width: 36
      height: 36
    link: /de/guide/dtop
    linkText: Mehr erfahren
  - title: MCP für KI-Assistenten
    details: Stelle Container, Logs und Statistiken über das Model Context Protocol bereit, damit dein Coding-Agent mit dir zusammen debuggen kann.
    icon:
      src: /icons/ai.svg
      width: 36
      height: 36
    link: /de/guide/mcp
    linkText: Mehr erfahren
  - title: Selbst gehostet und privat
    details: Läuft in deiner eigenen Infrastruktur, mit einfacher Auth oder Forward-Proxy-Auth. Deine Logs verlassen nie dein Netzwerk.
    icon:
      src: /icons/locked.svg
      width: 36
      height: 36
    link: /de/guide/authentication
    linkText: Mehr erfahren
sourceHash: a11fae50734d
---
