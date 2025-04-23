---
title: Following Log Files on Disk
---

# Following Log Files on Disk

Some containers do not write logs to `sysout` or `syserr`. Many folks have asked if Dozzle can also show logs that are written to files. Unfortunately, files in containers are not accessible to other containers, so Dozzle wouldn't have a way to access these files. Dozzle can only access logs written to `sysout` or `syserr`, which is the same functionality as the `docker logs` command.

If you are creating a service using Docker, then make sure to write logs to streams. An application should not attempt to write to logfiles. Instead, delegate the logging to Docker. The [twelve factor app](https://12factor.net/logs) has a great principle around logging that explains the importance of this principle.

However, there are workarounds to be able to still access files using mounts.

## Mounting Local Log Files with Alpine

Dozzle reads any output stream. This can be used in combination with Alpine to `tail` a mounted file. An example of this is as follows:

::: code-group

```sh
docker run -v /var/log/system.log:/var/log/test.log alpine tail -f /var/log/test.log
```

```yaml [docker-compose.yml]
services:
  dozzle-from-file:
    container_name: dozzle-from-file
    image: alpine
    volumes:
      - /var/log/system.log:/var/log/stream.log
    command:
      - tail
      - -f
      - /var/log/stream.log
    network_mode: none
    restart: unless-stopped
```

:::

In the above example, `/var/log/system.log` is mounted from the host and used with `tail -f` to follow the file. `tail` is smart to follow log rotations. During testing, using Alpine used about `~50KB` of memory.

The second tab shows a `docker-compose` file which is useful if you want the log stream to survive server reboot.
