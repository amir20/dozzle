---
title: Dozzle Cloud
---

# Dozzle Cloud

[Dozzle Cloud](https://cloud.dozzle.dev) is an optional managed companion to self-hosted Dozzle. Dozzle itself stays fully open source and self-hosted; Cloud sits on top of it and takes over the part that is genuinely hard to run yourself — deciding what is worth waking you up for, and figuring out what actually broke.

Your Dozzle makes an outbound connection to Cloud. There is no inbound port, no public IP, and no agent to install.

**Free keeps you quiet. Pro goes looking.**

## <Icon icon="mdi:bell-ring-outline" inline /> Free: an intelligent notification layer

Most log alerting is a regex and a webhook, which means the first crash loop turns into two hundred identical messages and you mute the channel. The free tier exists to fix that part, and it is the whole alerting product rather than a trial of one.

- **Smart alerts** — every alert your Dozzle rules fire is triaged into a sentence that names the cause, the container, and how bad it is, with a link back to the exact log line in your own Dozzle.
- **Repeats fold** — 47 crashes arrive as one alert that says 47. You get a recovery notice when it comes back.
- **Quiet by default** — suppression, severity filters, and pattern-based muting on every channel. Mute _this kind of alert_ rather than this one alert, and anything genuinely different still gets through.
- **Every channel** — email, Telegram, Discord, Slack, ntfy, webhooks, and browser push, all on free. See [Notification Channels](/guide/dozzle-cloud/channels).
- **Search and stats included** — every event is queryable the moment it lands, and CPU, memory, network, and disk are recorded as history. Neither counts against your event allowance.
- **One finding a week** — even on free, Cloud reads your logs and surfaces the most serious thing that nothing alerted on.
- **A working default rule** — linking an instance creates one for you (containers that exit with an error), so a new account gets a useful alert on day one without configuring anything.
- **Chat agent and MCP** — ask "any errors today?" in Telegram or Discord, and start, stop, or restart a container from the same conversation once you enable [Actions](/guide/actions) on your instance. MCP access is unlimited on every plan.

> [!TIP]
> A newly connected instance gets 7 days of the full Pro experience: every finding, every morning. Free settles into one finding a week after that.

## <Icon icon="mdi:robot-outline" inline /> Pro: it goes looking before anything alerts

Free tells you _that_ something happened, and keeps quiet when nothing did. Pro is the half that does not wait for an alert to exist.

- **Proactive triage, every morning** — Cloud reads your error logs, collapses them into patterns, and reports what is worth fixing. This is where a disk creeping toward full, or a container quietly restart-looping, shows up on a day when nothing fired at all. No alert rule has to exist for it.
- **Every finding, daily, with the fix** — not one a week with the rest locked. Findings age day to day while the problem lasts ("still happening, day four, three times worse") and close themselves when it stops.
- **Triage that goes and looks** — when the alert text alone is not enough to decide, it inspects the container and reads the surrounding logs before making a call, instead of guessing.
- **Full investigations on demand** — one click runs more passes with a stronger model, correlating across your containers, hosts, and timeline, and hands back a root cause with concrete steps.
- **Every host, one dashboard** — connect as many Dozzle instances as you run. Questions asked in chat cover all of them at once.
- **Longer memory** — 30 days of searchable logs and stats instead of 24 hours, which is the difference between "what happened last night" and "has this been happening all month".

See [Plans & Limits](/guide/dozzle-cloud/plans) for the full comparison.

## Where to go next

| Page                                                       | What it covers                                                                         |
| ---------------------------------------------------------- | -------------------------------------------------------------------------------------- |
| [Connecting Your Instance](/guide/dozzle-cloud/connecting) | Linking, why no public IP or open port is needed, firewall rules, troubleshooting      |
| [In Your Dozzle](/guide/dozzle-cloud/in-dozzle)            | The cloud rail, alerts that survive a reload, and what a Cloud-less install still does |
| [Notification Channels](/guide/dozzle-cloud/channels)      | Every channel, how to set each one up, and how to make alerts quieter                  |
| [Plans & Limits](/guide/dozzle-cloud/plans)                | What each plan includes, what a triaged event is, what happens when you go over        |
| [Your Data](/guide/dozzle-cloud/your-data)                 | What leaves your host, how to stop it, what Cloud stores, API keys                     |

Alert rules themselves are configured on your own instance, not in Cloud. See [Alerts](/guide/alerts-and-webhooks).

## Feedback

Dozzle Cloud is built by the same person who built Dozzle, and the bar is the same: things people actually want to use. If you try it and something feels off, missing, or genuinely useful, please [open a discussion](https://github.com/amir20/dozzle/discussions). That feedback shapes what gets built next.
