---
title: Remote Host Setup
---

# Remote Host Setup <Badge type="warning" text="Deprecated" />

Dozzle supports connecting to remote Docker hosts. This is useful when running Dozzle in a container and you want to monitor a different Docker host.

However, with Dozzle agents, you can connect to remote hosts without exposing the Docker socket. See the [agent](/guide/agent) page for more information.

> [!WARNING]
> Remote hosts will soon be deprecated in favor of agents. Agents provide a more secure way to connect to remote hosts. See the [agent](/guide/agent) page for more information. If you want keep using remote hosts then follow this discussion on [GitHub](https://github.com/amir20/dozzle/issues/3066).

## Connecting to remote hosts with TLS

Remote hosts can be configured with `--remote-host` or `DOZZLE_REMOTE_HOST`. All certs must be mounted to `/certs` directory. The `/certs` directory expects to have `/certs/{ca,cert,key}.pem` or `/certs/{host}/{ca,cert,key}.pem` in case of multiple hosts.

Note the `{host}` value referred to here is the IP or FQDN configured and not the [optional label](#adding-labels-to-hosts).

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
docker container run --privileged -e CONTAINERS=1 -e INFO=1 -v /var/run/docker.sock:/var/run/docker.sock -p 2375:2375 tecnativa/docker-socket-proxy
```

> [!TIP]
> Using `CONTAINERS=1` is required to list running containers. `EVENTS` is also needed but it is enabled by default. `INFO=1` is optional but it will provide more information on host meta data.

Running Dozzle without any certs should work. Here is an example:

```sh
docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --remote-host tcp://123.1.1.1:2375
```

> [!WARNING]
> Docker Socket Proxy is not recommended for production use. It is only for private networks.

## Adding labels to hosts

`--remote-host` supports host labels by appending them to the connection string with `|`. For example, `--remote-host tcp://123.1.1.1:2375|foobar.com` will use foobar.com as the label in the UI. A full example of this using the CLI or compose are:

::: code-group

```sh
docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --remote-host tcp://123.1.1.1:2375|foobar.com
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
      DOZZLE_REMOTE_HOST: tcp://167.99.1.1:2376|foo.com,tcp://167.99.1.2:2376|bar.com
```

:::

## Changing localhost label

`localhost` is a special connection and uses different configuration than `--remote-host`. Changing the label for localhost can be done using the `--hostname` or `DOZZLE_HOSTNAME` env variable. See [hostname](/guide/hostname) page for examples on how to use this flag.
