# Go Performance Reviewer Memory

## Hot Paths Identified

- **Stats processing pipeline**: `ContainerStatsListener.enrich()` -> channel -> `Manager.processStatEvents()` -> `processStatEvent()`. Runs continuously for every container stat tick (~1/sec per container).
- **Log processing pipeline**: `ContainerLogListener` -> `logChannel` (buffered 1000) -> `Manager.processLogEvents()` -> `processLogEvent()`. Every log line from matched containers flows here.
- Both pipelines do `expr.Run()` per subscription per event -- compiled programs are cached but still run per-event.

## Architecture Notes

- `ContainerStore.SubscribeStats` and `SubscribeEvents` both call `statsCollector.Start/Stop` -- multiple subscribers share the same collector. Stop-on-cancel can interfere across subscribers.
- `xsync.Map` (from puzpuzpuz/xsync/v4) used extensively for concurrent maps. No built-in TTL/eviction.
- `ContainerStatEvent` carries full `Container` + `Host` structs by value through channels. `Container` is ~200+ bytes (strings, map, RingBuffer pointer, times).
- TTL caches in both listeners (`cachedContainerInfo`) lack eviction -- grow unboundedly with container churn.

## Existing Patterns

- Semaphore-based concurrency limiting: `sendSem` (weighted 5) for notification dispatch, `maxFetchParallelism` (30) for container fetching.
- Lazy stats collection: stats collector starts on first subscriber, stops when last unsubscribes (but the multi-subscriber Stop race exists).
- `sync.Pool` not currently used in notification/stats paths.
- Cooldown tracking via `xsync.Map[string, time.Time]` per subscription -- also no eviction.

## Key File Paths

- `internal/container/container_store.go` -- Container store with stats collector lifecycle
- `internal/notification/processing.go` -- Log + stat event processing (hot path)
- `internal/notification/stats_listener.go` -- Stats subscription and enrichment
- `internal/notification/log_listener.go` -- Log stream management
- `internal/notification/manager.go` -- Subscription CRUD and listener orchestration
- `internal/notification/types.go` -- Subscription matching (expr evaluation)
