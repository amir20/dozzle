---
# https://vitepress.dev/reference/default-theme-home-page
layout: home

titleTemplate: Visor de logs de Docker en tiempo real
description: Dozzle es un visor de logs ligero y de código abierto para Docker, Swarm y Kubernetes. Transmite logs, observa estadísticas en vivo y actualiza contenedores de forma segura desde el navegador.

hero:
  name: "Dozzle"
  text: "Mira qué están haciendo tus contenedores"
  tagline: Logs, estadísticas y actualizaciones de Docker en tiempo real, desde el navegador.
  actions:
    - theme: brand
      text: Empezar
      link: /es/guide/getting-started
    - theme: alt
      text: Ver en GitHub
      link: https://github.com/amir20/dozzle

features:
  - title: Logs en tiempo real
    details: Transmite los logs de los contenedores según se generan. Busca, filtra por nivel o por rango de tiempo y consulta campos JSON con SQL sin tocar el host.
    icon:
      src: /icons/document.svg
      width: 36
      height: 36
    link: /es/guide/what-is-dozzle#advanced-log-handling
    linkText: Saber más
  - title: Métricas de contenedores y hosts
    details: Observa CPU, memoria, red y disco de cada contenedor, además de la carga, el disco y el tiempo activo de cada host.
    icon:
      src: /icons/chart-line-data.svg
      width: 36
      height: 36
    link: /es/guide/what-is-dozzle#real-time-monitoring
    linkText: Saber más
  - title: Actualizaciones seguras
    details: Detecta imágenes desactualizadas y actualiza contenedores uno a uno, en bloque o según una programación. Si un contenedor nuevo no se mantiene en marcha, vuelve el antiguo.
    icon:
      src: /icons/update-now.svg
      width: 36
      height: 36
    link: /es/guide/updates
    linkText: Saber más
  - title: Docker, Swarm y Kubernetes
    details: Vigila varios hosts de Docker, clústeres de Swarm o un clúster de Kubernetes desde una sola interfaz. Añade agentes TLS directamente desde el navegador.
    icon:
      src: /icons/network-3.svg
      width: 36
      height: 36
    link: /es/guide/remote-hosts
    linkText: Saber más
  - title: Alertas y webhooks
    details: Detecta patrones en los logs, métricas y eventos del ciclo de vida con expresiones, y avisa a Slack, Discord, ntfy o cualquier webhook.
    icon:
      src: /icons/notification-new.svg
      width: 36
      height: 36
    link: /es/guide/alerts-and-webhooks
    linkText: Saber más
  - title: Dozzle Cloud
    details: Una capa gestionada opcional que recuerda las alertas, marca los errores nuevos para un contenedor y te cuenta qué se ha roto por correo, Telegram o Discord.
    icon:
      src: /icons/cloud.svg
      width: 36
      height: 36
    link: /es/guide/dozzle-cloud
    linkText: Saber más
  - title: Shell y acciones
    details: Inicia, detén y reinicia contenedores, lanza un reinicio de rollout en Kubernetes o abre una shell cuando necesites indagar más.
    icon:
      src: /icons/terminal.svg
      width: 36
      height: 36
    link: /es/guide/actions
    linkText: Saber más
  - title: MCP para asistentes de IA
    details: Expón contenedores, logs y estadísticas mediante el Model Context Protocol, con inicio de sesión OAuth, para que tu agente de código depure contigo.
    icon:
      src: /icons/ai.svg
      width: 36
      height: 36
    link: /es/guide/mcp
    linkText: Saber más
sourceHash: 135ab46c4cca
---
