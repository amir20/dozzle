---
title: 跟踪磁盘上的日志文件
sourceHash: e6e23f2438f9
---

# 跟踪磁盘上的日志文件

有些容器把日志写入文件，而不是 `stdout` 或 `stderr`。Dozzle 只能读取 Docker 自身捕获的内容，也就是 `stdout` 和 `stderr`，和 `docker logs` 一样。容器内部的文件对其他容器不可见，所以 Dozzle 没有办法读到它们。

## 改为输出到流

最好的办法是不要再写入文件。大多数应用都有把日志输出到控制台的配置项，[十二要素应用](https://12factor.net/logs)也解释了为什么这才是正确的默认做法。

如果应用无法配置，可以在 `Dockerfile` 中把日志文件软链接到容器的 stdout。官方的 nginx 镜像就是这么做的：

```dockerfile
RUN ln -sf /dev/stdout /var/log/nginx/access.log \
    && ln -sf /dev/stderr /var/log/nginx/error.log
```

## 用 sidecar 跟踪文件

如果两种办法都行不通，可以运行一个小的 Alpine 容器来 tail 这个文件，让 Docker 捕获它的输出。这样 Dozzle 就会像显示其他容器一样显示它。

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

如果你希望日志流在服务器重启后继续工作，Compose 版本会更合适。测试中 Alpine 大约占用 `~50KB` 内存。

### 为什么用 `-F` 而不是 `-f`

`tail -f` 跟踪的是已打开的文件句柄。文件轮转后，句柄仍指向被改名的旧文件，日志流就没有内容了。`tail -F` 跟踪的是路径，轮转之后会重新打开文件，因此可以继续工作。

出于同样的原因，请挂载**目录**而不是文件。对单个文件做绑定挂载会绑定到该文件的 inode，主机上的一次轮转会替换掉这个文件，容器则仍然盯着旧文件，即使加了 `-F` 也一样。

### 预填历史记录

Docker 只保存容器启动之后打印的内容，因此重启 sidecar 会让 Dozzle 中已有的内容全部消失。`-n 1000` 会在启动时打印最后 1000 行，这样视图不会是空的。

### 多个文件

当给定多个文件时，`tail` 会在每段内容前加上文件名。使用通配符需要 shell，因为镜像没有可以展开它们的 entrypoint：

```sh
docker run -d -v /var/log:/logs:ro alpine sh -c 'tail -n 1000 -F /logs/*.log'
```

上面的 `dev.dozzle.name` 标签会给这个 sidecar 在界面中一个易读的名称。更多内容参见[容器名称](/zh/guide/container-names)。
