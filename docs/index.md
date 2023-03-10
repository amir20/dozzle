---
# https://vitepress.dev/reference/default-theme-home-page
layout: home

title: Home

hero:
  name: "Dozzle"
  tagline: A lightweight, web-based Docker log viewer that provides real-time monitoring and easy troubleshooting.
  actions:
    - theme: brand
      text: Get Started
      link: /guide/getting-started

features:
  - title: Real-time logging
    details: Allows you to view logs of other Docker containers in real-time. As new log entries are generated, they are streamed to the web interface without needing to refresh the page.
  - title: Lightweight
    details: An application written in Go consuming very little memory and CPU. It can be run alongside other containers without causing performance issues.
  - title: Multi-host Support
    details: Dozzle UI support connecting to multiple remote hosts with a simple drop down to choose between different hosts.
---
