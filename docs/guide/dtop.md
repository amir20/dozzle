---
title: Introducing dtop
---

# What is dtop?

`dtop` is a command-line companion to Dozzle that provides a real-time terminal view of the Docker containers running on your system. Think of it as a richer `docker ps` you can leave open in a tmux pane — and when you need the full log history, search, or charts, `dtop` lets you jump straight into Dozzle.

It connects to Docker hosts via `ssh`, `tcp`, or a local `unix socket`, making it well suited for the same multi-host setups Dozzle supports.

![dtop screenshot](https://github.com/amir20/dtop/raw/master/demo.gif)

## Installation

Install with Homebrew:

```bash
brew install dtop
```

Or run it via Docker without installing anything:

```bash
docker run -v /var/run/docker.sock:/var/run/docker.sock -it ghcr.io/amir20/dtop:latest
```

Full installation instructions can be found at [https://github.com/amir20/dtop](https://github.com/amir20/dtop?tab=readme-ov-file#installation).

## Project Status

`dtop` is a new project and not feature rich as Dozzle. However, I am actively working on adding more features. I use it personally to monitor all my containers across multiple hosts on the command line. If you have suggestions then please open issues at [https://github.com/amir20/dtop/issues](https://github.com/amir20/dtop/issues).
