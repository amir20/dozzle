---
title: Dozzle Cloud
---

# Dozzle Cloud

[Dozzle Cloud](https://cloud.dozzle.dev) is an optional managed companion to self-hosted Dozzle. Dozzle itself stays fully open source and self-hosted; Cloud sits on top of it and takes over the part that is genuinely hard to run yourself — deciding what is worth waking you up for, and figuring out what actually broke.

Your Dozzle makes an outbound connection to Cloud. There is no inbound port, no public IP, and no agent to install.

The free plan is the whole alerting layer: every alert your Dozzle rules fire is triaged into one readable message, repeats fold into a single alert with a count, and it goes out to email, Telegram, Discord, Slack, ntfy, webhooks, or browser push. Paid plans add the proactive half, where Cloud reads your logs every morning and reports problems that never fired an alert, along with longer history and more instances.

See [Features](https://cloud.dozzle.dev/features) for the full tour and [Pricing](https://cloud.dozzle.dev/pricing) for what each plan includes.

## Where to go next

| Page                                                       | What it covers                                                                         |
| ---------------------------------------------------------- | -------------------------------------------------------------------------------------- |
| [Connecting Your Instance](/guide/dozzle-cloud/connecting) | Linking, why no public IP or open port is needed, firewall rules, troubleshooting      |
| [In Your Dozzle](/guide/dozzle-cloud/in-dozzle)            | The cloud rail, alerts that survive a reload, and what a Cloud-less install still does |
| [Notification Channels](/guide/dozzle-cloud/channels)      | Every channel, how to set each one up, and how to make alerts quieter                  |
| [Plans & Limits](/guide/dozzle-cloud/plans)                | Running into a limit, the instance limit, usage, cancelling                            |
| [Your Data](/guide/dozzle-cloud/your-data)                 | What leaves your host, how to stop it, what Cloud stores, API keys                     |

Alert rules themselves are configured on your own instance, not in Cloud. See [Alerts](/guide/alerts-and-webhooks).

## Feedback

Dozzle Cloud is built by the same person who built Dozzle, and the bar is the same: things people actually want to use. If you try it and something feels off, missing, or genuinely useful, please [open a discussion](https://github.com/amir20/dozzle/discussions). That feedback shapes what gets built next.
