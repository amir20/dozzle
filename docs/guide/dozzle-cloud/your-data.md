---
title: Your Data
---

# Your Data

What leaves your host, how to stop it, and what [Dozzle Cloud](/guide/dozzle-cloud) stores once it arrives.

## Linking does not expose your Dozzle

Your Dozzle makes an **outbound** connection to Cloud. Nothing inbound is opened, no port is forwarded, and Cloud cannot reach your instance except over the connection your instance started. If you unlink, that access ends immediately.

Linking also does not add authentication to your self-hosted Dozzle. That is a separate question and an important one: **by default, Dozzle has no login.** Anyone who can reach it on your network can view your logs. If you have exposed Dozzle to the internet or share your network, configure [Authentication](/guide/authentication) on the instance itself. This applies whether or not you link.

Container actions — start, stop, restart — stay refused by your instance unless you turn them on with `DOZZLE_ENABLE_ACTIONS`. See [Actions](/guide/actions).

## Controlling what gets forwarded

The most effective privacy control is not sending something in the first place. By default every running container streams its logs to Cloud while linked. For containers whose info-level chatter has no diagnostic value, or that handle material you would rather keep on the host, filter or opt out with a label.

### `dev.dozzle.cloud.min_level`

| Value                                         | Effect                                                                                                |
| --------------------------------------------- | ----------------------------------------------------------------------------------------------------- |
| _(unset)_                                     | All log lines are forwarded. Default.                                                                 |
| `disabled`                                    | The container is completely skipped. No logs are forwarded to Cloud.                                  |
| `trace`                                       | Same as unset, since trace is the lowest level. Everything is forwarded.                              |
| `debug` / `info` / `warn` / `error` / `fatal` | Only lines at that level or higher are forwarded. Lines without a detected level always pass through. |

An unrecognized value (a typo like `warning` or `wran`) is logged as an error and ignored, so the container streams everything as if the label were unset.

The label is read when the log reader starts. Changing it on a running container takes effect after the container restarts.

```yaml
services:
  zigbee2mqtt:
    image: koenkk/zigbee2mqtt
    labels:
      # Only forward warn/error/fatal to Dozzle Cloud
      - dev.dozzle.cloud.min_level=warn

  noisy-debug-tool:
    image: example/debug
    labels:
      # Don't send anything from this container
      - dev.dozzle.cloud.min_level=disabled
```

The filter runs on your Dozzle instance **before logs leave the host**, so dropped lines never touch the network and never count against your plan. Local log viewing in Dozzle is unaffected.

## What Cloud stores

- **Log lines** forwarded from your linked instances, for full-text search.
- **Events and alerts** that matched your rules, with their investigations and findings.
- **Container and host metadata** — names, images, states, resource usage.
- **Your account** — email address, plan, notification channel settings.
- **Chat history** with the agent.

Everything is scoped to your account; other users cannot see your data. Stored data is kept for your plan's retention window and then deleted automatically. See [Plans & Limits](/guide/dozzle-cloud/plans).

## API keys

Each linked instance authenticates with its own API key. Keys are hashed with BLAKE2b, support expiration, and are never stored in plain text.

Deleting a key on the Instances page immediately and permanently disconnects that instance. The key cannot be recovered or reattached — link the instance again to get a new one. If you believe a key has been exposed, delete it and relink. That is the complete remedy: the old key stops working the moment it is deleted.

## Signing in

Cloud uses GitHub or Google sign-in. There is no separate password to create, and Cloud never sees your GitHub or Google password. If you sign up with one provider and later sign in with the other using the same email address, you reach the same account.

## Stopping collection without closing your account

1. Delete your instances' API keys on the Instances page. Forwarding stops immediately.
2. Disable your channels on the Channels page, so nothing is delivered.

Existing history then ages out on its own within your plan's retention window.

## Closing your account

There is no self-service delete button yet. Email **amir@dozzle.dev** from the address on the account and ask for deletion. If you are on a paid plan, cancel it first from settings so you are not billed again.

Before or instead of that, the steps above remove essentially everything yourself: deleting your API keys stops all collection, and stored data ages out with retention.
