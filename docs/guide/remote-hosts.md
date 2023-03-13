---
title: Remote Host Setup
---

# Remote Host Setup

Dozzle supports connecting to multiple remote hosts via `tcp://` using TLS and non-secured connections. Dozzle will need to have appropriate certs mounted to use secured connection. `ssh://` is not supported because Dozzle docker image does not ship with any ssh clients.

## Connecting to remote hosts

Remote hosts can be configured with `--remote-host` or `DOZZLE_REMOTE_HOST`. All certs must be mounted to `/certs` directory. The `/cert` directory expects to have `/certs/{ca,cert,key}.pem` or `/certs/{host}/{ca,cert,key}.pem` in case of multiple hosts.

Multiple `--remote-host` flags can be used to specify multiple hosts. However, using `DOZZLE_REMOTE_HOST` the value should be comma separated.

::: code-group

```sh [cli]
$ docker run -v /var/run/docker.sock:/var/run/docker.sock -v /path/to/certs:/certs -p 8080:8080 amir20/dozzle --remote-host tcp://167.99.1.1:2376 --remote-host tcp://167.99.1.2:2376
```

```yaml [docker-compose.yml]
version: "3"
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /path/to/certs:/certs
    ports:
      - 8080:8080
    environment:
      DOZZLE_REMOTE_HOST: tcp://167.99.1.1:2376,tcp://167.99.1.2:2376
```

:::

## Connecting with a socket proxy

If you are in a private network then you can use [Docker Socket Proxy](https://github.com/Tecnativa/docker-socket-proxy) which expose `docker.sock` file without the need of TLS. Dozzle will never try to write to Docker but it will need access to list APIs. The following command will start a proxy with minimal access.

```sh
docker container run --privileged -e CONTAINERS=1 -v /var/run/docker.sock:/var/run/docker.sock -p 2375:2375 tecnativa/docker-socket-proxy
```

Note that `CONTAINERS=1` is required to list running containers. `EVENTS` is also needed but it is enabled by default.

Running Dozzle without any certs should work. Here is an example:

```sh
docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --remote-host tcp://123.1.1.1:2375
```

::: warning
Exposing `docker.sock` publicly is not safe. Only use a proxy for an internal network where all clients are trusted.
:::
