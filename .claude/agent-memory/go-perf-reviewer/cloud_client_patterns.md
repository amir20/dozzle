---
name: Cloud gRPC Client Patterns
description: Architecture of internal/cloud/ package - gRPC bidirectional streaming for cloud tool calls
type: project
---

Cloud client (`internal/cloud/client.go`) uses a bidirectional gRPC stream (`ToolStream`) to receive tool requests from Dozzle Cloud and send responses.

Key architecture:

- `Client.Run()` is the reconnect loop with exponential backoff
- `Client.connect()` creates a new `grpc.ClientConn` per reconnect (potential optimization: reuse connection)
- `sendMu sync.Mutex` protects concurrent `stream.Send()` calls (tool calls dispatch to goroutines)
- `toolSem` (weighted semaphore, max 5) limits concurrent tool execution
- `apiKeyFunc` closure provides cloud API key; empty string means no cloud configured
- PermissionDenied from server causes permanent stop (no retry)
- Tool definitions cached via `sync.Once` + `cachedTools` field (fixed from prior re-serialization issue)
- `Notify()` / `startCh` pattern ensures zero overhead for non-cloud users
- Tool dispatch in `tools.go` uses typed proto responses (`CallToolResponse` with oneof `Result`)

Known perf issues found (2026-04):

- `containsIgnoreCase` in `tools_helpers.go` allocates two lowered strings per call; used in filter loops
- `executeInspectContainer` calls `buildHostNameMap` to resolve a single host name
- `executeFetchContainerLogs` doesn't drain log channel after early break (potential goroutine leak)
- `fmt.Sprintf("%v", event.Message)` in log path uses reflect-based formatting

**How to apply:** When reviewing future changes to this package, watch for: undrained channels from streaming APIs, per-call allocations in filter loops, and unnecessary API calls for single lookups.
