---
name: analytics-beacon
description: Usage counters (internal/analytics/usage.go) and beacon fact gathering (internal/web/beacon.go) cost profile
metadata:
  type: project
---

- `analytics.Count` is a fixed-map lookup + atomic add; callers are per-action/per-request only (no per-log-line use as of 2026-09). Safe.
- Cost lives in `beaconFacts`: `Hosts()` fan-out (plus a second one via `LocalHost()`), `config.Load` YAML read, subscriptions/dispatchers. Runs per events-stream connect (guarded only by an in-flight CAS, no time throttle) and in shutdown `FlushUsage` (3s budget, also does `ListAllContainers`).
- `usageRunner`/`usageFlusher` are package-global atomic func pointers set in `CreateServer`.
