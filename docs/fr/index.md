---
# https://vitepress.dev/reference/default-theme-home-page
layout: home

titleTemplate: Visualiseur de logs Docker en temps réel
description: Dozzle est un visualiseur de logs léger et open source pour Docker, Swarm et Kubernetes. Diffusez vos logs, suivez les statistiques en direct et mettez à jour vos conteneurs en toute sécurité depuis votre navigateur.

hero:
  name: "Dozzle"
  text: "Voyez ce que font vos conteneurs"
  tagline: Logs Docker, statistiques et mises à jour en temps réel, dans votre navigateur.
  actions:
    - theme: brand
      text: Démarrer
      link: /fr/guide/getting-started
    - theme: alt
      text: Voir sur GitHub
      link: https://github.com/amir20/dozzle

features:
  - title: Logs en temps réel
    details: Diffusez les logs des conteneurs au fil de l'eau. Cherchez, filtrez par niveau ou par période et interrogez les champs JSON en SQL sans toucher à l'hôte.
    icon:
      src: /icons/document.svg
      width: 36
      height: 36
    link: /fr/guide/what-is-dozzle#advanced-log-handling
    linkText: En savoir plus
  - title: Métriques des conteneurs et des hôtes
    details: Suivez le CPU, la mémoire, le réseau et le disque de chaque conteneur, ainsi que la charge, le disque et l'uptime de chaque hôte.
    icon:
      src: /icons/chart-line-data.svg
      width: 36
      height: 36
    link: /fr/guide/what-is-dozzle#real-time-monitoring
    linkText: En savoir plus
  - title: Mises à jour sûres
    details: Repérez les images obsolètes et mettez à jour vos conteneurs un par un, en lot ou selon un planning. Si un nouveau conteneur ne tient pas, l'ancien revient.
    icon:
      src: /icons/update-now.svg
      width: 36
      height: 36
    link: /fr/guide/updates
    linkText: En savoir plus
  - title: Docker, Swarm et Kubernetes
    details: Surveillez plusieurs hôtes Docker, clusters Swarm ou un cluster Kubernetes depuis une seule interface. Ajoutez des agents TLS directement depuis le navigateur.
    icon:
      src: /icons/network-3.svg
      width: 36
      height: 36
    link: /fr/guide/remote-hosts
    linkText: En savoir plus
  - title: Alertes et webhooks
    details: Repérez des motifs dans vos logs, des métriques et des événements de cycle de vie avec des expressions, puis notifiez Slack, Discord, ntfy ou n'importe quel webhook.
    icon:
      src: /icons/notification-new.svg
      width: 36
      height: 36
    link: /fr/guide/alerts-and-webhooks
    linkText: En savoir plus
  - title: Dozzle Cloud
    details: Une couche managée optionnelle qui garde la mémoire des alertes, signale les erreurs nouvelles pour un conteneur et vous dit ce qui a cassé par e-mail, Telegram ou Discord.
    icon:
      src: /icons/cloud.svg
      width: 36
      height: 36
    link: /fr/guide/dozzle-cloud
    linkText: En savoir plus
  - title: Shell et actions
    details: Démarrez, arrêtez et redémarrez vos conteneurs, lancez un redémarrage de rollout Kubernetes ou ouvrez un shell quand il faut creuser.
    icon:
      src: /icons/terminal.svg
      width: 36
      height: 36
    link: /fr/guide/actions
    linkText: En savoir plus
  - title: MCP pour les assistants IA
    details: Exposez conteneurs, logs et statistiques via le Model Context Protocol, avec connexion OAuth, pour que votre agent de code débogue avec vous.
    icon:
      src: /icons/ai.svg
      width: 36
      height: 36
    link: /fr/guide/mcp
    linkText: En savoir plus
sourceHash: 135ab46c4cca
---
