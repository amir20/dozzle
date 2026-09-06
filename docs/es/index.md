---
# https://vitepress.dev/reference/default-theme-home-page
layout: home

titleTemplate: Visor de logs de Docker en tiempo real
description: Dozzle es un visor de logs ligero y de código abierto para Docker, Swarm y Kubernetes. Transmite logs, observa estadísticas en vivo y depura contenedores desde el navegador.

hero:
  name: "Dozzle"
  text: "Mira qué están haciendo tus contenedores"
  tagline: Logs, estadísticas y depuración de Docker en tiempo real, desde el navegador.
  actions:
    - theme: brand
      text: Empezar
      link: /es/guide/getting-started
    - theme: alt
      text: Ver en GitHub
      link: https://github.com/amir20/dozzle

features:
  - title: Logs en tiempo real
    details: Transmite los logs de los contenedores según se generan. Busca, filtra y síguelos entre contenedores sin tocar el host.
    icon:
      src: /icons/document.svg
      width: 36
      height: 36
    link: /es/guide/what-is-dozzle#advanced-log-handling
    linkText: Saber más
  - title: Estadísticas y métricas en vivo
    details: Observa el uso de CPU, memoria y red actualizándose en tiempo real, con gráficos de historial en cada contenedor.
    icon:
      src: /icons/chart-line-data.svg
      width: 36
      height: 36
    link: /es/guide/what-is-dozzle#real-time-monitoring
    linkText: Saber más
  - title: Análisis de logs con SQL
    details: Consulta tus logs con DuckDB y WebAssembly, con SQL completo que se ejecuta enteramente en el navegador.
    icon:
      src: /icons/sql.svg
      width: 36
      height: 36
    link: /es/guide/sql-engine
    linkText: Saber más
  - title: Alertas y webhooks
    details: Detecta patrones en los logs con expresiones potentes y avisa a Slack, Discord, ntfy o cualquier webhook.
    icon:
      src: /icons/notification-new.svg
      width: 36
      height: 36
    link: /es/guide/alerts-and-webhooks
    linkText: Saber más
  - title: Multihost y Swarm
    details: Conéctate a varios hosts de Docker y clústeres de Swarm desde una sola interfaz, con agentes protegidos por TLS.
    icon:
      src: /icons/network-3.svg
      width: 36
      height: 36
    link: /es/guide/remote-hosts
    linkText: Saber más
  - title: Acceso a shell y exec
    details: Conéctate a contenedores en ejecución o ejecuta comandos directamente desde el navegador cuando necesites indagar más.
    icon:
      src: /icons/terminal.svg
      width: 36
      height: 36
    link: /es/guide/shell
    linkText: Saber más
  - title: dtop en tu terminal
    details: Un complemento de línea de comandos que te muestra tus contenedores en vivo y salta directamente a Dozzle.
    icon:
      src: /icons/terminal-command.svg
      width: 36
      height: 36
    link: /es/guide/dtop
    linkText: Saber más
  - title: MCP para asistentes de IA
    details: Expón contenedores, logs y estadísticas mediante el Model Context Protocol para que tu agente de código depure contigo.
    icon:
      src: /icons/ai.svg
      width: 36
      height: 36
    link: /es/guide/mcp
    linkText: Saber más
  - title: Autoalojado y privado
    details: Se ejecuta en tu propia infraestructura, con autenticación simple o por proxy. Tus logs nunca salen de tu red.
    icon:
      src: /icons/locked.svg
      width: 36
      height: 36
    link: /es/guide/authentication
    linkText: Saber más
sourceHash: a11fae50734d
---
