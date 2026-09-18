---
title: Dozzle Cloud
sourceHash: ddf1502ea23d
---

# Dozzle Cloud

[Dozzle Cloud](https://cloud.dozzle.dev) 是自托管 Dozzle 的可选托管伴侣。Dozzle 本身依然完全开源、完全自托管；Cloud 建在它之上，接手那些真正难以自己搭建的部分：判断什么值得把你叫醒，以及弄清楚到底是什么坏了。

你的 Dozzle 会向 Cloud 发起一条出站连接。没有入站端口，不需要公网 IP，也不用安装任何 agent。

免费套餐就是完整的告警层：你的 Dozzle 规则触发的每条告警都会被分诊成一条易读的消息，重复会折叠成一条带计数的告警，然后通过邮件、Telegram、Discord、Slack、ntfy、webhook 或浏览器推送发出。付费套餐加上主动的那一半：Cloud 每天早上读你的日志，报告那些从未触发告警的问题，另外还有更长的历史和更多实例。

完整介绍见[功能](https://cloud.dozzle.dev/features)，每个套餐包含什么见[价格](https://cloud.dozzle.dev/pricing)。

## 接下来看什么

| 页面                                                 | 内容                                                            |
| ---------------------------------------------------- | --------------------------------------------------------------- |
| [连接你的实例](/zh/guide/dozzle-cloud/connecting)    | 连接步骤、为什么不需要公网 IP 或开放端口、防火墙、排障          |
| [在你的 Dozzle 里](/zh/guide/dozzle-cloud/in-dozzle) | Cloud 侧栏、刷新后仍在的告警，以及未连接 Cloud 时依然可用的部分 |
| [通知渠道](/zh/guide/dozzle-cloud/channels)          | 所有渠道、逐个的配置方法，以及怎么让它安静下来                  |
| [套餐与限制](/zh/guide/dozzle-cloud/plans)           | 碰到限制、实例数量限制、用量、取消                              |
| [你的数据](/zh/guide/dozzle-cloud/your-data)         | 什么会离开你的主机、如何阻止、Cloud 存了什么、API 密钥          |

告警规则本身是在你自己的实例上配置的，不在 Cloud。见[警报](/zh/guide/alerts-and-webhooks)。

## 反馈

Dozzle Cloud 和 Dozzle 出自同一个人之手，标准也一样：做人们真正愿意用的东西。如果你用下来觉得哪里别扭、缺了什么，或者哪里特别好用，欢迎[发起一个 discussion](https://github.com/amir20/dozzle/discussions)。这些反馈决定了接下来做什么。
