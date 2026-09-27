---
name: host-follow-streams
description: RetriableClientManager publish ordering and agent stream lifetime pitfalls behind MultiHostService.followClients
metadata:
  type: project
---

- `RetriableClientManager.publish` spawns one goroutine per subscriber per message, so two quick publishes (Available then Removed, or Removed then re-Available) can reach a subscriber in either order. Check any new publish site for order sensitivity.
- `agent.Client.StreamStats/StreamEvents/StreamNewContainers` are one-shot: they return on the first Recv error and nothing reconnects them. An agent restart with a stable host id leaves every follower's streams dead; only a re-key (id change) triggers a resubscribe in `followClients`.
- That re-key resubscribe cannot tell a stream that died with the old process from one opened after the restart, so subscribers created in the restart-to-rekey window (events.go runs reconcileHosts right after subscribing) get duplicate streams.
- Host ids are mostly stable (engine/swarm node id via `container.DerivedHostID`), despite comments saying agents mint ids per process.
