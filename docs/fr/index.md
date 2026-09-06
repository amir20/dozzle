---
# https://vitepress.dev/reference/default-theme-home-page
layout: home

titleTemplate: Visualiseur de logs Docker en temps réel
description: Dozzle est un visualiseur de logs léger et open source pour Docker, Swarm et Kubernetes. Diffusez vos logs, suivez les statistiques en direct et déboguez vos conteneurs depuis votre navigateur.

hero:
  name: "Dozzle"
  text: "Voyez ce que font vos conteneurs"
  tagline: Logs Docker, statistiques et débogage en temps réel, dans votre navigateur.
  actions:
    - theme: brand
      text: Démarrer
      link: /fr/guide/getting-started
    - theme: alt
      text: Voir sur GitHub
      link: https://github.com/amir20/dozzle

features:
  - title: Logs en temps réel
    details: Diffusez les logs des conteneurs au fil de l'eau. Cherchez, filtrez et suivez plusieurs conteneurs sans toucher à l'hôte.
    icon:
      src: /icons/document.svg
      width: 36
      height: 36
    link: /fr/guide/what-is-dozzle#advanced-log-handling
    linkText: En savoir plus
  - title: Statistiques et métriques en direct
    details: Suivez l'utilisation du CPU, de la mémoire et du réseau en temps réel, avec un historique graphique sur chaque conteneur.
    icon:
      src: /icons/chart-line-data.svg
      width: 36
      height: 36
    link: /fr/guide/what-is-dozzle#real-time-monitoring
    linkText: En savoir plus
  - title: Analyse SQL des logs
    details: Interrogez vos logs avec DuckDB et WebAssembly. Du SQL complet, exécuté entièrement dans le navigateur.
    icon:
      src: /icons/sql.svg
      width: 36
      height: 36
    link: /fr/guide/sql-engine
    linkText: En savoir plus
  - title: Alertes et webhooks
    details: Repérez des motifs dans vos logs avec des expressions puissantes et notifiez Slack, Discord, ntfy ou n'importe quel webhook.
    icon:
      src: /icons/notification-new.svg
      width: 36
      height: 36
    link: /fr/guide/alerts-and-webhooks
    linkText: En savoir plus
  - title: Multi-hôtes et Swarm
    details: Connectez-vous à plusieurs hôtes Docker et clusters Swarm depuis une seule interface, sécurisée par des agents TLS.
    icon:
      src: /icons/network-3.svg
      width: 36
      height: 36
    link: /fr/guide/remote-hosts
    linkText: En savoir plus
  - title: Accès shell et exec
    details: Attachez-vous aux conteneurs en cours d'exécution ou exécutez des commandes depuis le navigateur quand il faut creuser.
    icon:
      src: /icons/terminal.svg
      width: 36
      height: 36
    link: /fr/guide/shell
    linkText: En savoir plus
  - title: dtop dans votre terminal
    details: Un compagnon en ligne de commande qui affiche vos conteneurs en direct, puis vous emmène directement dans Dozzle.
    icon:
      src: /icons/terminal-command.svg
      width: 36
      height: 36
    link: /fr/guide/dtop
    linkText: En savoir plus
  - title: MCP pour les assistants IA
    details: Exposez conteneurs, logs et statistiques via le Model Context Protocol pour que votre agent de code débogue avec vous.
    icon:
      src: /icons/ai.svg
      width: 36
      height: 36
    link: /fr/guide/mcp
    linkText: En savoir plus
  - title: Auto-hébergé et privé
    details: Tourne sur votre propre infrastructure, avec authentification simple ou par proxy. Vos logs ne quittent jamais votre réseau.
    icon:
      src: /icons/locked.svg
      width: 36
      height: 36
    link: /fr/guide/authentication
    linkText: En savoir plus
sourceHash: a11fae50734d
---
