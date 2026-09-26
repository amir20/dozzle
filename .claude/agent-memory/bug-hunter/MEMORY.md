# Bug Hunter Agent Memory

## Codebase Patterns

### Notification System Architecture

- `notification.Manager` owns subscriptions (xsync.Map) and dispatchers (xsync.Map)
- Two processing goroutines: `processLogEvents` and `processStatEvents` started in `NewManager`
- Stats listener has start/stop lifecycle; log listener is always-on once started
- `MultiHostService` wraps Manager and handles config persistence + agent broadcast

### Known Bug-Prone Areas

- **broadcastNotificationConfig**: Has historically missed fields when converting between internal and types packages (APIKey, Prefix, ExpiresAt were missed)
- **TriggeredContainerIDs lazy init**: Race condition risk - initialized lazily in AddTriggeredContainer without sync
- **Channel close handling**: enrich() in stats_listener doesn't check for closed channel, can hot-spin
- **LogAlertFields canSave**: Allows empty logExpression (no error = valid), creates dead subscriptions

### Type Mapping Gotchas

- `container.ContainerStat` -> `types.NotificationStat`: field names differ (CPUPercent vs cpu expr tag)
- `notification.DispatcherConfig` -> `types.DispatcherConfig`: must copy ALL fields including APIKey, Prefix, ExpiresAt
- Frontend `NotificationRule.cooldown` is optional, backend defaults to 300 via `GetCooldownSeconds()`

### Concurrency Model

- xsync.Map used throughout for concurrent access (subscriptions, dispatchers, activeStreams, containers)
- Subscription fields use atomic types: TriggerCount (atomic.Int64), LastTriggeredAt (atomic.Pointer)
- MetricCooldowns uses xsync.Map for per-container cooldown tracking
- sendSem (semaphore.Weighted=5) limits concurrent notification sends

### Container Update (feat/container-update)

- **progressCh close contract**: Docker and Agent implementations close progressCh via defer; K8s does NOT, causing handler hang
- **NetworkSettings nil risk**: `docker.InspectResponse.NetworkSettings` is a pointer; `ContainerCreate` doesn't nil-check before accessing `.Networks`
- **Destructive recreate**: stop->remove->create->start has no rollback if create fails after remove
- **SSE parsing in frontend**: Uses manual ReadableStream reader, not EventSource; no AbortController cleanup on unmount

### Auth / JWT token confusion

- Session JWT = any token signed with session key WITHOUT `token_use` claim (`auth.isSessionClaims`). Any new token type signed with that key must set `token_use`, and every `jwtauth.FromContext` consumer (e.g. oidc_logout.go LogoutRedirect) must be checked for the claim
- Session cookie is SameSite=Lax: cross-site CSRF/framing is blocked, but same-site sibling subdomains (common in homelabs) are not, so path-exact security header checks in index.go matter (vue-router matches case-insensitively and with trailing slash)
