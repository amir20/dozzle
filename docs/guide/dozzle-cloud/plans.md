---
title: Plans & Limits
---

# Plans & Limits

What each plan includes, what counts against it, and what happens when you go over.

The free plan is the complete alerting product, not a trial of one. What the paid plans buy is the proactive half — the review that reads your logs and finds what never fired an alert — plus room to run.

## Plans

|                                                      |     Free     |     Pro      |     Team      |
| ---------------------------------------------------- | :----------: | :----------: | :-----------: |
| Price                                                |      $0      |  $5 / month  |  $15 / month  |
| Findings (reads your logs, finds what never alerted) |   1 / week   |  All, daily  |  All, daily   |
| Fix included with each finding                       |      —       |      ✓       |       ✓       |
| Triage inspects containers and logs when unsure      |      —       |      ✓       |       ✓       |
| Full investigation on demand                         |      —       |      ✓       |       ✓       |
| Triaged events per month                             |    2,000     |     50K      |     250K      |
| Searchable logs                                      | 10 GB · 24 h | 50 GB · 30 d | 100 GB · 30 d |
| Alert and event history                              |    1 day     |   14 days    |    30 days    |
| Stats history (CPU, memory, network, disk)           |     24 h     |   30 days    |    30 days    |
| Connected instances                                  |      1       |  Unlimited   |   Unlimited   |
| Assistant chats per month                            |      10      |     200      |     1,000     |
| Priority support                                     |      —       |      ✓       |       ✓       |

Smart alerts, repeat folding, suppression and severity filters, search indexing, stats monitoring, every notification channel, container actions, and unlimited MCP access are on **every** plan, including free.

Current pricing lives at [cloud.dozzle.dev](https://cloud.dozzle.dev).

## What counts as a triaged event

A **triaged event** is a container event or matching log line that went through the triage pipeline, which decides whether to send a new alert, fold it into an existing one, or stay silent. You are paying for that work, not for raw storage.

Ordinary log lines are not events. They count toward searchable log volume instead. So a very chatty container costs storage while a container in a crash loop costs events.

If a container exits 47 times, that is 47 events against the limit — but one alert that says 47. That is the whole point.

**Findings cost neither.** The log review reads your logs rather than your alert history, so it does not touch the event count, and it works with no alert rules configured at all. Only your monthly log volume applies.

**Search and stats are free on every plan.** Search indexing is on from the moment an instance connects, and CPU, memory, network, and disk series stream from your instances at no charge. The plan only changes how much and how far back.

## Going over the allowance

Nothing breaks. You drop into sampling mode:

- Triage pauses.
- Event history keeps recording, so nothing is lost.
- Roughly one in ten events comes through as a **raw** alert, so you can still see what is happening.
- Repeats are no longer folded into a single alert with a count.

The practical effect is that alerts get noisier and less useful rather than disappearing, and you will feel it in your inbox before you notice it on a usage page. If your alerts have suddenly become raw and repetitive, check usage first.

This applies on paid plans too. Allowances reset at the start of each month.

> [!TIP]
> Most accounts never come close. Only a genuine firehose — a container crash-looping for days — goes past the free allowance, and the alert naming that container arrives long before the limit does.

## Retention

Retention sets how far back your history goes: alerts, events, and log search results. On free that is one day, so a search for something from last week returns nothing even though it happened. That is the most common reason a search appears to "lose" data.

Stats retention is 24 hours on free and 30 days on paid plans. Asking for a longer window than your plan allows returns the window you actually have rather than an error.

## The instance limit

The free plan links one instance at a time. Linking a second shows a limit message. You have two options:

- **Move the slot.** Delete the existing instance's API key on the Instances page, then link the new one. This is permanent for the old instance — its history stays, but you would have to link it again from scratch.
- **Upgrade** to keep both connected at once.

The limit is about how much a free account forwards, not a lock on a feature. Everything else on free works with the instance you have linked.

## Checking your usage

The usage page in Cloud shows events, log bytes, and assistant chats used this month against your allowance. You can also ask in chat: "how much have I used this month?".

## Changing or cancelling

Upgrade from the pricing page or from settings. Billing is handled by Stripe; payment methods, invoices, and receipts are managed there through the billing link in your settings.

Cancelling stops future charges and moves you to the free plan at the end of the period you have paid for. Your account and history stay.
