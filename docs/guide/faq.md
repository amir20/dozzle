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

Traefik reverse proxy can be configured via middlewares to support compression. If implemented, the usual configuration looks like this:

```
http:
  middlewares:
    middlewares-compress:
      compress: {}
```

With this setup, you may find that certain containers do not show logs in dozzle anymore if you open dozzle via traefik (e.g., dozzle.mydomain.com).
You will also note that the same dozzle instance does show the logs when accessed directly (e.g., localhost:8080).

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

## We have tools that use Dozzle when a new container is created. How can I get a direct link to a container by name?

Dozzle has a special [route](https://github.com/amir20/dozzle/blob/master/assets/pages/show.vue) that can be used to search containers by name and then forward to that container. For example, if you have a container with name `"foo.bar"` and id `abc123`, you can send your users to `/show?name=foo.bar` which will be forwarded to `/container/abc123`.

## I installed Dozzle but memory consumption doesn't show up!

_This is an issue specific to ARM devices._

Dozzle uses the Docker API to gather information about the containers' memory usage. If the memory usage is not showing up, then it is likely that the Docker API is not returning the memory usage.

You can verify this by running docker info, and you should see the following:

```
WARNING: No memory limit support
WARNING: No swap limit support
```

In this case, you'll need to add the following line to your `/boot/cmdline.txt` file and reboot your device:

```
cgroup_enable=cpuset cgroup_enable=memory cgroup_memory=1
```

## I am seeing duplicate hosts error in the logs. How do I fix it?

If you are seeing the following error in the logs, then you may have duplicate hosts configured with the same host ID:

```
time="2024-07-10T13:35:53Z" level=warning msg="duplicate host ID: *********, Endpoint: 1.1.1.1:7007 found, skipping"
```

Dozzle uses the Docker API to gather information about the hosts. Each host must have a unique ID. This ID is used to identify the host in the UI. In swarm mode, Dozzle uses the node ID from `docker system info` to identify the host. If you are not using swarm mode, then Dozzle will use the system ID from `docker system info` as the host ID.

Sometimes, VMs may be restored from backups with the same host ID. This can cause Dozzle to think that the host is already present and skip adding it to the list of hosts. To fix this, you need to remove `/var/lib/docker/engine-id` file. This file contains the host ID and is created when the Docker daemon starts.

## I am seeing host not found error in the logs. How do I fix it?

This should be mainly a Podman only error: Using Podman doesn't create an engine-id like Docker.
If you are using Docker check if the `engine-id` file exists with correct permissions in `/var/lib/docker` and has the UUID inside.

To resolve the error take following steps:

1. Create the folders: `mkdir -p /var/lib/docker`
2. Install uuidgen if necessary
3. Using uuidgen generate a UUID: `uuidgen > engine-id`

The engine-id file should now have a UUID inside.

An example setup for Ansible can be found in [Podman Infos](podman.md)

It might be necessary to clean up your existing Dozzle deployment under Podman, stop the container and remove the associated data (container/volumes). After that you can redeploy the Dozzle container and your logs should now show up.

## Why am I only seeing running containers? How do I see stopped containers?

By default, Dozzle only shows running containers. To see stopped containers, you need to enable the `Show Stopped Containers` option in the settings. This option is disabled by default to reduce the number of containers shown in the UI.

## Is there a way to sync my settings across multiple instances of Dozzle?

In single-user mode, Dozzle stores the settings in the browser's local storage. This means that the settings are only available on the browser where they were set. For Dozzle to enable syncing settings across multiple instances, it needs to know who the user is. In multi-user mode, Dozzle uses the user's username to store the settings on disk and sync them across multiple instances. This information is stored in `/data` directory. If you want to sync settings across multiple instances, you need to [enable](/guide/authentication) multi-user mode and provide a username.
