---
title: Dozzle Cloud
---

# Dozzle Cloud

<Badge type="tip" text="New in v10" />

[Dozzle Cloud](https://cloud.dozzle.dev) is a companion service that extends your self-hosted Dozzle instances with centralized monitoring, smart alerting, and log intelligence. While Dozzle remains fully open source and self-hosted, Dozzle Cloud adds a managed layer on top for teams that need more visibility across their infrastructure.

## Why Dozzle Cloud?

Container logs are noisy. You could wire up custom webhooks and build your own alerting pipeline — but then you're maintaining alert logic, deduplication, summarization, and delivery infrastructure yourself.

Dozzle Cloud handles all of that out of the box, and adds capabilities that webhooks simply can't provide — like AI-powered summaries, two-way chat with your containers, and proactive daily digests.

## Key Features

### Log Summaries

Raw log lines aren't useful at 2am. Dozzle Cloud automatically batches related container events into concise, AI-powered summaries. Each summary includes severity levels, source container information, and direct links to the full logs in your Dozzle instance.

### Pattern Clustering

Instead of showing the same error 50 times, Dozzle Cloud groups similar errors together and displays frequency counts. This makes it easy to identify recurring issues across multiple containers without the noise.

### AI Agent

Talk to your containers. The AI agent lets you ask questions about container health, search historical logs, and get context on what went wrong — all from a chat interface in Telegram or Discord.

On Pro and Team plans, the agent can also take action: restart, start, or stop containers directly from the conversation. Get an alert, ask what happened, and fix it — without leaving the chat or SSH-ing into a server.

### Daily Digests

Webhooks are reactive — they only fire when something breaks. Daily digests give you a proactive summary of what's happening across your infrastructure: top error patterns, event counts, and overall health. Delivered to your inbox at a time you choose, in your timezone.

### Smart Alert Distribution

Receive notifications through multiple channels — and unlike one-way webhooks, some channels support full two-way interaction:

- **Telegram** — alerts plus two-way AI agent chat
- **Discord** — alerts plus two-way AI agent chat
- **Email** — alerts and daily digests
- **Slack**
- **ntfy**
- **Webhooks**
- **Browser push notifications**

You can enable or disable channels independently, scope them to specific Dozzle instances, and set up as many as you need.

### Notification Muting

Sometimes you know things are broken and you're working on it. Mute notifications for an hour, eight hours, until tomorrow, or until next week — so you can focus on fixing without the noise.

### Multi-Instance Dashboard

Monitor all your Dozzle servers from a single dashboard. Each instance connects with an API key — no additional agents required. See which instances are online, browse containers, and stream logs in real time.

### Searchable Event History

Search across all logged events with full-text search. Filter by container, severity, or keyword. Retention is configurable from 24 hours to 30 days depending on your plan.

### Security

- API keys are hashed with BLAKE2b and support expiration
- GitHub and Google OAuth authentication
- Your logs remain your own — Dozzle Cloud is committed to data privacy

## Connecting to Dozzle Cloud

To link your Dozzle instance to Dozzle Cloud:

1. Open your local Dozzle and click the **cloud** icon in the top bar
2. Click **Link instance** — you'll be redirected to authenticate and confirm the connection
3. Once linked, configure alert subscriptions in Dozzle to start receiving notifications

## Pricing

Dozzle Cloud offers a free tier with 500 events per month — no credit card required. Visit [cloud.dozzle.dev](https://cloud.dozzle.dev) for plan details and pricing.
