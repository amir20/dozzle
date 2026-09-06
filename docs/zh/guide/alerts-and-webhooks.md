---
title: 警报与 Webhook
sourceHash: daf372975955
---

# 警报与 Webhook

Dozzle 内置了一套警报系统，可以监控容器日志、资源指标和生命周期事件，并在满足特定条件时发出通知。警报使用可自定义的表达式来筛选容器和触发条件，并可把通知发送到 webhook、Slack、Discord、ntfy 或 [Dozzle Cloud](/zh/guide/dozzle-cloud)。

## <Icon icon="mdi:format-list-bulleted-type" inline /> 警报类型

Dozzle 支持三种警报，都在**通知**页面用同样的方式配置：

| 类型                       | 触发条件                       | 使用场景示例           |
| -------------------------- | ------------------------------ | ---------------------- |
| [**日志**](#log-alerts)    | 日志消息匹配某个模式           | 5xx 错误、堆栈跟踪     |
| [**指标**](#metric-alerts) | CPU / 内存越过阈值             | 容器 CPU 超过 90%      |
| [**事件**](#event-alerts)  | 来自 Docker 的容器生命周期事件 | OOM 杀进程、容器不健康 |

每条警报都由一个**容器表达式**（监控哪些容器）加上一个**触发表达式**（在什么条件下触发）组成。

> [!IMPORTANT]
> 警报和通知目标的配置保存在 `/data` 目录中。你必须把这个目录挂载为卷，通知设置才能在容器重启后保留。

::: code-group

```sh
docker run -v /var/run/docker.sock:/var/run/docker.sock -v /path/to/data:/data -p 8080:8080 amir20/dozzle:latest
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /path/to/data:/data
    ports:
      - 8080:8080
```

:::

## <Icon icon="mdi:send-outline" inline /> 配置通知目标

创建警报之前，至少要配置一个通知目标。在 Dozzle 中打开**通知**页面，点击**添加目标**。

### Webhook

Webhook 会向你指定的 URL 发送一个 HTTP POST 请求。Dozzle 为常用服务内置了载荷模板：

- **Slack**，使用 blocks 和 markdown 格式
- **Discord**，按 Discord webhook API 的格式
- **ntfy**，按 [ntfy.sh](https://ntfy.sh) 推送通知的格式
- **自定义**，通用 JSON 载荷，可自行修改

你也可以用 Go 的 `text/template` 语法编写自己的载荷模板。可用的变量如下：

<div v-pre>

| 变量                      | 说明                       |
| ------------------------- | -------------------------- |
| `{{.Detail}}`             | 摘要（日志消息或指标数值） |
| `{{.Container.Name}}`     | 容器名称                   |
| `{{.Container.Image}}`    | 容器镜像                   |
| `{{.Container.HostName}}` | Docker 主机名称            |
| `{{.Container.State}}`    | 容器状态                   |
| `{{.Log.Message}}`        | 日志消息内容               |
| `{{.Log.Level}}`          | 日志级别                   |
| `{{.Log.Timestamp}}`      | 日志时间戳                 |
| `{{.Log.Stream}}`         | 流类型（stdout/stderr）    |
| `{{.Stat.CPUPercent}}`    | CPU 使用百分比             |
| `{{.Stat.MemoryPercent}}` | 内存使用百分比             |
| `{{.Stat.MemoryUsage}}`   | 内存使用量（字节）         |
| `{{.Subscription.Name}}`  | 警报规则名称               |

</div>

> [!TIP]
> 保存前可以用**测试**按钮确认 webhook 是否正常工作。

### Dozzle Cloud

你也可以把警报发送到 [Dozzle Cloud](/zh/guide/dozzle-cloud)，集中监控多个 Dozzle 实例。详见 [Dozzle Cloud 指南](/zh/guide/dozzle-cloud)。

## <Icon icon="mdi:plus-circle-outline" inline /> 创建警报

打开**通知**页面，点击**添加警报**。每条警报都包含一个**容器表达式**，外加**日志**、**指标**或**事件**三种触发表达式中的一种。

### 容器表达式

容器表达式用来选择要监控的容器。可用属性：

| 属性       | 类型   | 示例                            |
| ---------- | ------ | ------------------------------- |
| `name`     | 字符串 | `name contains "api"`           |
| `image`    | 字符串 | `image == "nginx:latest"`       |
| `state`    | 字符串 | `state == "running"`            |
| `health`   | 字符串 | `health == "unhealthy"`         |
| `hostName` | 字符串 | `hostName == "prod-host"`       |
| `labels`   | 映射   | `labels["env"] == "production"` |

可以用 `&&`（与）、`||`（或）和 `!`（非）组合多个条件：

```
name contains "api" && labels["env"] == "production"
```

## <Icon icon="mdi:text-search" inline /> 日志警报

### 日志表达式

日志表达式用来筛选哪些日志消息会触发警报。可用属性：

| 属性      | 类型        | 示例                       |
| --------- | ----------- | -------------------------- |
| `message` | 字符串/映射 | `message contains "error"` |
| `level`   | 字符串      | `level == "error"`         |
| `stream`  | 字符串      | `stream == "stderr"`       |
| `type`    | 字符串      | `type == "complex"`        |

对于 JSON 日志，可以用点号访问嵌套字段：

```
message.status >= 500 && message.path contains "/api"
```

支持的字符串运算符包括 `contains`、`startsWith`、`endsWith` 和 `matches`（正则）。

### 日志示例

**对生产环境容器的所有错误发出警报：**

```
Container: labels["env"] == "production"
Log:       level == "error"
```

**对 API 容器的 HTTP 5xx 错误发出警报：**

```
Container: name contains "api"
Log:       message.status >= 500
```

**对某个镜像的任何 stderr 输出发出警报：**

```
Container: image startsWith "myapp/"
Log:       stream == "stderr"
```

**对生产环境中响应缓慢的 API 发出警报：**

```
Container: name contains "api" && labels["env"] == "production"
Log:       message.duration > 5000 && message.path contains "/api"
```

**用正则对认证失败发出警报：**

```
Container: name contains "auth" || name contains "gateway"
Log:       message matches "(?i)(unauthorized|forbidden|invalid token)"
```

> [!NOTE]
> 警报编辑器带有自动补全和实时校验。保存前你可以预览匹配到的容器和日志。

## <Icon icon="mdi:chart-line" inline /> 指标警报

当容器的 CPU 或内存使用率越过阈值时，指标警报就会触发。触发表达式针对的是滑动窗口内采样统计的平滑平均值，这样可以避免短暂尖峰造成误报。

### 指标表达式

可用属性：

| 属性          | 类型 | 说明                                |
| ------------- | ---- | ----------------------------------- |
| `cpu`         | 数字 | CPU 使用百分比（0–100），与界面一致 |
| `memory`      | 数字 | 内存使用百分比（0–100）             |
| `memoryUsage` | 数字 | 内存使用量（字节）                  |

### 冷却时间与采样窗口

- **采样窗口**，在计算表达式前对多少秒的统计数据取平均。窗口越长，尖峰越平滑；窗口越短，反应越快。
- **冷却时间**，同一容器两次触发之间的最短间隔秒数。可以避免容器持续超过阈值时警报刷屏。

### 指标示例

**生产环境容器 CPU 过高：**

```
Container: labels["env"] == "production"
Metric:    cpu > 90
```

**某个服务的内存压力：**

```
Container: name contains "api"
Metric:    memory > 85
```

**内存使用量绝对值（1 GiB）：**

```
Container: name == "postgres"
Metric:    memoryUsage > 1073741824
```

## <Icon icon="mdi:bell-outline" inline /> 事件警报

事件警报在 Docker 容器生命周期事件发生时触发，适合在不解析日志的情况下捕获崩溃、OOM 杀进程和健康状态变化。

### 事件表达式

可用属性：

| 属性         | 类型   | 说明                                           |
| ------------ | ------ | ---------------------------------------------- |
| `name`       | 字符串 | 事件名称（见下文）                             |
| `actorId`    | 字符串 | Docker actor ID（通常就是容器 ID）             |
| `attributes` | 映射   | 来自 Docker 的事件属性（随事件类型不同而不同） |
| `timestamp`  | 时间   | 事件发生的时间                                 |

常见的 Docker 事件名称有 `start`、`stop`、`die`、`kill`、`oom`、`restart`、`destroy` 和 `health_status`。

对于 `health_status` 事件，Dozzle 会把当前状态放在 `attributes["healthStatus"]` 中（`healthy` 或 `unhealthy`）。

### 事件示例

**任何生产环境容器退出时发出警报：**

```
Container: labels["env"] == "production"
Event:     name == "die"
```

**对 OOM 杀进程发出警报：**

```
Container: true
Event:     name == "oom"
```

**容器变为不健康时发出警报：**

```
Container: true
Event:     name == "health_status" && attributes["healthStatus"] == "unhealthy"
```

**对意外退出发出警报（忽略正常和优雅关闭）：**

退出码 0（成功）、130（SIGINT）、143（SIGTERM）和 137（SIGKILL）在 `docker stop`、Ctrl+C 和更新时都会出现，因此排除掉以免噪音。真正的错误退出码（1、2、125 等）仍会触发警报。

```
Container: name contains "worker"
Event:     name == "die" && !(attributes["exitCode"] in ["0", "130", "143", "137"])
```

## <Icon icon="mdi:cog-outline" inline /> 管理警报

在通知页面上，你可以：

- **启用/停用**警报，而不必删除它们
- **编辑**警报的表达式和通知目标
- **查看统计信息**，包括触发次数、匹配的容器和最近一次触发时间
- **删除**不再需要的警报
