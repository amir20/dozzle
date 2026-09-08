---
title: Notification Channels
---

# Notification Channels

Channels are configured in [Dozzle Cloud](/guide/dozzle-cloud) and control _where_ alerts go. What _triggers_ an alert is configured on your self-hosted instance — see [Alerts](/guide/alerts-and-webhooks).

Enable as many as you like. Every enabled channel receives every alert, and each can be turned on or off independently.

## Available channels

| Channel                                                     | Alerts | Daily digest | Two-way agent |
| ----------------------------------------------------------- | :----: | :----------: | :-----------: |
| <Icon icon="mdi:email-outline" inline /> Email              |   ✓    |      ✓       |               |
| <Icon icon="mdi:telegram" inline /> Telegram                |   ✓    |      ✓       |       ✓       |
| <Icon icon="ic:baseline-discord" inline /> Discord bot (DM) |   ✓    |      ✓       |       ✓       |
| <Icon icon="ic:baseline-discord" inline /> Discord webhook  |   ✓    |      ✓       |               |
| <Icon icon="mdi:slack" inline /> Slack                      |   ✓    |              |               |
| <Icon icon="simple-icons:ntfy" inline /> ntfy               |   ✓    |              |               |
| <Icon icon="mdi:webhook" inline /> Webhooks                 |   ✓    |              |               |
| <Icon icon="mdi:bell-badge-outline" inline /> Browser push  |   ✓    |              |               |

All channels are available on every plan, including free.

## Email

Set up automatically with the address you signed up with. Nothing to configure. To stop it, disable the email channel. If alerts stop arriving unexpectedly, check spam first — the first alert occasionally lands there, and marking it "not spam" fixes it permanently.

## Telegram

Choose **Telegram** on the Channels page, follow the link to open the bot, and press **Start**. The channel activates once the bot has heard from you.

Telegram is two-way. You can reply in the same chat and ask about your containers — "any errors today?", "show CPU usage", "what alerts do I have?" — and get answers about live state.

## Discord

Discord has **two separate channel types**, and having both running at once is the usual reason for receiving every alert twice.

**Discord bot (direct message)** — the bot sends alerts to you personally as a DM. Two-way, so you can ask it questions. Set up by authorizing the bot from the Channels page.

**Discord webhook (server channel)** — alerts post into a channel on your server, such as `#alerts`. One-way. Set up by creating a webhook in your Discord server settings and pasting the URL into Cloud.

If alerts arrive in both your DMs and a server channel, you have both configured. Disable whichever you do not want; turning one off leaves the other running. A common setup is to keep the shared server channel and switch off the DM.

## Slack

Create an incoming webhook in your Slack workspace and paste the URL into the Slack channel on the Channels page.

## ntfy

Enter your topic URL. Both ntfy.sh and a self-hosted ntfy server work. Popular for phone notifications without an extra account.

## Webhooks

Enter any URL that accepts a POST. Alerts are delivered as JSON, so you can route them into whatever you already run — Home Assistant, n8n, a script, another alerting tool.

> [!NOTE]
> This is a Cloud channel, distinct from the webhooks your self-hosted Dozzle can call directly. See [Alerts](/guide/alerts-and-webhooks) for those, including the Go template variables.

## Browser push

Enable it on the Channels page and allow notifications when your browser asks. Alerts then arrive as desktop notifications.

If nothing arrives after enabling it, the browser most likely denied the permission prompt. Browsers do not re-ask once denied — clear the site's notification permission in browser settings and enable it again. Browser push does not work in a private or incognito window.

## <Icon icon="mdi:bell-sleep-outline" inline /> Making alerts quieter

You should only be interrupted when it matters. If Cloud is noisy, that is a tuning problem, and these are the tools for it.

| Situation                                  | Do this                               |
| ------------------------------------------ | ------------------------------------- |
| One recurring error you already know about | **Mute the pattern**                  |
| Alerts are useful but too frequent         | **Rate one thumbs down**              |
| Planned maintenance, backups, upgrades     | **Mute the pattern before you start** |
| Right alert, wrong app                     | **Disable that channel**              |
| You want none of it, from anywhere         | **Disable every channel**             |

Deleting the alert rule is almost never the right answer. It removes a whole category of monitoring to solve one noisy line.

### Mute a recurring alert

Muting is pattern-based: it silences _this kind of alert_, not just the one in front of you. Later occurrences stay quiet, and anything genuinely different still gets through.

- **On an alert** — open it in Cloud and choose to mute it.
- **In chat** — say "mute this" or "stop telling me about X". The agent states the exact pattern it is about to mute and waits for you to confirm, because a mute is durable and could hide a real failure later.

Muting lasts until you undo it. Ask "what have I muted?" to list your mute rules, and unmute the same way. Muted alerts are still recorded — muting changes what interrupts you, not what is monitored.

### Fewer, not none

If an alert is genuinely useful but arrives too often, rate it **thumbs down** rather than muting it. That is the signal for "keep watching this, interrupt me less". Thumbs up on alerts that got it right helps the same way.

### Repeats are already grouped

Before muting, check whether the problem is repetition. Repeated occurrences of the same failure are folded into a single alert with a count. If you are getting many alerts, they are usually many _different_ problems, or you are over your plan's event allowance and alerts have dropped to raw and ungrouped. See [Plans & Limits](/guide/dozzle-cloud/plans).

### Filter at the source

For a container that is noisy during normal operation, the better fix is upstream: the `dev.dozzle.cloud.min_level` label stops low-severity lines from ever leaving your host. See [Your Data](/guide/dozzle-cloud/your-data).

## Why didn't I get an alert?

**1. Is there a rule for it?** An error in your logs does not by itself produce an alert; something has to be watching for it. The default rule only covers containers exiting with an error — a container that logs errors while staying up needs a log rule.

**2. Is a channel enabled?** A rule with no enabled channel has nowhere to deliver.

**3. Is the instance connected?** If it was offline when the problem happened, nothing was forwarded. See [Connecting Your Instance](/guide/dozzle-cloud/connecting).

**4. Was it grouped into an alert you already got?** Forty failures produce one alert saying forty. That is intended, not a miss.

**5. Did you mute it?** Check your mute rules.

**6. Is the container excluded from forwarding?** See [Your Data](/guide/dozzle-cloud/your-data).

**7. Are you over your plan's limits?** Past the allowance, delivery changes and alerts are sampled.

**8. Check your spam folder**, for email specifically.
