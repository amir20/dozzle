---
title: Changing Application Base
---

# Changing Dozzle Base

Dozzle by default mounts to "/". This can be changed with the `--base` flag. For example, if you want to mount to "/foobar" then you can use `--base /foobar` or the env variable `DOZZLE_BASE`.

::: code-group

```sh
docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --base /foobar
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    ports:
      - 8080:8080
    environment:
      DOZZLE_BASE: /foobar
```

:::

Dozzle will be available at `http://localhost:8080/foobar/`. This option rewrites all assets to `/foobar/{file.path}` and automatically redirects `/foobar` to `/foobar/`.

## Example with Proxy

Here is an example with Nginx to proxy Dozzle with a different base:

```nginx
location ^~ /foobar/ {
    set $upstream_app dozzle;
    set $upstream_port 8080;
    set $upstream_proto http;
    proxy_pass $upstream_proto://$upstream_app:$upstream_port;

    chunked_transfer_encoding off;
    proxy_buffering off;
    proxy_cache off;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
}
```
