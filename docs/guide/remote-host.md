---
title: Remote Host Setup
---

# Remote Host Setup

Dozzle supports connecting to multiple remote hosts via `tcp://` using TLS and non-secured connections. Dozzle will need to have appropriate certs mounted to use secured connection. `ssh://` is not supported because Dozzle docker image does not ship with any ssh clients.

## Connecting remote hosts

Remote hosts can be configured with `--remote-host` or `DOZZLE_REMOTE_HOST`. All certs must be mounted to `/certs` directory. The `/cert` directory expects to have `/certs/{ca,cert,key}.pem` or `/certs/{host}/{ca,cert,key}.pem` in case of multiple hosts.

Multiple `--remote-host` flags can be used to specify multiple hosts.

::: code-group

```sh [cli]
$ docker run -v /var/run/docker.sock:/var/run/docker.sock -v /path/to/certs:/certs -p 8080:8080 amir20/dozzle --remote-host tcp://167.99.1.1:2376
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
