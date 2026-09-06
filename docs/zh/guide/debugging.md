---
title: 调试
sourceHash: 2cb9b9a633e2
---

# 通过日志调试

Dozzle 默认以 `info` 级别记录日志，输出刻意保持得很少。遇到问题时，可以用 `--level` 参数或 `DOZZLE_LEVEL` 环境变量提高日志详细程度。

| 级别    | 适用场景                                                  |
| ------- | --------------------------------------------------------- |
| `info`  | 默认级别。启动信息、错误和警告。                          |
| `debug` | 请求级诊断信息、认证判定、agent 连接、配置输出。          |
| `trace` | 全部内容。逐条日志事件、信标负载、gRPC 帧。输出量非常大。 |

Dozzle 把所有日志写入 `stdout`，所以用 `docker logs dozzle` 就能查看。

```yaml
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    ports:
      - 8080:8080
    environment:
      DOZZLE_LEVEL: debug
```

## 报告 Bug

如果你认为遇到了 bug，请到 [github.com/amir20/dozzle/issues](https://github.com/amir20/dozzle/issues) 提交 issue，并附上：

- Dozzle 版本（界面页脚可见，或执行 `dozzle --version`）
- 部署模式：server、swarm、k8s 或 agent
- Docker 或 Kubernetes 版本
- 相关的 `debug` 或 `trace` 级别日志输出
- 复现步骤，最好附上一个最小化的 `docker-compose.yml`

初次报告中的信息越完整，问题分类处理得就越快。
