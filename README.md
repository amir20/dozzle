[![Go Report Card](https://goreportcard.com/badge/github.com/amir20/dozzle)](https://goreportcard.com/report/github.com/amir20/dozzle)
[![Docker Pulls](https://img.shields.io/docker/pulls/amir20/dozzle.svg)](https://hub.docker.com/r/amir20/dozzle/)
[![Docker Size](https://images.microbadger.com/badges/image/amir20/dozzle.svg)](https://hub.docker.com/r/amir20/dozzle/)
[![Docker Version](https://images.microbadger.com/badges/version/amir20/dozzle.svg)](https://hub.docker.com/r/amir20/dozzle/)
![Test](https://github.com/amir20/dozzle/workflows/Test/badge.svg)

# Dozzle - [dozzle.dev](https://dozzle.dev/)

Dozzle is a real-time log viewer for Docker. It's free. It's small. And it's in your browser.

While dozzle should work for most, it is not meant to be a full logging solution. For enterprise use, I recommend you look at [Loggly](https://www.loggly.com), [Papertrail](https://papertrailapp.com) or [Kibana](https://www.elastic.co/products/kibana).

But if you don't want to pay for these services, then Dozzle can help! Dozzle will be able to capture all logs from your containers and send them in real-time to your browser. Installation is also very easy. Dozzle is not a database. It does not store or save any logs. You can only see live logs while using Dozzle.

![Image](.github/demo.gif)

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
| `--showAll`  | `DOZZLE_SHOWALL`     | `false` |
| n/a          | `DOCKER_API_VERSION` | not set |
| `--tailSize` | `DOZZLE_TAILSIZE`    | `300`   |
| `--filter`   | `DOZZLE_FILTER`      | `""`    |

## Troubleshooting

### Sample Nginx config

```

    server {
        listen                          80;
        server_name                     <example.com>;
        return                          301 https://<example.com>$request_uri;
    }

    server {
        listen                          443 ssl http2;
        server_name                     <example.com>;

        ssl_certificate                 </path/to/your/certificate>;
        ssl_certificate_key             </path/to/your/key>;

        location / {
            proxy_pass                  http://<dozzle.container.ip.address>:8080;
        }

        location /api {
            proxy_pass                  http://<dozzle.container.ip.address>:8080;

            proxy_http_version          1.1;
            proxy_set_header            Connection "";
            proxy_buffering             off;
            proxy_cache                 off;

            chunked_transfer_encoding   off;
        }
    }

```

## License

[MIT](LICENSE)

## Building

To Build and test locally:

1. Install NodeJs.
2. Install Go.
3. Globally install [packr utility](https://github.com/gobuffalo/packr) with `go get -u github.com/gobuffalo/packr/packr` outside of dozzle directory.
4. Install [reflex](https://github.com/cespare/reflex) with `get -u github.com/cespare/reflex` outside of dozzle.
5. Install node modules with `npm install`.
6. Do `npm start`

Instructions for Github actions can be found [here](.github/goreleaser/Dockerfile) which build and tests Dozzle.
