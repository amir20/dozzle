---
title: 主机名
sourceHash: 8769ba2c0e47
---

# 修改 Dozzle 的主机名

Dozzle 的默认连接名为 localhost。使用 `--hostname` 参数可以把这个名称改成任意值。该值会显示在页面标题以及 Dozzle 徽标下方。

修改后，多主机菜单中 `localhost` 连接的标签也会随之改变。下面的示例用 `--hostname` 把副标题改为 `mywebsite.xyz`。

::: code-group

```sh
docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --hostname mywebsite.xyz
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
      DOZZLE_HOSTNAME: mywebsite.xyz
```

:::

## 多主机与 Agent

`--hostname` 只会重命名运行**当前** Dozzle 进程的主机。远程 [Agent](/zh/guide/agent) 会公布各自的名称，因此需要在每个 Agent 上设置 `DOZZLE_HOSTNAME`（或 `--hostname`）来决定它在多主机菜单中的显示方式。在 [Swarm 模式](/zh/guide/swarm-mode)下每个节点都运行自己的 Agent，请为每个节点设置不同的主机名以便区分。
