---
title: Default Profile
---

# Default Profile

Dozzle persists per-user UI preferences (theme, locale, pinned containers, collapsed groups, visible JSON keys, etc.) to disk under `/data/<username>/profile.json`. When [authentication](/guide/authentication) is disabled, or for any user before they have signed in and customized their settings, Dozzle falls back to a special profile named `__default__`.

You can ship a pre-configured profile by creating the file `/data/__default__/profile.json`. Anonymous visitors and any new user without a saved profile will load these settings on first visit.

## File Location

```
/data/__default__/profile.json
```

If the file does not exist, Dozzle starts with built-in defaults. You only need to create it if you want to override them.

## Example

```json
{
  "settings": {
    "showTimestamp": true,
    "showStd": false,
    "showAllContainers": false,
    "softWrap": true,
    "collapseNav": false,
    "smallerScrollbars": false,
    "search": false,
    "compact": false,
    "menuWidth": 250,
    "size": "medium",
    "lightTheme": "auto",
    "hourStyle": "auto",
    "dateLocale": "auto",
    "locale": "en",
    "groupContainers": "stack",
    "automaticRedirect": ""
  },
  "pinned": [],
  "visibleKeys": [],
  "collapsedGroups": []
}
```

All fields are optional — include only the ones you want to override.

## Available Settings

| Field               | Type    | Description                                                          |
| ------------------- | ------- | -------------------------------------------------------------------- |
| `showTimestamp`     | boolean | Show timestamps next to each log line                                |
| `showStd`           | boolean | Show stdout/stderr stream indicator                                  |
| `showAllContainers` | boolean | Include stopped containers in the sidebar                            |
| `softWrap`          | boolean | Wrap long log lines instead of horizontal scroll                     |
| `collapseNav`       | boolean | Start with the sidebar collapsed                                     |
| `smallerScrollbars` | boolean | Use thinner scrollbars                                               |
| `search`            | boolean | Enable inline search by default                                      |
| `compact`           | boolean | Compact log row spacing                                              |
| `menuWidth`         | number  | Sidebar width in pixels                                              |
| `size`              | string  | Font size: `small`, `medium`, `large`                                |
| `lightTheme`        | string  | Theme preference: `auto`, `light`, `dark`                            |
| `hourStyle`         | string  | Time format: `auto`, `12`, `24`                                      |
| `dateLocale`        | string  | Locale used for date/time formatting (e.g. `en-US`, `de-DE`, `auto`) |
| `locale`            | string  | UI language (e.g. `en`, `fr`, `de`)                                  |
| `groupContainers`   | string  | Default sidebar grouping (e.g. `stack`, `none`)                      |
| `automaticRedirect` | string  | Path to redirect to on load                                          |

The top-level fields `pinned`, `visibleKeys`, and `collapsedGroups` accept arrays and let you pre-pin containers or pre-collapse groups for first-time visitors.

## How It Works

- On page load, Dozzle reads `/data/<username>/profile.json` for the signed-in user, or `/data/__default__/profile.json` when no user is authenticated.
- When a user changes a setting in the UI, the new value is persisted under their own username (or back into `__default__` when auth is disabled).
- The `__default__` profile is therefore both the **template for new visitors** and the **live profile for the anonymous user** in unauthenticated deployments.

::: tip
If you only want to seed defaults but still let the anonymous user customize them at runtime, mount the file read-only — Dozzle will fail to persist changes but the UI continues to work.
:::
