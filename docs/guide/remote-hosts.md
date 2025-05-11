---
title: Remote Host Setup
---

# Remote Host Setup

Dozzle supports connecting to remote Docker hosts. This is useful when running Dozzle in a container and you want to monitor a different Docker host.

However, with Dozzle agents, you can connect to remote hosts without exposing the Docker socket. See the [agent](/guide/agent) page for more information.

Dozzle agents remove the need to remotely expose the Docker socket but cannot be uses with a Docker Socket proxy inside the Dozzle agent stack. If you wish to use a Socket Proxy on it's own without an agent see the [connecting with a socket proxy](#connecting-with-a-socket-proxy) section.

> [!WARNING]
> Remote hosts have been replaced with agents. Agents provide a more secure way to connect to remote hosts. Although remote hosts are still supported, it is recommended to use agents. See the [agent](/guide/agent) page for more information and examples. For comparison, see the [comparing agents with remote connections](/guide/agent#comparing-agents-with-remote-connection) section. I won't be able to investigate user's issues with remote hosts as it is very time consuming.

## Connecting to Remote Hosts with TLS

Remote hosts can be configured with `--remote-host` or `DOZZLE_REMOTE_HOST`. All certificates must be mounted to `/certs` directory. The `/certs` directory expects to have `/certs/{ca,cert,key}.pem` or `/certs/{host}/{ca,cert,key}.pem` in case of multiple hosts.

Note the `{host}` value referred to here is the IP or FQDN configured and not the [optional label](#adding-labels-to-hosts).

Multiple `--remote-host` flags can be used to specify multiple hosts. However, when using `DOZZLE_REMOTE_HOST`, the value should be comma-separated.

::: code-group

```sh [cli]
$ docker run -v /var/run/docker.sock:/var/run/docker.sock -v /path/to/certs:/certs -p 8080:8080 amir20/dozzle --remote-host tcp://167.99.1.1:2376 --remote-host tcp://167.99.1.2:2376
```

```yaml [docker-compose.yml]
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

## Connecting with a Socket Proxy

If you are in a private network, then you can use [Docker Socket Proxy](https://github.com/Tecnativa/docker-socket-proxy) which exposes `docker.sock` file without the need for TLS. This will remove the need for a Dozzle agent and Dozzle will connect directly to the Socket Proxy instead. Dozzle will never try to write to Docker but it will need access to list APIs. The following command will start a proxy with minimal access:

```sh
$ docker container run --privileged -e CONTAINERS=1 -e INFO=1 -v /var/run/docker.sock:/var/run/docker.sock -p 2375:2375 tecnativa/docker-socket-proxy
```

> [!TIP]
> Using `CONTAINERS=1` is required to list running containers. `EVENTS` is also needed but it is enabled by default. `INFO=1` is needed to list system information.

Running Dozzle without any certificates should work. Here is an example:

::: code-group

```sh [cli]
$ docker run -p 8080:8080 amir20/dozzle --remote-host tcp://123.1.1.1:2375
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    ports:
      - 8080:8080
    environment:
      DOZZLE_REMOTE_HOST: tcp://123.1.1.1:2375
```

:::

When using remote host, mounting `/var/run/docker.sock` is optional. You need to have at least one remote host to connect to.

> [!WARNING]
> Docker Socket Proxy exposes the Docker API to the internet. This can be a security risk if not properly secured.

## Adding Labels to Hosts

`--remote-host` supports host labels by appending them to the connection string with `|`. For example, `--remote-host tcp://123.1.1.1:2375|foobar.com` will use foobar.com as the label in the UI. A full example using the CLI or compose:

::: code-group

```sh
docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --remote-host tcp://123.1.1.1:2375|foobar.com
```

```yaml [docker-compose.yml]
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

> [!WARNING]
> Dozzle uses the Docker API to gather information about hosts. Each agent needs a unique host ID. They use Docker's system ID or node ID to identify the host. If you are using swarm, then the node ID is used. If you don't see all hosts, then you may have duplicate hosts configured that have the same host ID. To fix this, remove `/var/lib/docker/engine-id` file. See [FAQ](/guide/faq#i-am-seeing-duplicate-hosts-error-in-the-logs-how-do-i-fix-it) for more information.

## Changing Localhost Label

`localhost` is a special connection and uses different configuration than `--remote-host`. Changing the label for localhost can be done using the `--hostname` or `DOZZLE_HOSTNAME` env variable. See [hostname](/guide/hostname) page for examples on how to use this flag.
