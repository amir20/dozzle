---
# https://vitepress.dev/reference/default-theme-home-page
layout: home

title: Home

hero:
  name: "Dozzle"
  tagline: Real-time logging and monitoring for Docker in the browser
  actions:
    - theme: brand
      text: Get Started
      link: /guide/getting-started
    - theme: alt
      text: View on GitHub
      link: https://github.com/amir20/dozzle

features:
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
  - title: No Database Required
    details: Streams logs directly from Docker, remaining lightweight without extra overhead or complexity.
    icon: 📦
---
