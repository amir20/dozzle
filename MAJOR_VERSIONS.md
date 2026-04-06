# Dozzle Major Version Highlights

A summary of major features introduced with each major version bump since 2020.

## v2.0.0 — 2020-06-27

- Theme overhaul using CSS variables
- Progress notification bar
- Toggle between new light and dark themes

## v3.0.0 — 2020-09-06

- Live container stats (CPU/memory) streamed in real time
- Pinned tabs and improved mobile/responsive layout
- Major UI overhaul

## v4.0.0 — 2022-08-17

- First-class JSON log support
- Jump-to-context, soft wraps, clear-logs action
- Auto color scheme, dark mode polish
- Container stats panel, total CPU/mem usage
- Healthcheck endpoint and "wait for docker" startup option

## v5.0.0 — 2023-09-23

- Multi-host support with parallel client connections
- New homepage dashboard with all containers, sortable table, pagination, bar charts
- Exponential moving average for stats
- Container pinning, keyboard shortcut overlay
- stdout/stderr stream separation
- i18n: Chinese, German added; locale infrastructure
- Refactored UI with faster components

## v6.0.0 — 2024-01-01

- **Forward-proxy authentication** (Authelia, etc.) and `users.yml` file-based auth
- Container actions: start/stop/restart from the UI
- Hot-reload of users.yml without restart
- Custom headers for forward-proxy auth
- Settings synced to disk for authenticated users
- Toast notifications, release list, redirect-to-new-container
- Removed legacy auth model (breaking)

## v7.0.0 — 2024-05-24

- **Docker Swarm mode** with stacks and services on remote hosts
- Host cards on dashboard with per-host stats
- Container grouping by stack/compose
- Background stats collection (up to 5 min) with idle deactivation
- LogFmt parser support
- Compact mode, draggable search, alt-click split panes
- Many new locales (French, Italian, Polish, Danish, Turkish, …)
- `generate` subcommand for users.yml

## v8.0.0 — 2024-07-05

- **Swarm mode rebuilt on gRPC agents** (breaking architecture change)
- Critical/severe log levels
- Stacks and services in fuzzy search
- Improved search with full match scrolling
- Container start events shown inline

## v9.0.0 — 2026-01-06

- **Kubernetes mode** with k8s-specific menu
- **User roles** and `dozzle_*` role mapping; logout URL for forward proxy
- Historical stats on homepage, hosts, and containers
- Permanent links to specific past log lines
- Grouping by `dev.dozzle.group` label, log message grouping
- Shell resize support, action toolbar in menu
- Settings page, collapsible side menu sections
- Parallel container fetching, gRPC compression
- CLEF (`@l`) log level extraction
- New locales: Korean, Indonesian, Dutch

## v10.0.0 — 2026-02-10

- **Dozzle Cloud** integration (bidirectional gRPC tool execution)
- **Notifications & alerts**: full notifications page, webhooks, dispatchers, Go template support, test connections
- Notifications work across agents
- Per-container network usage stats (with mobile view)
- Coolify label fallbacks for container name/group
- Alert creation shortcut, JSON syntax in templates
