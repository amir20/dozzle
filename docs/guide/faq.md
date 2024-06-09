---
title: FAQ
---

# Frequently Asked Questions

## I installed Dozzle, but logs are slow or they never load. What do I do?

Dozzle uses Server Sent Events (SSE) which connects to a server using a HTTP stream without closing the connection. If any proxy tries to buffer this connection, then Dozzle never receives the data and hangs forever waiting for the reverse proxy to flush the buffer. Since version `1.23.0`, Dozzle sends the `X-Accel-Buffering: no` header which should stop reverse proxies buffering. However, some proxies may ignore this header. In those cases, you need to explicitly disable any buffering.

### Disabling buffering in nginx

Below is an example with nginx and using `proxy_pass` to disable buffering:

```
server {
    ...

    location / {
        proxy_pass                  http://<dozzle.container.ip.address>:8080;
    }

    location /api {
        proxy_pass                  http://<dozzle.container.ip.address>:8080;

        proxy_buffering             off;
        proxy_cache                 off;
    }
}
```

### Disabling compression in traefik

Traefik reverse proxy can be configures via middlewares to support compression. If implemented, the usual configuration looks like this:
```
http:
  middlewares:
    middlewares-compress:
      compress: {}
```

With this setup, you may find that certain containers do not show logs in dozzle anymore if you open dozzle via traefik (e.g. dozzle.mydomain.com). 
You will also note that the same dozzle instance does show the logs when accessed directly (e.g. localhost:8080).

Containers where this has been observed (non-exhaustive list) are: dozzle, homepage, glances, filebrowser.

To re-enable the logs to flow, exclude `text/event-stream` from the compression middleware:
```
http:
  middlewares:
    middlewares-compress:
      compress: 
        excludedContentTypes:
          - text/event-stream
```

## We have tools that uses Dozzle when a new container is created. How can I get a direct link to a container by name?

Dozzle has a special [route](https://github.com/amir20/dozzle/blob/master/assets/pages/show.vue) that can be used to search containers by name and then forward to that container. For example, if you have a container with name `"foo.bar"` and id `abc123`, you can send your users to `/show?name=foo.bar` which will be forwarded to `/container/abc123`.

## I installed Dozzle but memory consumption doesn't show up!

_This is an issue specific to ARM devices._

Dozzle uses the Docker API to gather information about the containers' memory usage. If the memory usage is not showing up, then it is likely that the Docker API is not returning the memory usage.

You can verify this by running docker info, and you should see the following:

```
WARNING: No memory limit support
WARNING: No swap limit support
```

In this case, you'll need to add the following line to your `/boot/cmdline.txt` file and reboot your device.

```
cgroup_enable=cpuset cgroup_enable=memory cgroup_memory=1
```
