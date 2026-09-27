---
name: hostservice-fanout
description: Per-connection host-follow subscriptions and RetryAndList lock behavior in hostservice; analytics counters are per-request only
metadata:
  type: project
---

- `MultiHostService.followClients` (added after v11.1.2) runs per SSE events stream AND per log stream (`logs.go` calls SubscribeContainersStarted). Each call = manager.Subscribe + goroutine + seen map. Late-joining host triggers a gRPC ListContainers per open log stream.
- `RetriableClientManager.publish` spawns one goroutine per subscriber per host event; subscriber count scales with open log streams.
- `RetryAndList` holds `m.mu` write lock across agent dials (up to args.Timeout). Called by ListAllContainers and ClientServices(true). Blocks List/Find/AgentHostID while any agent is down.
- `analytics.Count` is map lookup + atomic add on a fixed map; only called per request, never per log line (verified 2026-09).
- `analytics.SendBeacon` uses `beaconClient` with a 10s timeout (fixed in 45206b46).
