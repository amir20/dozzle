# Dozzle - [dozzle.dev](https://dozzle.dev/)

Dozzle is a small lightweight application with a web based interface to monitor Docker logs. It doesn’t store any log files. It is for live monitoring of your container logs only.

https://user-images.githubusercontent.com/260667/227634771-9ebbe381-16a8-465a-b28a-450c5cd20c94.mp4

[![Go Report Card](https://goreportcard.com/badge/github.com/amir20/dozzle)](https://goreportcard.com/report/github.com/amir20/dozzle)
[![Docker Pulls](https://img.shields.io/docker/pulls/amir20/dozzle.svg)](https://hub.docker.com/r/amir20/dozzle/)
[![Docker Version](https://img.shields.io/docker/v/amir20/dozzle?sort=semver)](https://hub.docker.com/r/amir20/dozzle/)
![Test](https://github.com/amir20/dozzle/workflows/Test/badge.svg)

## Features

- Intelligent fuzzy search for container names 🤖
- Search logs using regex 🔦
- Small memory footprint 🏎
- Split screen for viewing multiple logs
- Download logs easy
- Live stats with memory and CPU usage
- Authentication with username and password 🚨

Dozzle should work for most. It has been tested with hundreds of containers. However, it doesn't support offline searching. Products like [Loggly](https://www.loggly.com), [Papertrail](https://papertrailapp.com) or [Kibana](https://www.elastic.co/products/kibana) are more suited for full search capabilities.

Dozzle doesn't cost any money and aims to focus on real-time debugging.

## Getting Started

Dozzle is a very small Docker container (4 MB compressed). Pull the latest release with:

    $ docker pull amir20/dozzle:latest

### Running Dozzle

The simplest way to use dozzle is to run the docker container. Also, mount the Docker Unix socket with `--volume` to `/var/run/docker.sock`:

    $ docker run --name dozzle -d --volume=/var/run/docker.sock:/var/run/docker.sock -p 8888:8080 amir20/dozzle:latest

Dozzle will be available at [http://localhost:8888/](http://localhost:8888/).

Here is the Docker Compose file:

    version: "3"
    services:
      dozzle:
        container_name: dozzle
        image: amir20/dozzle:latest
        volumes:
          - /var/run/docker.sock:/var/run/docker.sock
        ports:
          - 8080:8080

For advanced options like [authentication](https://dozzle.dev/guide/authentication), [remote hosts](https://dozzle.dev/guide/remote-hosts) or common [questions](https://dozzle.dev/guide/faq) see documentation at [dozzle.dev](https://dozzle.dev/guide/getting-started).

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
podman run --volume=/run/user/1000/podman/podman.sock:/var/run/docker.sock -d -p 8080:8080 amir20/dozzle:latest
```

## Security

Dozzle supports file based authentication and forward proxy like [Authelia](https://www.authelia.com/). These are documented at https://dozzle.dev/guide/authentication.

## Analytics collected

Dozzle collects anonymous user configurations using Google Analytics. Why? Dozzle is an open source project with no funding. As a result, there is no time to do user studies of Dozzle. Analytics is collected to prioritize features and fixes based on how people use Dozzle. This data is completely public and can be viewed live using [ Data Studio dashboard](https://datastudio.google.com/s/naeIu0MiWsY).

If you do not want to be tracked at all, see the `--no-analytics` flag below.

#### Environment variables and configuration

Dozzle follows the [12-factor](https://12factor.net/) model. Configurations can use the CLI flags or environment variables. The table below outlines all supported options and their respective env vars.

| Flag             | Env Variable           | Default |
| ---------------- | ---------------------- | ------- |
| `--addr`         | `DOZZLE_ADDR`          | `:8080` |
| `--base`         | `DOZZLE_BASE`          | `/`     |
| `--hostname`     | `DOZZLE_HOSTNAME`      | `""`    |
| `--level`        | `DOZZLE_LEVEL`         | `info`  |
| `--filter`       | `DOZZLE_FILTER`        | `""`    |
| `--username`     | `DOZZLE_USERNAME`      | `""`    |
| `--password`     | `DOZZLE_PASSWORD`      | `""`    |
| `--usernamefile` | `DOZZLE_USERNAME_FILE` | `""`    |
| `--passwordfile` | `DOZZLE_PASSWORD_FILE` | `""`    |
| `--no-analytics` | `DOZZLE_NO_ANALYTICS`  | false   |
| `--remote-host`  | `DOZZLE_REMOTE_HOST`   |         |

## License

[MIT](LICENSE)

## Building

To Build and test locally:

1. Install [NodeJs](https://nodejs.org/en/download/) and [pnpm](https://pnpm.io/installation).
2. Install [Go](https://go.dev/doc/install).
3. Install [reflex](https://github.com/cespare/reflex) with `go install github.com/cespare/reflex@latest`.
4. Install node modules `pnpm install`.
5. Do `pnpm dev`
