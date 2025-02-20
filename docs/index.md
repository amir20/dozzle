---
# https://vitepress.dev/reference/default-theme-home-page
layout: home

title: Home

hero:
  name: "Dozzle"
  tagline: Simple Container Monitoring and Logging
  actions:
    - theme: brand
      text: Get Started
      link: /guide/getting-started
    - theme: alt
      text: View on GitHub
      link: https://github.com/amir20/dozzle
    - theme: alt
      text: Support 🙏🏼
      link: https://www.buymeacoffee.com/amirraminfar

features:
  - title: Self Hosted
    details: Dozzle is a self-hosted application that runs in your own infrastructure, ensuring your logs remain private and secure.
    icon: 🏠
  - title: Real-time Logging & Monitoring
    details: Captures real-time Docker container logs, enabling quick and efficient issue diagnosis.
    icon: 🚀
  - title: Docker Swarm Support
    details: Supports Docker services, allowing you to monitor logs from multiple nodes in a single interface.
    link: /guide/swarm-mode
    linkText: Learn More
    icon: 🐳
  - title: Multi-host Support
    details: UI support connecting to multiple remote hosts with a simple drop down to choose between different hosts.
    link: /guide/remote-hosts
    linkText: Learn More
    icon: 🌐
  - title: SQL Engine
    details: Use SQL queries to analyze logs inside your browser with WebAssembly and DuckDB.
    icon: 📊
    linkText: Learn More
    link: /guide/sql-engine
  - title: Secured Agents
    details: Connect to remote hosts securely with agents, providing a more secure way to connect to remote hosts.
    icon: 🔒
    link: /guide/agent
    linkText: Learn More
  - title: Easy to Use
    details: Simple to set up and use, with a clean and intuitive interface that requires no additional configuration.
    icon: 🎨
  - title: Sponsored by Docker OSS
    details: Dozzle is open source and free to use, with the source code available on GitHub.
    icon: 📜
    link: https://hub.docker.com/r/amir20/dozzle
    linkText: Docker Hub
---
