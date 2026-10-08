---
# https://vitepress.dev/reference/default-theme-home-page
layout: home

titleTemplate: Docker-Logs in Echtzeit
description: Dozzle ist ein schlanker, quelloffener Log-Viewer für Docker, Swarm und Kubernetes. Streame Logs, verfolge Live-Statistiken und aktualisiere Container sicher direkt im Browser.

hero:
  name: "Dozzle"
  text: "Sieh, was deine Container gerade tun"
  tagline: Docker-Logs, Statistiken und Updates in Echtzeit — direkt im Browser.
  actions:
    - theme: brand
      text: Loslegen
      link: /de/guide/getting-started
    - theme: alt
      text: Auf GitHub ansehen
      link: https://github.com/amir20/dozzle

features:
  - title: Logs in Echtzeit
    details: Streame Container-Logs, während sie entstehen. Suche, filtere nach Level oder Zeitraum und frage JSON-Felder mit SQL ab, ohne den Host anzufassen.
    icon:
      src: /icons/document.svg
      width: 36
      height: 36
    link: /de/guide/what-is-dozzle#advanced-log-handling
    linkText: Mehr erfahren
  - title: Container- und Host-Metriken
    details: Verfolge CPU, Speicher, Netzwerk und Festplatte für jeden Container, dazu Last, Festplatte und Uptime für jeden Host.
    icon:
      src: /icons/chart-line-data.svg
      width: 36
      height: 36
    link: /de/guide/what-is-dozzle#real-time-monitoring
    linkText: Mehr erfahren
  - title: Sichere Updates
    details: Erkenne veraltete Images und aktualisiere Container einzeln, gesammelt oder nach Zeitplan. Läuft ein neuer Container nicht stabil, kommt der alte zurück.
    icon:
      src: /icons/update-now.svg
      width: 36
      height: 36
    link: /de/guide/updates
    linkText: Mehr erfahren
  - title: Docker, Swarm und Kubernetes
    details: Behalte mehrere Docker-Hosts, Swarm-Cluster oder einen Kubernetes-Cluster in einer Oberfläche im Blick. TLS-Agents fügst du direkt im Browser hinzu.
    icon:
      src: /icons/network-3.svg
      width: 36
      height: 36
    link: /de/guide/remote-hosts
    linkText: Mehr erfahren
  - title: Alarme und Webhooks
    details: Erkenne Log-Muster, Metriken und Lifecycle-Events mit Ausdrücken und benachrichtige Slack, Discord, ntfy oder jeden Webhook.
    icon:
      src: /icons/notification-new.svg
      width: 36
      height: 36
    link: /de/guide/alerts-and-webhooks
    linkText: Mehr erfahren
  - title: Dozzle Cloud
    details: Eine optionale verwaltete Ebene, die sich Alarme merkt, Fehler markiert, die für einen Container neu sind, und dir per E-Mail, Telegram oder Discord sagt, was kaputt ist.
    icon:
      src: /icons/cloud.svg
      width: 36
      height: 36
    link: /de/guide/dozzle-cloud
    linkText: Mehr erfahren
  - title: Shell und Aktionen
    details: Starte, stoppe und starte Container neu, löse einen Kubernetes-Rollout-Neustart aus oder öffne eine Shell, wenn du tiefer graben musst.
    icon:
      src: /icons/terminal.svg
      width: 36
      height: 36
    link: /de/guide/actions
    linkText: Mehr erfahren
  - title: MCP für KI-Assistenten
    details: Stelle Container, Logs und Statistiken über das Model Context Protocol mit OAuth-Anmeldung bereit, damit dein Coding-Agent mit dir zusammen debuggen kann.
    icon:
      src: /icons/ai.svg
      width: 36
      height: 36
    link: /de/guide/mcp
    linkText: Mehr erfahren
sourceHash: 135ab46c4cca
---
