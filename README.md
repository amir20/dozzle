[![Go Report Card](https://goreportcard.com/badge/github.com/amir20/dozzle)](https://goreportcard.com/report/github.com/amir20/dozzle)
[![Docker Pulls](https://img.shields.io/docker/pulls/amir20/dozzle.svg)](https://hub.docker.com/r/amir20/dozzle/)
[![Docker Size](https://images.microbadger.com/badges/image/amir20/dozzle.svg)](https://hub.docker.com/r/amir20/dozzle/)
[![Docker Version](https://images.microbadger.com/badges/version/amir20/dozzle.svg)](https://hub.docker.com/r/amir20/dozzle/)
![Test](https://github.com/amir20/dozzle/workflows/Test/badge.svg)

# Dozzle - [dozzle.dev](https://dozzle.dev/)

Dozzle is a simple, lightweight application that provides you with a web based interface to monitor your Docker container logs live. It doesn’t store log information, it is for live monitoring of your container logs only.

While dozzle should work for most, it is not meant to be a full logging solution. For enterprise applications, products like [Loggly](https://www.loggly.com), [Papertrail](https://papertrailapp.com) or [Kibana](https://www.elastic.co/products/kibana) are more suited.

Dozzle doesn't cost any money. Dozzle aims to stay simple, small and free.

![Image](https://github.com/amir20/dozzle/blob/master/.github/demo.gif?raw=true)

## Getting dozzle

Dozzle is a very small Docker container (4 MB compressed). Pull the latest release from the index:

    $ docker pull amir20/dozzle:latest

## Using dozzle

The simplest way to use dozzle is to run the docker container. Also, mount the Docker Unix socket with `--volume` to `/var/run/docker.sock`:

    $ docker run --name dozzle -d --volume=/var/run/docker.sock:/var/run/docker.sock -p 8888:8080 amir20/dozzle:latest

dozzle will be available at [http://localhost:8888/](http://localhost:8888/). You can change `-p 8888:8080` to any port. For example, if you want to view dozzle over port 4040 then you would do `-p 4040:8080`.

### With Docker swarm

    docker service create \
    --name=dozzle \
    --publish=8888:8080 \
    --constraint=node.role==manager \
    --mount=type=bind,src=/var/run/docker.sock,dst=/var/run/docker.sock \
    amir20/dozzle:latest

### With Docker compose

    version: "3"
    services:
      dozzle:
        container_name: dozzle
        image: amir20/dozzle:latest
        volumes:
          - /var/run/docker.sock:/var/run/docker.sock
        ports:
          - 9999:8080

#### Security

dozzle doesn't support authentication out of the box. You can control the device dozzle binds to by passing `--addr` parameter. For example,

    $ docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8888:1224 amir20/dozzle:latest --addr localhost:1224

will bind to `localhost` on port `1224`. You can then use a reverse proxy to control who can see dozzle.

If you wish to restrict the containers shown you can pass the `--filter` parameter. For example,

    $ docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8888:1224 amir20/dozzle:latest --filter name=foo

this would then only allow you to view containers with a name starting with "foo". You can use other filters like `status` as well, please check the official docker [command line docs](https://docs.docker.com/engine/reference/commandline/ps/#filtering) for available filters.

#### Changing base URL

dozzle by default mounts to "/". If you want to control the base path you can use the `--base` option. For example, if you want to mount at "/foobar",
then you can override by using `--base /foobar`. See env variables below for using `DOZZLE_BASE` to change this.

    $ docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle:latest --base /foobar

dozzle will be available at [http://localhost:8080/foobar/](http://localhost:8080/foobar/).

#### Environment variables and configuration

Dozzle follows the [12-factor](https://12factor.net/) model. Configurations can use the CLI flags or enviroment variables. The table below outlines all supported options and their respective env vars.

| Flag         | Env Variable         | Default |
| ------------ | -------------------- | ------- |
| `--addr`     | `DOZZLE_ADDR`        | `:8080` |
| `--base`     | `DOZZLE_BASE`        | `/`     |
| `--level`    | `DOZZLE_LEVEL`       | `info`  |
| n/a          | `DOCKER_API_VERSION` | not set |
| `--tailSize` | `DOZZLE_TAILSIZE`    | `300`   |
| `--filter`   | `DOZZLE_FILTER`      | `""`    |

## Troubleshooting and FAQs

<details>
 <summary>I installed Dozzle, but logs are slow or they never load. Help!</summary>

 Dozzle uses Server Sent Events (SSE) which connects to a server using a HTTP stream without closing the connection. If any proxy tries to buffer this connection, then Dozzle never receives the data and hangs forever waiting for the reverse proxy to flush the buffer.  Since version `1.23.0`, Dozzle sends the `X-Accel-Buffering: no` header which should stop reverse proxies buffering. However, some proxies may ignore this header. In those cases, you need to explicitly disable any buffering.

 Below is an example with nginx and using `proxy_pass` to disable buffering.

```
    server {
        ...

        location / {
            proxy_pass                  http://<dozzle.container.ip.address>:8080;
        }

        location /api {
            proxy_pass                  http://<dozzle.container.ip.address>:8080;

            proxy_buffering             off;
            proxy_cache                 off;
        }
    }

```
</details>

<details>
 <summary>What data does Dozzle collect?</summary>

 Dozzle does not collect any metrics or analytics. Dozzle has a [strict](https://github.com/amir20/dozzle/blob/master/routes.go#L33-L38) Content Security Policy which only allows the following policies:

 - Allow connect to `api.github.com` to fetch most recent version.
 - Allow fonts from `fonts.gstatic.com` and styles from `fonts.googleapis.com`
 - Only allow `<script>` and `<style>` files from `self`

 Dozzle opens all links with `rel="noopener"`.
</details>


## License

[MIT](LICENSE)

## Building

To Build and test locally:

1. Install NodeJs.
2. Install Go.
3. Install [reflex](https://github.com/cespare/reflex) with `get -u github.com/cespare/reflex` outside of dozzle.
4. Install node modules with `yarn`.
5. Do `yarn dev`

