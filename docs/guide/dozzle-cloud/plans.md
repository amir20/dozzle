---
title: Plans & Limits
---

# Plans & Limits

Plans, prices, and allowances are listed on the [Dozzle Cloud pricing page](https://cloud.dozzle.dev/pricing), which is always current. This page covers what the limits look like from your side when you run into them.

## Alerts suddenly turned raw and repetitive

You are most likely over your monthly event allowance. Nothing breaks: event history keeps recording, but triage pauses, roughly one in ten events comes through as a raw alert, and repeats are no longer folded into one alert with a count. Check the usage page in Cloud first. Allowances reset at the start of each month, and this applies on paid plans too.

## A search returns nothing from last week

Search only reaches back as far as your plan's retention window. Anything older has already been deleted, even though it happened. Asking for a longer stats window than your plan allows returns the window you actually have rather than an error.

## The instance limit

The free plan links one instance at a time. Linking a second shows a limit message. You have two options:

- **Move the slot.** Delete the existing instance's API key on the Instances page, then link the new one. This is permanent for the old instance. Its history stays, but you would have to link it again from scratch.
- **Upgrade** to keep both connected at once.

## Checking your usage

The usage page in Cloud shows events, log bytes, and assistant chats used this month against your allowance. You can also ask in chat: "how much have I used this month?".

## Changing or cancelling

Upgrade from the pricing page or from settings. Billing is handled by Stripe; payment methods, invoices, and receipts are managed there through the billing link in your settings.

Cancelling stops future charges and moves you to the free plan at the end of the period you have paid for. Your account and history stay.
