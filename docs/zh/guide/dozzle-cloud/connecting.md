---
title: 连接你的实例
sourceHash: d6f8f7b6b845
---

# 连接你的实例

如何把自托管的 Dozzle 连到 [Dozzle Cloud](/zh/guide/dozzle-cloud)、确认它真的连上了，以及没连上时怎么修。

## 连接一个实例

1. 打开自托管的 Dozzle，点击顶栏的**云**图标。
2. 点击 **Link instance**。你会被带到 Cloud，用 GitHub 或 Google 登录并确认。
3. 几秒后实例就会出现在 Cloud 面板上。

不用创建密码，主机上也不用装任何 agent。

## 不需要公网 IP、开放端口或域名

这是最常见的顾虑，三个问题的答案都是不需要。

你的 Dozzle 实例向 Cloud 发起一条**出站**连接并保持长连。Cloud 从不反向连接你、从不扫描你的主机，也永远不需要访问你的地址。所以在下面这些情况下都能正常工作：

- 在家庭网络的 NAT 后面，没有端口转发
- 使用 `192.168.1.50` 这样的 RFC1918 私有地址
- 在 Tailscale、WireGuard 或 ZeroTier 网络里
- 在 CGNAT 后面，就算想做端口转发也做不了
- 在一台不断切换网络的笔记本上

不需要反向代理、动态 DNS 域名或固定 IP。

## 防火墙规则

只需要**出站**访问。允许你的 Dozzle 主机访问：

```
agent.doligence.dozzle.dev:443    (TCP, outbound)
```

443 端口上的这一个目标就够了。如果你的防火墙按主机名而不是 IP 过滤，请放行这个主机名，它背后的地址可能会变。大多数家用和小型办公防火墙本来就放行全部出站流量，所以通常根本不用配置。

## 规则和渠道分别在哪里

几乎所有人都会在这里绊一下，所以直说：

| 你想改的东西                                 | 在哪里改                                               |
| -------------------------------------------- | ------------------------------------------------------ |
| **什么会触发告警** —— 容器、模式、阈值       | 自托管 Dozzle → [警报](/zh/guide/alerts-and-webhooks)  |
| **告警发到哪里** —— 邮件、Telegram、Slack 等 | Dozzle Cloud → [渠道](/zh/guide/dozzle-cloud/channels) |
| 查看历史告警、静音、升级套餐                 | Dozzle Cloud                                           |

规则定义在你自己的实例上，因为日志在那里。投递配置在 Cloud，因为握着通往你手机那条连接的是它。如果你在 Cloud 里到处找“这个容器报错时告诉我”的设置却找不到，原因就在这里：请打开你自托管的 Dozzle。

## 连接多个实例

每个实例分别连接，步骤相同，用同一个 Cloud 账号。连上之后它们会一起出现在面板里，聊天里问的问题一次覆盖所有已连接的实例。

这个汇总视图存在于 Cloud，而不在某一个自托管的 Dozzle 里。自托管的 Dozzle 只显示你直接在它上面配置的主机，不会显示其他已连接的实例。

连接多个 Dozzle 实例和 Dozzle 自己的 [agent 模式](/zh/guide/agent)、[远程主机](/zh/guide/remote-hosts)是两码事，后者是把额外的 Docker 主机接到同一个 Dozzle 上。两者都支持，也可以组合使用。

免费套餐同时只能连接一个实例。见[套餐与限制](/zh/guide/dozzle-cloud/plans)。

## 什么都没出现

按顺序排查。

**1. Dozzle 容器在运行吗？**
如果 Dozzle 本身停了或正在重启，什么都到不了 Cloud。

**2. 连接流程完成了吗？**
开始连接但没有确认，是不会留下任何东西的。重做上面的步骤，然后确认实例出现在 Instances 页面上。

**3. API 密钥被删了吗？**
删除实例的 API 密钥会永久解除连接。旧密钥无法重新挂回去，重新连接一次获取新的即可。

**4. 出站流量被拦了吗？**
限制严格的网络（公司、学校、某些 VPS 服务商）可能会拦截到非白名单目标的出站 443。见上面的*防火墙规则*。

**5. 是不是到了免费版的实例上限？**
免费套餐同时只能连一个实例。尝试连接第二个时会显示限制提示，而不会连上。

**6. 这个容器是不是被排除在转发之外？**
带有 `dev.dozzle.cloud.min_level=disabled` 标签的容器按设计什么都不发。如果只有某个容器缺失而其他都正常，检查它的标签。见[你的数据](/zh/guide/dozzle-cloud/your-data)。

## 让 agent 控制容器

实例一连上，读取日志和容器状态就能用。启动、停止和重启则会被你的实例拒绝，除非你自己打开：

::: code-group

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle
    environment:
      DOZZLE_ENABLE_ACTIONS: true
```

```sh
docker run ... amir20/dozzle --enable-actions
```

:::

这是**你的** Dozzle 上的设置，不在 Cloud 里，因为它决定的是你的 Dozzle 愿意对你的容器做什么。改完后重启 Dozzle。见[操作](/zh/guide/actions)。

## 断开连接

在 Cloud 的 Instances 页面删除该实例的 API 密钥。连接随即断开，不再转发任何数据，你自托管的 Dozzle 和之前完全一样继续工作。连接从来不会改变本地的日志查看。
