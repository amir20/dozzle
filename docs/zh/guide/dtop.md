---
title: 认识 dtop
sourceHash: 3137db243510
---

# 什么是 dtop？

`dtop` 是 Dozzle 的命令行搭档，可以在终端里实时查看系统上运行的 Docker 容器。可以把它看作功能更丰富的 `docker ps`，适合一直开在某个 tmux 窗格里。当你需要完整的历史日志、搜索或图表时，`dtop` 能让你直接跳转到 Dozzle。

它通过 `ssh`、`tcp` 或本地 `unix socket` 连接 Docker 主机，因此同样适用于 Dozzle 支持的多主机场景。

![dtop screenshot](https://github.com/amir20/dtop/raw/master/demo.gif)

## 安装

使用 Homebrew 安装：

```bash
brew install dtop
```

也可以不安装任何东西，直接通过 Docker 运行：

```bash
docker run -v /var/run/docker.sock:/var/run/docker.sock -it ghcr.io/amir20/dtop:latest
```

完整的安装说明见 [https://github.com/amir20/dtop](https://github.com/amir20/dtop?tab=readme-ov-file#installation)。

## 项目状态

`dtop` 是一个新项目，功能还不如 Dozzle 丰富。不过我正在积极添加更多功能。我自己就用它在命令行上监控跨多台主机的所有容器。如果你有建议，欢迎到 [https://github.com/amir20/dtop/issues](https://github.com/amir20/dtop/issues) 提交 issue。
