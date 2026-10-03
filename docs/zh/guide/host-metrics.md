---
title: 主机指标
sourceHash: 5d6314385b49
---

# 主机指标

主机卡片可以显示 Docker 所在机器的三项读数。它们位于卡片标题栏右侧的一个小框里，并带有脉搏图标，这样就不会和下方 CPU、内存仪表中的容器合计数值混淆。在手机上，磁盘会作为第三个仪表显示在 CPU 和内存旁边，运行时间和负载则移到仪表下方的一行：

- **运行时间**，主机已经连续运行了多久
- **负载**，1 分钟平均负载，超过 CPU 核心数时变黄，超过核心数的两倍时变红（鼠标悬停可查看 5 分钟和 15 分钟的数值以及核心数）
- **磁盘**，存放 Docker 数据目录的文件系统用了多少，显示为一个小进度条，超过 70% 变黄，超过 90% 变红（鼠标悬停可查看已用量和总量）

只要有标签页开着，这些读数每 15 秒刷新一次。每一项都只在 Dozzle 能读到真实数值时才会出现，所以在默认安装下，这个小框可能根本不显示，或者只显示其中几项。

## 在容器中运行 Dozzle

在容器内部，`/proc` 描述的是容器本身，而不是主机。Dozzle 不会把容器的数据冒充成主机的数据，所以在你把主机的 `/proc` 挂载到 `/host/proc` 之前，负载和运行时间都不会显示。

::: code-group

```sh
docker run -d \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /proc:/host/proc:ro \
  -v dozzle_data:/data \
  -p 8080:8080 amir20/dozzle:latest
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /proc:/host/proc:ro
      - dozzle_data:/data
    ports:
      - 8080:8080
volumes:
  dozzle_data:
```

:::

磁盘读数不需要额外挂载。Dozzle 测量的是它自己的 `/data` 所在的文件系统，无论 `/data` 是像上面那样的命名卷，还是根本没有挂载，它都位于 Docker 所在的磁盘上。如果你在那里改为绑定挂载一个主机目录（`./data:/data`），读数描述的就是该目录所在的磁盘，通常也是同一块磁盘。

## 直接在主机上运行 Dozzle

直接在主机上运行的 Dozzle 二进制文件会原样读取 `/proc`，所以负载和运行时间无需任何配置。磁盘读数来自 Docker 的数据目录（`docker info | grep "Docker Root Dir"`，通常是 `/var/lib/docker`），所以只要运行 Dozzle 的用户能访问该目录，磁盘读数就能正常工作。

## 代理

每个[代理](/zh/guide/agent)都会读取自己所在的机器，并把数值发送给你正在查看的 Dozzle，所以每个代理的卡片都会显示它自己的运行时间、负载和磁盘。给代理容器加上和 Dozzle 相同的挂载即可：

```yaml [docker-compose.yml]
services:
  dozzle-agent:
    image: amir20/dozzle:latest
    command: agent
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /proc:/host/proc:ro
    ports:
      - 7007:7007
```

更多磁盘的做法也一样，在代理上挂载到 `/host/disks` 下即可。在 [Swarm 模式](/zh/guide/swarm-mode)下，每个节点都以代理身份运行 Dozzle，所以把这些挂载加到服务上，每个节点就会上报自己的数据。

如果代理的版本比它所上报的 Dozzle 旧，它不会发送任何指标，它的卡片也保持原样。更新代理后即可看到这些读数。

## 更多磁盘

磁盘读数默认覆盖 Docker 自己所在的磁盘。如果还想监控其他磁盘，把每一块都挂载到 `/host/disks/<name>` 下即可。目录名会成为该磁盘的标签。

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /proc:/host/proc:ro
      - /mnt/media:/host/disks/media:ro
      - /mnt/backup:/host/disks/backup:ro
```

这时进度条显示的是最满的那块磁盘，因为它会最先用完；鼠标悬停时会列出每块磁盘的已用量和总量。

Dozzle 测量一块磁盘只需要它的挂载点，并不需要其中的文件。但如果挂载的是磁盘根目录，Dozzle 仍然能读到上面的内容。如果你在意这一点，可以在该磁盘上新建一个空目录，改为挂载这个目录（`/mnt/media/.dozzle:/host/disks/media:ro`）。它位于同一个文件系统上，报告的数值完全相同。

直接在主机上安装的 Dozzle 可以用符号链接实现同样的效果：`ln -s /mnt/media /host/disks/media`。

## 限制

- 通过 TCP 连接的[远程主机](/zh/guide/remote-hosts)那一端没有任何东西可以读取机器信息，所以它仍然和以前一样显示 CPU 和内存，但不会有这个小框。改为在那台机器上运行代理即可获得这些读数。
- 如果 `DOCKER_HOST` 指向另一台机器（`tcp://` 或 `ssh://`），Dozzle 会跳过这些读数，因为它自己的 `/proc` 和磁盘反映不了那台引擎的情况。
- Docker Desktop 把引擎运行在虚拟机里。在 macOS 或 Windows 上直接运行的 Dozzle 二进制文件没有 `/proc` 可读，而在那里以容器方式运行的 Dozzle 报告的是虚拟机的数据，不是你这台机器的。
