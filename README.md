# Dozzle - [dozzle.dev](https://dozzle.dev/)

Dozzle is a small lightweight application with a web based interface to monitor Docker logs. It doesn’t store any log files. It is for live monitoring of your container logs only.

https://github.com/user-attachments/assets/66a7b4b2-d6c9-4fca-ab04-aef6cd7c0c31

[![Docker Image Size (latest by date)](https://img.shields.io/docker/image-size/amir20/dozzle)](https://hub.docker.com/r/amir20/dozzle/)
[![Docker Pulls](https://img.shields.io/docker/pulls/amir20/dozzle.svg)](https://hub.docker.com/r/amir20/dozzle/)
[![Docker Version](https://img.shields.io/docker/v/amir20/dozzle?sort=semver)](https://hub.docker.com/r/amir20/dozzle/)
![Test](https://github.com/amir20/dozzle/workflows/Test/badge.svg)

## Features

- Intelligent fuzzy search for container names 🤖
- Search logs using regex 🔦
- Search logs using [SQL queries](https://dozzle.dev/guide/sql-engine) 📊
- Small memory footprint 🏎
- Split screen for viewing multiple logs
- Live stats with memory and CPU usage
- Multi-user [authentication](https://dozzle.dev/guide/authentication) with support for proxy forward authorization 🚨
- [Swarm](https://dozzle.dev/guide/swarm-mode) mode support 🐳
- [Agent](https://dozzle.dev/guide/agent) mode for monitoring multiple Docker hosts 🕵️‍♂️
- Dark mode 🌙

Dozzle has been tested with hundreds of containers. However, it doesn't support offline searching. Products like [Loggly](https://www.loggly.com), [Papertrail](https://papertrailapp.com) or [Kibana](https://www.elastic.co/products/kibana) are more suited for full search capabilities.

## Getting Started

Dozzle is a small container (7 MB compressed). Pull the latest release with:

    $ docker pull amir20/dozzle:latest

### Running Dozzle

The simplest way to use dozzle is to run the docker container. Also, mount the Docker Unix socket with `--volume` to `/var/run/docker.sock`:

    $ docker run --name dozzle -d --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle:latest

Dozzle will be available at [http://localhost:8080/](http://localhost:8080/).

Here is the Docker Compose file:

    services:
      dozzle:
        container_name: dozzle
        image: amir20/dozzle:latest
        volumes:
          - /var/run/docker.sock:/var/run/docker.sock
        ports:
          - 8080:8080

For advanced options like [authentication](https://dozzle.dev/guide/authentication), [remote hosts](https://dozzle.dev/guide/remote-hosts) or common [questions](https://dozzle.dev/guide/faq) see documentation at [dozzle.dev](https://dozzle.dev/guide/getting-started).

## Swarm Mode

Dozzle works with Docker Swarm mode. You can run Dozzle as a global service with:

    $ docker service create --name dozzle --env DOZZLE_MODE=swarm --mode global --mount type=bind,source=/var/run/docker.sock,target=/var/run/docker.sock -p 8080:8080 amir20/dozzle:latest

See the [Swarm Mode](https://dozzle.dev/guide/swarm-mode) documentation for more details.

## Agent Mode

Dozzle can be used to monitor multiple Docker hosts. You can run Dozzle in agent mode with:

    $ docker run -v /var/run/docker.sock:/var/run/docker.sock -p 7007:7007 amir20/dozzle:latest agent

See the [Agent Mode](https://dozzle.dev/guide/agent) documentation for more details.

## Technical Details

Dozzle users automatic API negotiation which works with most Docker configurations. Dozzle also works with [Colima](https://github.com/abiosoft/colima) and [Podman](https://podman.io/).

### Installation on podman

By default Podman doesn't have a background process but you can enable this for Dozzle to work.

Verify first if your podman installation has enabled remote socket:

```
podman info
```

When you get under the key remote socket output like this, its already enabled:

```
  remoteSocket:
    exists: true
    path: /run/user/1000/podman/podman.sock
```

If it's not enabled please follow [this tutorial](https://github.com/containers/podman/blob/main/docs/tutorials/socket_activation.md) to enable it.

Once you have the podman remote socket you can run Dozzle on podman.

```
podman run --volume=/run/user/1000/podman/podman.sock:/var/run/docker.sock -d -p 8080:8080 docker.io/amir20/dozzle:latest
```

Additionally you have to create a fake engine-id to prevent `host not found` errors. Podman doesn't generate an engine-id like Docker by itself due to its daemonless architecture.

Under `/var/lib/docker` create a file named `engine-id`. On a system with Podman you will have to create the folder path as well. Inside the file place the UUID, for instance using `uuidgen > engine-id`. After that the file should have an identifier that looks like this: `b9f1d7fc-b459-4b6e-9f7a-e3d1cd2e14a9`.

For more details check [Podman Infos](docs/guide/podman.md) or the [FAQ](docs/guide/faq.md#i-am-seeing-host-not-found-error-in-the-logs-how-do-i-fix-it)

## Security

Dozzle supports file based authentication and forward proxy like [Authelia](https://www.authelia.com/). These are documented at https://dozzle.dev/guide/authentication.

## Analytics collected

Dozzle collects anonymous user configurations using Google Analytics. Why? Dozzle is an open source project with no funding. As a result, there is no time to do user studies of Dozzle. Analytics is collected to prioritize features and fixes based on how people use Dozzle. This data is completely public and can be viewed live using [ Data Studio dashboard](https://datastudio.google.com/s/naeIu0MiWsY).

If you do not want to be tracked at all, see the `--no-analytics` flag below.

## Environment variables and configuration

Dozzle follows the [12-factor](https://12factor.net/) model. Configurations can use the CLI flags or environment variables. See documentation at [https://dozzle.dev/guide/supported-env-vars](https://dozzle.dev/guide/supported-env-vars) for more details.

## Support

There are many ways you can support Dozzle:

- Use it! Write about it! Star it! If you love Dozzle, drop me a line and tell me what you love.
- Blog about Dozzle to spread the word. If you are good at writing send PRs to improve the documentation at [dozzle.dev](https://dozzle.dev/)
- Sponsor my work at https://www.buymeacoffee.com/amirraminfar

<a href="https://www.buymeacoffee.com/amirraminfar" target="_blank"><img src="https://cdn.buymeacoffee.com/buttons/v2/default-yellow.png" alt="Buy Me A Coffee" style="height: 60px !important;width: 217px !important;" ></a>

## License

[MIT](LICENSE)

## Building

To build and test locally:

1. Install [NodeJs](https://nodejs.org/en/download/) and [pnpm](https://pnpm.io/installation).
2. Install [Go](https://go.dev/doc/install).
3. Install tools with `make tools`.
4. Install node modules `pnpm install`.
5. Run `make dev` to start a development server with hot reload.
