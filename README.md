[![Go Report Card](https://goreportcard.com/badge/github.com/amir20/dozzle)](https://goreportcard.com/report/github.com/amir20/dozzle)
[![Docker Pulls](https://img.shields.io/docker/pulls/amir20/dozzle.svg)](https://hub.docker.com/r/amir20/dozzle/)
[![Docker Size](https://images.microbadger.com/badges/image/amir20/dozzle.svg)](https://hub.docker.com/r/amir20/dozzle/)
[![Docker Version](https://images.microbadger.com/badges/version/amir20/dozzle.svg)](https://hub.docker.com/r/amir20/dozzle/)

# dozzle

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

#### Security

dozzle doesn't support authentication out of the box. You can control the device dozzle binds to by passing `--addr` parameter. For example,

    $ docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8888:1224 amir20/dozzle:latest --addr localhost:1224

will bind to `localhost` on port `1224`. You can then use a reverse proxy to control who can see dozzle.

#### Changing base URL

dozzle by default mounts to "/". If you want to control the base path you can use the `--base` option. For example, if you want to mount at "/foobar",
then you can override by using `--base /foobar`.

    $ docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle:latest --base /foobar

dozzle will be available at [http://localhost:8080/foobar/](http://localhost:8080/foobar/).


#### Environment variable, DOCKER_API_VERSION

If you see

    2018/10/31 08:53:17 Error response from daemon: client version 1.40 is too new. Maximum supported API version is 1.38

Then you need to modify `DOCKER_API_VERSION` to let dozzle know which version of the API is supported. By default, `DOCKER_API_VERSION=1.38` and you can change it by passing `-e` flag. For example, this would change the `DOCKER_API_VERSION` to `1.20`

    $ docker run --volume=/var/run/docker.sock:/var/run/docker.sock -e DOCKER_API_VERSION=1.20 -p 8888:8080 amir20/dozzle:latest

If you are not sure what to set `DOCKER_API_VERSION` then run `docker version` which will show supported API version.

## License

[MIT](LICENSE)
