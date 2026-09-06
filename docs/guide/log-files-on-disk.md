---
title: Following Log Files on Disk
---

# Following Log Files on Disk

Some containers write logs to files instead of `stdout` or `stderr`. Dozzle can only read what Docker itself captures, which is `stdout` and `stderr`, the same as `docker logs`. Files inside a container are not visible to other containers, so Dozzle has no way to reach them.

## Log to Streams Instead

The best fix is to stop writing to files. Most applications have a config option to log to the console, and the [twelve factor app](https://12factor.net/logs) explains why that is the right default.

If the application cannot be configured, symlink the log file to the container's stdout in your `Dockerfile`. This is what the official nginx image does:

```dockerfile
RUN ln -sf /dev/stdout /var/log/nginx/access.log \
    && ln -sf /dev/stderr /var/log/nginx/error.log
```

## Tailing a File With a Sidecar

When neither is possible, run a small Alpine container that tails the file and lets Docker capture the output. Dozzle then shows it like any other container.

::: code-group

```sh [docker run]
docker run -d \
  --name system-log \
  --label dev.dozzle.name=system-log \
  --network none \
  --restart unless-stopped \
  --log-opt max-size=10m --log-opt max-file=3 \
  -v /var/log:/logs:ro \
  alpine tail -n 1000 -F /logs/system.log
```

```yaml [docker-compose.yml]
services:
  system-log:
    container_name: system-log
    image: alpine
    volumes:
      - /var/log:/logs:ro
    command:
      - tail
      - -n
      - "1000"
      - -F
      - /logs/system.log
    labels:
      dev.dozzle.name: system-log
    logging:
      options:
        max-size: 10m
        max-file: "3"
    network_mode: none
    restart: unless-stopped
```

:::

The Compose version is useful if you want the log stream to survive a server reboot. During testing, Alpine used about `~50KB` of memory.

### Why `-F` and Not `-f`

`tail -f` follows the open file handle. When the file is rotated, the handle points at the old, renamed file and the stream goes quiet. `tail -F` follows the path and reopens the file after a rotation, so it keeps working.

For the same reason, mount the **directory** rather than the file. A bind mount of a single file is bound to that file's inode, so a rotation on the host replaces the file and the container keeps looking at the old one, even with `-F`.

### Seeding History

Docker only stores what the container has printed since it started, so restarting the sidecar drops everything Dozzle had. `-n 1000` prints the last 1000 lines on start so the view is not empty.

### Multiple Files

`tail` prefixes each block with the file name when given more than one file. Globs need a shell, since the image has no entrypoint to expand them:

```sh
docker run -d -v /var/log:/logs:ro alpine sh -c 'tail -n 1000 -F /logs/*.log'
```

The `dev.dozzle.name` label above gives the sidecar a readable name in the UI. See [Container Names](/guide/container-names) for more.
