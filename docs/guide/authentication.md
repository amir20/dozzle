---
title: Authentication
---

# Setting Up Authentication

Dozzle supports a very simple authentication out of the box with just username and password. You should deploy using SSL to keep the credentials safe. See configuration to use `--username` and `--password`. You can also use docker secrets `--usernamefile` and `--passwordfile`.

::: code-group

```sh [cli]
$ docker run -v /var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --username amirraminfar --password supersecretpassword
```

```yaml [docker-compose.yml]
version: "3"
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    ports:
      - 8080:8080
    environment:
      DOZZLE_USERNAME: amirraminfar
      DOZZLE_PASSWORD: supersecretpassword
```

:::
