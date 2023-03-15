---
title: Getting Started
---

# Getting Started

This section will help you to setup Dozzle locally. Dozzle can also be used to connect to remote hosts via `tcp://` and tls. See remote host if you want to connect to other hosts.

## Using Docker CLI

The easiest way to setup Dozzle is to use the CLI and mount `docker.sock` file. This file is usually located at `/var/run/docker.sock` and can be mounted with the `--volume` flag. You also need to expose the port to view Dozzle. By default, Dozzle listens on port 8080, but you can change the external port using `-p`.

```sh
docker run --detach --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle
```

## Using Docker Compose

Docker compose makes it easier to configure Dozzle as part of an existing configuration.

```yaml
version: "3"
services:
  dozzle:
    container_name: dozzle
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    ports:
      - 9999:8080
```
