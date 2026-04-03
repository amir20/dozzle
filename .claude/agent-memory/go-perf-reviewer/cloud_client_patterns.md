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
- `apiKeyFunc` closure polls `hostService.Dispatchers()` to find cloud API key
- PermissionDenied from server causes permanent stop (no retry)
- Tool definitions are static but re-serialized on each ListTools request

**How to apply:** When reviewing future changes to this package, watch for: timer leaks in the reconnect loop, unnecessary re-serialization of static data, and gRPC connection lifecycle issues.
