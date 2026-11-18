---
title: In Your Dozzle
description: What changes in your own Dozzle after you link it to Dozzle Cloud, including Ask Dozzle, metrics history, and alerts that survive a reload.
---

# In Your Dozzle

What changes in your own Dozzle once an instance is [linked](/guide/dozzle-cloud/connecting). Everything here lives in the Dozzle UI you already use, next to the logs it is about, rather than on a separate site you have to go to.

## Nothing local is gated

Dozzle does not lose features when Cloud is not configured. Alerts still fire, still splice into the log stream, and still reach your webhooks. What linking adds is **memory**: the same alert is still there after a refresh, a restart, and a week later.

That is the whole line between the two. Dozzle owns the container in front of you and can act on it. Cloud owns what is history, what spans instances, and what belongs to your account.

So a Cloud-less install shows an empty history section with a line saying what it would hold, not a locked card:

> Alerts show up here once this instance is linked to Dozzle Cloud. Until then they appear in the log stream and are forgotten on refresh.

## The Dozzle Cloud drawer

Ask Dozzle, Metrics and Alerts are tabs in one drawer that opens from the right. It exists only where Cloud is linked, on every page.

A strip of icons on the right edge is how you find it: each icon opens its tab in the drawer. Hiding the strip is remembered, and a small tab on the edge brings it back, the way the sidebar collapses on the other side. On a phone there is no strip, and the drawer opens full screen from the toolbar or the command palette.

The drawer covers the logs while it is open, so anything in it that points back at the stream (show the lines, a moment on a chart) closes it first and then scrolls the stream there. <kbd>Esc</kbd> closes it.

Metrics and Alerts are about the containers on screen, so they appear only on a page that shows one. On the home page or in settings the drawer offers the chat alone.

### <Icon icon="mdi:message-outline" inline /> Ask Dozzle

A question about what you are looking at. On a log view it is answered against the logs in it; on any other page, against the whole instance. Above the composer is the evidence that goes with the question: which containers or host are in view, how many lines, and the log line you pointed at if you started from a row's menu. Nothing is sent silently.

Answers end in Dozzle rather than in a link out. When the answer is about a moment, the action drives your own stream to it.

Open it from the **Ask Dozzle** button beside the search bar, with <kbd>⇧</kbd><kbd>⌘</kbd><kbd>K</kbd> on any page, by pressing <kbd>.</kbd> in a container's logs, from the container menu, or from a log row. In the <kbd>⌘</kbd><kbd>K</kbd> palette, a query that reads like a question puts **Ask** first so <kbd>Enter</kbd> sends it, and <kbd>⇧</kbd><kbd>⌘</kbd><kbd>↵</kbd> asks from any row. On Windows and Linux, <kbd>Ctrl</kbd> stands in for <kbd>⌘</kbd>.

### <Icon icon="mdi:chart-line" inline /> Metrics

Dozzle keeps 300 stat samples in the browser and nothing behind them, so "was it like this an hour ago?" has no local answer. This panel reads back the samples your instance has been pushing all along: CPU, memory, network and disk over the last 1h, 6h or 24h. Each bar is an average with the highest sample drawn faintly behind it, so a short spike stays visible however long the window is, and hovering any chart reads all of them back at the same moment. Network and disk add up what went out and what came in over the window. Pro adds a 7 day window.

### <Icon icon="mdi:bell-outline" inline /> Alerts

What fired on the containers currently in view. The [notifications](/guide/alerts-and-webhooks) page answers the same question for the whole instance; this tab is scoped to what is on screen, which is the only reason it earns a place in Dozzle. Opening it clears the unseen dot on the bell.

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
