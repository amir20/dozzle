---
title: Introducing dtop
---

# What is dtop?

`dtop` is a command-line tool that provides a real-time view of the Docker containers running on your system. It is a lightweight alternative to the `docker ps` command, and it is designed to be used in a terminal or command prompt. `dtop` supports connecting to multiple hosts via `ssh`, `tcp` or `unix socket`. It also integrates with Dozzle by providing a quick way to open logs quickly.

![dtop screenshot](https://github.com/amir20/dtop/raw/master/demo.gif)

## Installation

Full installation instructions can be found at [https://github.com/amir20/dtop](https://github.com/amir20/dtop?tab=readme-ov-file#installation). You can also install using brew:

```bash
brew install amir20/homebrew-dtop/dtop
```

## Project Status

`dtop` is a new project and not feature rich as Dozzle. However, I am actively working on adding more features. I use it personally to monitor all my containers across multiple hosts on the command line. If you have suggestions then please open issues at [https://github.com/amir20/dtop/issues](https://github.com/amir20/dtop/issues).
