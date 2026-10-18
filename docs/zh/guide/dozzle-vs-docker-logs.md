---
title: Dozzle 与 docker logs 对比
description: 相比 docker logs 和终端工具，Dozzle 多了什么，以及什么时候终端就够用了。
sourceHash: 6b766c4db7ff
---

# Dozzle 与 `docker logs` 对比

Dozzle 读取的日志和 `docker logs` 相同。它不复制也不存储任何内容，所以它显示的东西，这条命令都能显示。它改变的是你阅读日志的方式：在浏览器中，跨容器、跨主机，而且不需要登录服务器的 shell。

## 一览

|            | Dozzle                                                         | `docker logs`                                     |
| ---------- | -------------------------------------------------------------- | ------------------------------------------------- |
| 在哪里查看 | 浏览器，任何设备都可以                                         | Docker 主机上的 shell                             |
| 多个容器   | 任意一组容器放在同一个合并视图中，或者整个 stack 或服务        | 一个容器。`docker compose logs` 可以覆盖一个项目  |
| 多台主机   | 通过 [agent](/zh/guide/agent)，用一个界面查看所有主机          | 每个 shell 一台主机，或每条命令一个 `DOCKER_HOST` |
| 搜索       | 边输入边搜索，支持正则表达式                                   | 通过管道交给 `grep`                               |
| JSON 日志  | 格式化并着色，可对字段使用 [SQL](/zh/guide/sql-engine)         | 原始行，通过管道交给 `jq`                         |
| 堆栈跟踪   | 多行条目归为一组                                               | 一次一行                                          |
| CPU 和内存 | 日志旁边的图表                                                 | 在另一个 shell 中运行 `docker stats`              |
| 分享       | 一个指向容器的链接，受[身份验证](/zh/guide/authentication)保护 | 复制粘贴                                          |
| 安装       | 一个容器                                                       | 已经自带                                          |

## 什么时候终端就够了

如果你已经在主机上，只想看某个容器的最后一百行，`docker logs --tail 100 -f <name>` 是最快的办法。如果想在终端里实时查看所有容器，[dtop](/zh/guide/dtop) 会用类似 `docker ps` 的表格显示它们，需要完整日志时再打开 Dozzle。

## 什么时候 Dozzle 更合适

当不止一个人查看日志，当你不想为了看日志而分发 SSH 访问权限，或者当你要同时在多个容器之间跟踪一个请求时，就用 Dozzle。在手机上查看日志，它也更方便。

## 试一试

```sh
docker run -d -v /var/run/docker.sock:/var/run/docker.sock -v dozzle_data:/data -p 8080:8080 amir20/dozzle:latest
```

然后打开 `http://localhost:8080`。更多选项见[快速开始](/zh/guide/getting-started)。
