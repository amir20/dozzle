---
title: Changing Application Base
---

# Changing Dozzle Base

Dozzle by default mounts to "/". This can be changed with the `--base` flag. For example, if you want to mount to "/foobar" then you can use `--base foobar` or the env variable `DOZZLE_BASE`.

::: code-group

```sh
docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --base foobar
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
      DOZZLE_BASE: foobar
```

:::

Dozzle will be available at `http://localhost:8080/foobar/`. This option rewrites all assets to `/foobar/{file.path}` and automatically redirects `/foobar` to `/foobar/`.

## Example with proxy

Here is an example with Nginx and proxy Dozzle with a different base:

```conf
location ^~ /foobar/ {
    include /config/nginx/proxy.conf;
    include /config/nginx/resolver.conf;
    set $upstream_app dozzle;
    set $upstream_port 8080;
    set $upstream_proto http;
    proxy_pass $upstream_proto://$upstream_app:$upstream_port;

    chunked_transfer_encoding off;
    proxy_buffering off;
    proxy_cache off;
}
```
