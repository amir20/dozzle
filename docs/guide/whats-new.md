---
title: What's New in v11
---

# <Icon icon="mdi:party-popper" inline /> What's New in v11

v11 is the biggest visual change Dozzle has had. Almost every surface was redrawn on one design language: flat, quiet, neutral panels, with colour reserved for the thing that actually needs attention. Sign in with GitHub and OIDC landed too, and the log stream learned a few new formats.

## A new look

- **Sidebar** rebuilt around collapsible groups with a count, container app icons carrying status as a corner badge, and a tinted selected row. Merging a whole group is a button on the group itself.
- **Log stream** redesigned. Timestamps are unboxed and quiet, single lines get a level dot while grouped entries get a rail, and warn and error rows carry a light level tint so they are findable while scrolling.
- **Container title bar** simplified. The name leads, the image is plain dimmed text you click to copy, and pinning moved to a map pin that matches the sidebar's Pinned section.
- **Homepage dashboard**, **command palette**, **toasts**, **attach and shell drawers** and the **container menu** were all redrawn. The menu is grouped into labeled sections instead of one long list.
- **Live log indicator** and the **scroll position readout** are new. The readout floats over the stream, says where in the container's lifetime you are, and leaves once you stop scrolling.
- Every menu moved to the native popover API, so menus no longer get clipped or trapped inside a scrolling panel.

## Sign in with GitHub and OIDC

Users can sign in with a GitHub account or any OIDC provider (Authentik, Keycloak, Pocket ID, Google). It is part of the `simple` provider, so `users.yml` is still the allowlist and still decides who gets in. No account is ever created automatically, and password login keeps working next to it.

```yaml
environment:
  DOZZLE_AUTH_PROVIDER: simple
  DOZZLE_AUTH_GITHUB_CLIENT_ID: Ov23liABCDEFGHIJKLMN
  DOZZLE_AUTH_GITHUB_CLIENT_SECRET: 0123456789abcdef0123456789abcdef01234567
```

See [Sign in with GitHub & OIDC](/guide/authentication/oauth) for the full setup.

## Logs

- OpenTelemetry `severityText` and `severityNumber` are recognized as log levels, with `severityNumber` used when the text is not a level name.
- Pino numeric levels (`30`, `40`, `50`) are parsed.
- Field toggles apply in service and stack views, not only single containers.
- Pinned columns live in the URL, so a side-by-side view is a link you can send someone.

## Alerts and Dozzle Cloud

- Alerts are remembered. They show up on the rule that fired them and as a dot on the container row, and they survive a reload.
- A new cloud rail sits beside the stream with three panels: ask a question about what you are looking at, the metrics behind the live chart, and the alerts that fired on the containers in view. It is mounted only when Cloud is linked, and it collapses to a tab on the edge.
- Cloud tools are scoped to the user who asked, so what the assistant can see matches what that account can see.
- `min_level=disabled` no longer stops metrics along with logs.

## Performance and fixes

- First log lines paint noticeably sooner.
- Log streams the browser silently gave up on now reconnect.
- Stat points in the containers-changed payload are much smaller.
- Merged log streams no longer deadlock the container store.

## Upgrading

Session tokens are now signed with a random secret persisted to `session_secret` in the data directory, next to `users.yml`. **Everyone is signed out once after upgrading.** If `/data` is not writable, Dozzle still boots with an in-memory secret and warns, which means sessions drop on every restart.

The old key was derived from `users.yml`, which was not enough entropy once an account can be proven by OAuth and carry no password hash at all.
