---
title: Dozzle Cloud
---

# Dozzle Cloud

<Badge type="tip" text="New in v10" />

[Dozzle Cloud](https://cloud.dozzle.dev) is an optional managed companion to self-hosted Dozzle. It links your instances together, summarizes container events, distributes alerts across multiple channels, and lets you ask questions about your infrastructure from chat. Dozzle itself remains fully open source and self-hosted; Cloud sits on top.

The goal is for Dozzle Cloud to feel like the personal SRE assistant you never knew you wanted: it watches your containers, tells you when something matters, and stays out of the way when nothing does.

## Features

### <Icon icon="mdi:text-box-outline" inline /> Log Summaries

Container events are batched and summarized using an LLM. Each summary records severity, the source container, and a link back to the full log line in your Dozzle instance.

### <Icon icon="mdi:group" inline /> Pattern Clustering

Repeated errors are grouped and counted instead of being delivered individually. A loop emitting the same exception 200 times produces one notification with a frequency, not 200.

### <Icon icon="mdi:robot-outline" inline /> AI Agent

A chat-based agent answers questions about container state and recent log activity. It is available in Telegram and Discord.

On Pro and Team plans, the agent can also act on containers (start, stop, restart) directly from the conversation, without requiring shell access to the host.

### <Icon icon="mdi:calendar-clock" inline /> Daily Digests

A scheduled summary of recent activity across your linked instances: top error patterns, event counts, and overall health. Delivered by email at a time and timezone you configure.

### <Icon icon="mdi:bell-ring-outline" inline /> Notification Channels

Alerts can be routed to multiple channels in parallel. Each channel can be enabled or disabled independently and scoped to specific Dozzle instances.

| Channel                                                    | Alerts | Daily Digest | Two-way agent |
| ---------------------------------------------------------- | :----: | :----------: | :-----------: |
| <Icon icon="mdi:telegram" inline /> Telegram               |   ✓    |      ✓       |       ✓       |
| <Icon icon="ic:baseline-discord" inline /> Discord         |   ✓    |      ✓       |       ✓       |
| <Icon icon="mdi:email-outline" inline /> Email             |   ✓    |      ✓       |               |
| <Icon icon="mdi:slack" inline /> Slack                     |   ✓    |              |               |
| <Icon icon="simple-icons:ntfy" inline /> ntfy              |   ✓    |              |               |
| <Icon icon="mdi:webhook" inline /> Webhooks                |   ✓    |              |               |
| <Icon icon="mdi:bell-badge-outline" inline /> Browser push |   ✓    |              |               |

### <Icon icon="mdi:bell-sleep-outline" inline /> Notification Muting

Notifications can be muted for one hour, eight hours, until the next morning, or until the following week. Useful during incidents or planned maintenance.

### <Icon icon="mdi:view-dashboard-outline" inline /> Multi-Instance Dashboard

Linked Dozzle instances appear in a single dashboard. Each instance authenticates with an API key, with no additional agent required on the host. The dashboard shows online status, container inventory, and live log streaming.

### <Icon icon="mdi:database-search-outline" inline /> Full-Text Log Search

Every log line forwarded from your linked instances is written into a full-text search index. You can query across all instances at once, or filter by container, severity, or time range. Searches return results in milliseconds even over weeks of history, and each match links back to the surrounding context in the source instance. Retention is plan-dependent and ranges from 24 hours to 30 days.

### <Icon icon="mdi:shield-lock-outline" inline /> Security

- API keys are hashed with BLAKE2b and support expiration.
- Sign-in uses GitHub or Google OAuth.
- Logs and event content are stored only for as long as your plan's retention window.

## Connecting an Instance

To link a self-hosted Dozzle to Dozzle Cloud:

1. Open your Dozzle instance and click the **cloud** icon in the top bar.
2. Click **Link instance**. You will be redirected to authenticate and confirm the connection.
3. Once linked, configure alert subscriptions inside Dozzle to choose which events are forwarded.

## Pricing

The free tier is intentionally generous; you should be able to actually use Dozzle Cloud on a homelab or a small team without hitting a wall. Paid plans exist for higher event volumes, longer retention, and the agent's container actions. See [cloud.dozzle.dev](https://cloud.dozzle.dev) for current limits and plan details.

## Feedback

Dozzle Cloud is built by the same person who built Dozzle, and the bar is the same: things people actually want to use. If you try it and something feels off, missing, or genuinely useful, please [open a discussion](https://github.com/amir20/dozzle/discussions). That feedback shapes what gets built next.
