<p align="center">
  <img src="assets/logo.svg" alt="Dozzle Logo" width="200"/>
</p>

# Dozzle - [dozzle.dev](https://dozzle.dev/)

Dozzle is a lightweight, web-based application for monitoring Docker logs in real time. It doesn't store any log files—it's designed purely for live log viewing.

https://github.com/user-attachments/assets/66a7b4b2-d6c9-4fca-ab04-aef6cd7c0c31

[![Docker Image Size (latest by date)](https://img.shields.io/docker/image-size/amir20/dozzle)](https://hub.docker.com/r/amir20/dozzle/)
[![Docker Pulls](https://img.shields.io/docker/pulls/amir20/dozzle.svg)](https://hub.docker.com/r/amir20/dozzle/)
[![Docker Version](https://img.shields.io/docker/v/amir20/dozzle?sort=semver)](https://hub.docker.com/r/amir20/dozzle/)
![Test](https://github.com/amir20/dozzle/workflows/Test/badge.svg)

> [!NOTE]
> If you like Dozzle, check out [`dtop`](https://github.com/amir20/dtop), a top-like application for monitoring Docker containers. It integrates with Dozzle to link directly to container logs.

## Features

- Intelligent fuzzy search for container names
- Search logs using regex
- Search logs using [SQL queries](https://dozzle.dev/guide/sql-engine)
- Small memory footprint
- Split screen for viewing multiple logs
- Live stats with memory and CPU usage
- Multi-user [authentication](https://dozzle.dev/guide/authentication) with support for forward proxy authorization
- [Swarm mode](https://dozzle.dev/guide/swarm-mode) support
- [Agent mode](https://dozzle.dev/guide/agent) for monitoring multiple Docker hosts
- Dark mode

Dozzle has been tested with hundreds of containers. However, it doesn't support offline searching. Products like [Loggly](https://www.loggly.com), [Papertrail](https://papertrailapp.com), or [Kibana](https://www.elastic.co/products/kibana) are better suited for full search capabilities.

## Getting Started

Dozzle is a small container (7 MB compressed). Pull the latest release with:

    $ docker pull amir20/dozzle:latest

### Running Dozzle

The simplest way to use Dozzle is to run the Docker container. Mount the Docker Unix socket with `--volume` to `/var/run/docker.sock`:

    $ docker run --name dozzle -d --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle:latest

Dozzle will be available at [http://localhost:8080/](http://localhost:8080/).

Here is a Docker Compose example:

    services:
      dozzle:
        container_name: dozzle
        image: amir20/dozzle:latest
        volumes:
          - /var/run/docker.sock:/var/run/docker.sock
        ports:
          - 8080:8080

For advanced options like [authentication](https://dozzle.dev/guide/authentication), [remote hosts](https://dozzle.dev/guide/remote-hosts), or common [questions](https://dozzle.dev/guide/faq), see the documentation at [dozzle.dev](https://dozzle.dev/guide/getting-started).

## Swarm Mode

Dozzle works with Docker Swarm. You can run Dozzle as a global service:

    $ docker service create --name dozzle --env DOZZLE_MODE=swarm --mode global --mount type=bind,source=/var/run/docker.sock,target=/var/run/docker.sock -p 8080:8080 amir20/dozzle:latest

See the [Swarm Mode](https://dozzle.dev/guide/swarm-mode) documentation for more details.

## Agent Mode

Dozzle can monitor multiple Docker hosts. Run Dozzle in agent mode with:

    $ docker run -v /var/run/docker.sock:/var/run/docker.sock -p 7007:7007 amir20/dozzle:latest agent

See the [Agent Mode](https://dozzle.dev/guide/agent) documentation for more details.

## Technical Details

Dozzle uses automatic API negotiation, which works with most Docker configurations. Dozzle also works with [Colima](https://github.com/abiosoft/colima) and [Podman](https://podman.io/).

### Installation on Podman

By default, Podman doesn't have a background process, but you can enable the remote socket for Dozzle to work.

First, verify if your Podman installation has the remote socket enabled:

```
podman info
```

If you see output like this under the remote socket key, it's already enabled:

```
  remoteSocket:
    exists: true
    path: /run/user/1000/podman/podman.sock
```

If it's not enabled, follow [this tutorial](https://github.com/containers/podman/blob/main/docs/tutorials/socket_activation.md) to enable it.

Once the Podman remote socket is enabled, you can run Dozzle:

```
podman run --volume=/run/user/1000/podman/podman.sock:/var/run/docker.sock -d -p 8080:8080 docker.io/amir20/dozzle:latest
```

Additionally, you need to create a fake engine-id to prevent `host not found` errors. Podman doesn't generate an engine-id like Docker does, due to its daemonless architecture.

Create a file named `engine-id` under `/var/lib/docker`. On a system with Podman, you'll need to create the folder path as well. Place a UUID inside the file, for example using `uuidgen > engine-id`. The file should contain an identifier like: `b9f1d7fc-b459-4b6e-9f7a-e3d1cd2e14a9`.

For more details, see [Podman Info](docs/guide/podman.md) or the [FAQ](docs/guide/faq.md#i-am-seeing-host-not-found-error-in-the-logs-how-do-i-fix-it).

## Security

Dozzle supports file-based authentication and forward proxy authentication with tools like [Authelia](https://www.authelia.com/). See the documentation at https://dozzle.dev/guide/authentication.

## Analytics

Dozzle collects anonymous user configurations using Google Analytics. Why? Dozzle is an open source project with no funding, so there's no time for formal user studies. Analytics help prioritize features and fixes based on how people use Dozzle. This data is completely public and can be viewed live on the [Data Studio dashboard](https://datastudio.google.com/s/naeIu0MiWsY).

To disable analytics, use the `--no-analytics` flag.

## Environment Variables and Configuration

Dozzle follows the [12-factor](https://12factor.net/) model. Configuration can be done via CLI flags or environment variables. See the documentation at [dozzle.dev/guide/supported-env-vars](https://dozzle.dev/guide/supported-env-vars) for more details.

## Support

There are many ways to support Dozzle:

- Use it! Write about it! Star it! If you love Dozzle, drop me a line and tell me what you love.
- Blog about Dozzle to spread the word. If you're good at writing, send PRs to improve the documentation at [dozzle.dev](https://dozzle.dev/).
- Sponsor my work at https://www.buymeacoffee.com/amirraminfar

<a href="https://www.buymeacoffee.com/amirraminfar" target="_blank"><img src="https://cdn.buymeacoffee.com/buttons/v2/default-yellow.png" alt="Buy Me A Coffee" style="height: 60px !important;width: 217px !important;" ></a>

## License

[MIT](LICENSE)

## Building

To build and test locally:

1. Install [Node.js](https://nodejs.org/en/download/) and [pnpm](https://pnpm.io/installation).
2. Install [Go](https://go.dev/doc/install).
3. Install [protoc](https://grpc.io/docs/protoc-installation/).
4. Install Go tools with `go install tool`.
5. Install Node modules with `pnpm install`.
6. Run `make dev` to start a development server with hot reload.
