[![Go Report Card](https://goreportcard.com/badge/github.com/amir20/dozzle)](https://goreportcard.com/report/github.com/amir20/dozzle)
[![Build Status](https://wdp9fww0r9.execute-api.us-west-2.amazonaws.com/production/badge/amir20/dozzle)](https://wdp9fww0r9.execute-api.us-west-2.amazonaws.com/production/results/amir20/dozzle)
[![Docker Pulls](https://img.shields.io/docker/pulls/amir20/dozzle.svg)](https://hub.docker.com/r/amir20/dozzle/)
[![Docker Size](https://images.microbadger.com/badges/image/amir20/dozzle.svg)](https://hub.docker.com/r/amir20/dozzle/)
[![Docker Version](https://images.microbadger.com/badges/version/amir20/dozzle.svg)](https://hub.docker.com/r/amir20/dozzle/)

# Dozzle - [dozzle.dev](https://dozzle.dev/)

Dozzle is a log viewer for Docker. It's free. It's small. And it's right in your browser. Oh, did I mention it is also real-time?

While dozzle should work for most, it is not meant to be a full logging solution. For enterprise use, I recommend you look at [Loggly](https://www.loggly.com), [Papertrail](https://papertrailapp.com) or [Kibana](https://www.elastic.co/products/kibana).

But if you don't want to pay for those services, then you are in luck! Dozzle will be able to capture all logs from your containers and send them in real-time to your browser. Installation is also very easy.

![Image](demo.gif)

## Getting dozzle

Dozzle is a very small Docker container (4 MB compressed). Pull the latest release from the index:

    $ docker pull amir20/dozzle:latest

## Using dozzle

The simplest way to use dozzle is to run the docker container. Also, mount the Docker Unix socket with `-volume` to `/var/run/docker.sock`:

    $ docker run --name dozzle -d --volume=/var/run/docker.sock:/var/run/docker.sock -p 8888:8080 amir20/dozzle:latest

dozzle will be available at [http://localhost:8888/](http://localhost:8888/). You can change `-p 8888:8080` to any port. For example, if you want to view dozzle over port 4040 then you would do `-p 4040:8080`.

## Docker swarm deploy

     docker service create \
    --name=dozzle \
    --publish=8888:8080 \
    --constraint=node.role==manager \
    --mount=type=bind,src=/var/run/docker.sock,dst=/var/run/docker.sock \
    amir20/dozzle:latest

#### Security

dozzle doesn't support authentication out of the box. You can control the device dozzle binds to by passing `--addr` parameter. For example,

    $ docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8888:1224 amir20/dozzle:latest --addr localhost:1224

will bind to `localhost` on port `1224`. You can then use a reverse proxy to control who can see dozzle.

#### Changing base URL

dozzle by default mounts to "/". If you want to control the base path you can use the `--base` option. For example, if you want to mount at "/foobar",
then you can override by using `--base /foobar`. See env variables below for using `DOZZLE_BASE` to change this.

    $ docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle:latest --base /foobar

dozzle will be available at [http://localhost:8080/foobar/](http://localhost:8080/foobar/).


#### Environment variables and configuration

Dozzle follows the [12-factor](https://12factor.net/) model. Configurations can use the CLI flags or enviroment variables. The table below outlines all supported options and their respective env vars.

| Flag | Env Variable | Default |
| --- | --- | --- |
| `--addr` | `DOZZLE_ADDR` | `:8080` |
| `--base` | `DOZZLE_BASE` | `/` |
| `--level` | `DOZZLE_LEVEL` | `info` |
| n/a | `DOCKER_API_VERSION` | `1.38` |
| `--tailSize` | `DOZZLE_TAILSIZE` | `300` |

## License

[MIT](LICENSE)
