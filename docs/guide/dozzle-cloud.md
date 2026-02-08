---
title: Dozzle Cloud
---

# Dozzle Cloud

<Badge type="tip" text="New in v10" />

[Dozzle Cloud](https://cloud.dozzle.dev) is a companion service that extends your self-hosted Dozzle instances with centralized monitoring, smart alerting, and log intelligence. While Dozzle remains fully open source and self-hosted, Dozzle Cloud adds a managed layer on top for teams that need more visibility across their infrastructure.

## Why Dozzle Cloud?

Container logs are noisy. Dozzle Cloud helps you cut through the noise with intelligent summarization and multi-instance aggregation — without requiring additional agents or complex setup.

## Key Features

### Log Summaries

Dozzle Cloud automatically batches related container events into concise summaries. Each summary includes severity levels, source container information, and direct links to the full logs in your Dozzle instance.

### Pattern Clustering

Instead of showing duplicate errors, Dozzle Cloud groups similar errors together and displays frequency counts. This makes it easy to identify recurring issues across multiple containers.

### Smart Alert Distribution

Receive notifications through multiple channels:

- **Email**
- **Slack**
- **ntfy**
- **Webhooks**
- **Browser push notifications**

You can enable or disable channels independently with unlimited configurations.

### Multi-Instance Dashboard

Monitor all your Dozzle servers from a single dashboard. Connecting is simple — just link your instance with an API key. No additional agents are required.

### Searchable Event History

Search across all logged events with full-text search. Filter by container, severity (error, warning, info), and keywords. Retention is configurable from 24 hours to 30 days.

### Security

- API keys are hashed with expiration support
- GitHub OAuth authentication
- Your logs remain your own — Dozzle Cloud is committed to data privacy

## Connecting to Dozzle Cloud

To link your Dozzle instance to Dozzle Cloud:

1. Navigate to the **Notifications** page in Dozzle
2. Click **Add Destination** and select **Dozzle Cloud**
3. Click **Link Dozzle Cloud** — you'll be redirected to authenticate
4. Once linked, your API key is automatically configured

You can then select Dozzle Cloud as a destination when creating [alerts](/guide/alerts-and-webhooks).

## Pricing

Dozzle Cloud offers a free tier with 500 events per month — no credit card required. Visit [cloud.dozzle.dev](https://cloud.dozzle.dev) for more details.
