---
title: In Your Dozzle
---

# In Your Dozzle

What changes in your own Dozzle once an instance is [linked](/guide/dozzle-cloud/connecting). Everything here lives in the Dozzle UI you already use, next to the logs it is about, rather than on a separate site you have to go to.

## Nothing local is gated

Dozzle does not lose features when Cloud is not configured. Alerts still fire, still splice into the log stream, and still reach your webhooks. What linking adds is **memory**: the same alert is still there after a refresh, a restart, and a week later.

That is the whole line between the two. Dozzle owns the container in front of you and can act on it. Cloud owns what is history, what spans instances, and what belongs to your account.

So a Cloud-less install shows an empty history section with a line saying what it would hold, not a locked card:

> Alerts show up here once this instance is linked to Dozzle Cloud. Until then they appear in the log stream and are forgotten on refresh.

## The cloud rail

A strip of icons on the right edge of the log view, with the panel one of them opens. It is mounted only where Cloud is linked and only on a view that has logs on screen, so it never appears on the home page or in settings.

The panel sits **beside** the stream rather than over it: the page reserves exactly its width, so nothing on the rail ever covers the lines it is talking about. On a phone there is no room for a permanent strip, so the panel becomes a full-screen sheet opened from the toolbar or the command palette.

Hiding the rail is remembered. A small tab on the edge brings it back, the way the sidebar collapses on the other side.

### <Icon icon="mdi:message-outline" inline /> Ask Dozzle

A question about the view you are looking at, answered against the logs in it. Above the composer is the evidence that goes with the question: which containers are in view, how many lines, and the log line you pointed at if you started from a row's menu. Nothing is sent silently.

Answers end in Dozzle rather than in a link out. When the answer is about a moment, the action drives your own stream to it.

Open it with <kbd>Shift</kbd> + <kbd>⌘</kbd> + <kbd>K</kbd>, from the container menu, or from a log row.

### <Icon icon="mdi:chart-line" inline /> Metrics

Dozzle keeps 300 stat samples in the browser and nothing behind them, so "was it like this an hour ago?" has no local answer. This panel reads back the samples your instance has been pushing all along: CPU and memory over the last 1h, 6h or 24h, with the peak called out and hover reading the chart back to you. Pro adds a 7 day window.

### <Icon icon="mdi:bell-outline" inline /> Alerts

What fired on the containers currently in view. The [notifications](/guide/alerts-and-webhooks) page answers the same question for the whole instance; this panel is scoped to what is on screen, which is the only reason it earns a place beside the stream. Opening it clears the unseen dot on the bell.

## Alerts that survive a reload

Once alerts are remembered, they show up in three more places:

- **On the rule that fired them**, so a rule you wrote months ago can be judged by what it actually caught.
- **As a dot on the container row** in the container table, coloured by severity. Clicking it opens a small panel with the headline, the level, when it fired, how many events folded into it, and a one-line summary.
- **In the activity list** on the notifications page, filterable and readable long after the toast is gone.

## Show me the lines

Every one of those surfaces ends in the same action, and it is always the primary one.

**Show me the lines** opens the historical view of that container, scrolled to the moment the alert or finding describes, with the exact line highlighted and the search term already filled in. Metric and event alerts carry no log line, so those land on the moment rather than on a row.

This is the one move Dozzle can make that Cloud cannot, which is why these panels live in Dozzle at all. Linking out is kept for the things Cloud genuinely does better: billing and API keys, the report archive, cross-instance rollups, and full investigation transcripts.

## Findings

Findings are not alerts. An alert is a rule you wrote; a finding is something Cloud noticed that no rule covers. They are never listed alongside each other, and the findings surface exists only when Cloud is configured, rather than sitting in the nav as a permanent advert.

See [Dozzle Cloud](/guide/dozzle-cloud) for what findings cover on each plan.

## Turning it off

Unlinking the instance removes every surface on this page and leaves Dozzle exactly as it was: alerts still fire, still appear in the stream, and are forgotten on refresh. See [Your Data](/guide/dozzle-cloud/your-data) for what stops leaving your host.
