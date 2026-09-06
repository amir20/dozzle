---
title: Dozzle Cloud
sourceHash: 34c0056128a5
---

# Dozzle Cloud

[Dozzle Cloud](https://cloud.dozzle.dev) 是自托管 Dozzle 的可选托管伴侣服务。它把你的各个实例连接起来，汇总容器事件，把警报分发到多个渠道，并让你在聊天中就能询问基础设施的情况。Dozzle 本身仍然是完全开源、自托管的；Cloud 只是在其之上。

我们的目标是让 Dozzle Cloud 成为那个你没想到自己需要的私人 SRE 助手：它盯着你的容器，有事时告诉你，没事时不打扰你。

## 功能

### <Icon icon="mdi:text-box-outline" inline /> 日志摘要

容器事件会被批量汇总，并由大语言模型生成摘要。每条摘要都会记录严重程度、来源容器，以及回到你 Dozzle 实例中完整日志行的链接。

### <Icon icon="mdi:group" inline /> 模式聚类

重复出现的错误会被归并并计数，而不是逐条推送。一个循环抛出同一个异常 200 次，只会产生一条带频次的通知，而不是 200 条。

### <Icon icon="mdi:robot-outline" inline /> AI 助手

一个基于聊天的助手，可以回答关于容器状态和近期日志活动的问题。它可以在 Telegram 和 Discord 中使用。

在 Pro 和 Team 套餐中，助手还可以直接在对话里对容器执行操作（启动、停止、重启），不需要主机的终端访问权限。

### <Icon icon="mdi:calendar-clock" inline /> 每日摘要

对你所有已连接实例近期活动的定时汇总：主要错误模式、事件数量以及整体健康状况。按你配置的时间和时区通过邮件发送。

### <Icon icon="mdi:bell-ring-outline" inline /> 通知渠道

警报可以并行发送到多个渠道。每个渠道都可以独立启用或禁用，也可以限定到特定的 Dozzle 实例。

| 渠道                                                     | 警报 | 每日摘要 | 双向助手 |
| -------------------------------------------------------- | :--: | :------: | :------: |
| <Icon icon="mdi:telegram" inline /> Telegram             |  ✓   |    ✓     |    ✓     |
| <Icon icon="ic:baseline-discord" inline /> Discord       |  ✓   |    ✓     |    ✓     |
| <Icon icon="mdi:email-outline" inline /> 邮件            |  ✓   |    ✓     |          |
| <Icon icon="mdi:slack" inline /> Slack                   |  ✓   |          |          |
| <Icon icon="simple-icons:ntfy" inline /> ntfy            |  ✓   |          |          |
| <Icon icon="mdi:webhook" inline /> Webhook               |  ✓   |          |          |
| <Icon icon="mdi:bell-badge-outline" inline /> 浏览器推送 |  ✓   |          |          |

### <Icon icon="mdi:bell-sleep-outline" inline /> 通知静音

通知可以静音一小时、八小时、到第二天早上，或者到下一周。在故障处理或计划维护期间很有用。

### <Icon icon="mdi:view-dashboard-outline" inline /> 多实例仪表盘

已连接的 Dozzle 实例会出现在同一个仪表盘中。每个实例使用 API 密钥进行身份验证，主机上不需要额外的代理。仪表盘会显示在线状态、容器清单和实时日志流。

### <Icon icon="mdi:database-search-outline" inline /> 全文日志搜索

从已连接实例转发过来的每一行日志都会写入全文搜索索引。你可以一次跨所有实例查询，也可以按容器、严重程度或时间范围过滤。即使跨越数周的历史记录，搜索也能在毫秒级返回结果，每条匹配都会链接回源实例中的上下文。保留时长取决于套餐，从 24 小时到 30 天不等。

### <Icon icon="mdi:shield-lock-outline" inline /> 安全性

- API 密钥使用 BLAKE2b 哈希存储，并支持设置过期时间。
- 登录使用 GitHub 或 Google OAuth。
- 日志和事件内容只在你套餐的保留期内存储。

## 连接一个实例

要把自托管的 Dozzle 连接到 Dozzle Cloud：

1. 打开你的 Dozzle 实例，点击顶栏中的 **cloud** 图标。
2. 点击 **Link instance**。你会被跳转去完成身份验证并确认连接。
3. 连接完成后，在 Dozzle 中配置警报订阅，选择要转发哪些事件。

## 控制转发的内容

默认情况下，实例连接期间每个运行中的容器都会把日志推送到 Dozzle Cloud。对于那些 info 级别的絮叨没有诊断价值的吵闹容器，你可以用一个标签按容器过滤，或者完全退出。

### `dev.dozzle.cloud.min_level`

| 值                                            | 效果                                                         |
| --------------------------------------------- | ------------------------------------------------------------ |
| _（未设置）_                                  | 转发所有日志行。这是默认行为。                               |
| `disabled`                                    | 完全跳过该容器，不向 Cloud 转发任何日志。                    |
| `trace`                                       | 与未设置相同，因为 trace 是最低级别。所有内容都会被转发。    |
| `debug` / `info` / `warn` / `error` / `fatal` | 只转发该级别及以上的日志行。没有识别出级别的行始终会被转发。 |

无法识别的值（比如把 `warn` 写成 `warning` 或 `wran`）会被记为一条错误并忽略，容器会像没有设置标签一样转发全部内容。

这个标签在日志读取器启动时读取。在运行中的容器上修改它，需要容器重启后才会生效。

```yaml
services:
  zigbee2mqtt:
    image: koenkk/zigbee2mqtt
    labels:
      # 只把 warn/error/fatal 转发到 Dozzle Cloud
      - dev.dozzle.cloud.min_level=warn

  noisy-debug-tool:
    image: example/debug
    labels:
      # 不发送这个容器的任何内容
      - dev.dozzle.cloud.min_level=disabled
```

过滤在日志离开主机之前就已经在你的 Dozzle 实例上完成，因此被丢弃的日志行不会经过网络，也不会计入你的套餐用量。Dozzle 中的本地日志查看不受影响。

## 价格

免费套餐是刻意做得慷慨的；在家庭实验室或小团队中，你应该能真正用起来 Dozzle Cloud 而不会撞墙。付费套餐面向更大的事件量、更长的保留期，以及助手的容器操作能力。当前的限额和套餐详情请见 [cloud.dozzle.dev](https://cloud.dozzle.dev)。

## 反馈

Dozzle Cloud 由开发 Dozzle 的同一个人打造，标准也一样：做人们真正愿意用的东西。如果你试用之后觉得哪里别扭、缺了什么，或者发现某个功能特别好用，欢迎[发起一个讨论](https://github.com/amir20/dozzle/discussions)。这些反馈决定了接下来会做什么。
