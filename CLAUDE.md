# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Comment Style

**Always use ultra-brief mode for all PR reviews and responses.**

Format:

- Critical issues only (bugs, security, blockers)
- Brief bullet points, no lengthy explanations
- Skip verbose sections (no "Strengths", "Summary", etc.)
- Include file:line references when relevant
- Maximum ~10-15 lines per response

## Project Overview

Dozzle is a lightweight, web-based Docker log viewer with real-time monitoring capabilities. It's a hybrid application with:

- **Backend**: Go (HTTP server, Docker API client, WebSocket streaming)
- **Frontend**: Vue 3 (SPA with Vite, TypeScript)

The application supports multiple deployment modes: standalone server, Docker Swarm, and Kubernetes (k8s).

## Development Commands

### Setup

```bash
# Install dependencies
pnpm install

# Generate certificates and protobuf files
make generate
```

### Development

```bash
# Run full development environment (backend + frontend with hot reload)
make dev

# Alternative: Run backend and frontend separately
pnpm run watch:backend  # Go backend with air (port 3100)
pnpm run watch:frontend # Vite dev server (port 3100)

# Run in agent mode for development
pnpm run agent:dev
```

### Building

```bash
# Build frontend assets
pnpm build
# or
make dist

# Build entire application (includes frontend build)
make build

# Build Docker image
make docker
```

### Testing

```bash
# Run Go tests
make test

# Run frontend tests (Vitest)
pnpm test
# Run in watch mode
TZ=UTC pnpm test --watch

# Type checking
pnpm typecheck
```

### Preview & Other

```bash
# Preview production build locally
pnpm preview
# or
make preview

# Run integration tests (Playwright)
make int
```

## Architecture

### Backend (Go)

The Go backend is organized into these key packages:

- **`internal/web/`** - HTTP server and routing layer
  - Routes defined in `routes.go` using chi router
  - WebSocket/SSE handlers for log streaming (`logs.go`)
  - Authentication middleware and token management (`auth.go`)
  - Container action handlers (`actions.go`)

- **`internal/docker/`** - Docker API client implementation
  - `client.go`: Main Docker client wrapper with container operations
  - `log_reader.go`: Streaming container logs
  - `stats_collector.go`: Real-time container stats collection

- **`internal/agent/`** - gRPC agent for multi-host support
  - Uses Protocol Buffers (protos defined in `protos/`)
  - Enables distributed log collection across Docker hosts

- **`internal/k8s/`** - Kubernetes client support
  - Alternative to Docker client for k8s deployments

- **`internal/support/`** - Support utilities
  - `cli/`: Command-line argument parsing and validation
  - `docker/`: Multi-host Docker management and Swarm support
  - `container/`: Container service abstractions
  - `web/`: Web service utilities

- **`internal/auth/`** - Authentication providers
  - Simple file-based auth (`simple.go`)
  - Forward proxy auth (`proxy.go`)
  - Role-based authorization (`roles.go`)

- **`internal/container/`** - Container domain models and interfaces
  - `event_generator.go`: Log parsing and grouping logic (multi-line, JSON detection)

- **`internal/notification/`** - Alert and notification system
  - `manager.go`: Notification rule evaluation and dispatching
  - `log_listener.go`: Log pattern matching for alerts
  - `dispatcher/`: Notification channel implementations (email, webhook, etc.)

- **`graph/`** - GraphQL API layer
  - `schema.graphqls`: GraphQL schema definitions
  - `*.resolvers.go`: GraphQL resolver implementations

- **`main.go`** - Application entry point with mode switching (server/swarm/k8s/agent)

### Frontend (Vue 3)

The frontend uses file-based routing with these conventions:

- **`assets/pages/`** - File-based routes (unplugin-vue-router)
  - `container/[id].vue`: Single container view
  - `merged/[ids].vue`: Multi-container merged view
  - `host/[id].vue`: Host-level logs
  - `service/[name].vue`: Swarm service logs
  - `stack/[name].vue`: Docker stack logs
  - `group/[name].vue`: Custom grouped logs

- **`assets/components/`** - Vue components (auto-imported)
  - `LogViewer/`: Core log viewing components
    - `SimpleLogItem.vue`: Single-line log entries
    - `ComplexLogItem.vue`: JSON/structured log entries
    - `GroupedLogItem.vue`: Multi-line grouped log entries
    - `ContainerEventLogItem.vue`: Container lifecycle events
    - `SkippedEntriesLogItem.vue`: Placeholder for skipped logs
    - `LoadMoreLogItem.vue`: Load more historical logs
  - `ContainerViewer/`: Container-specific UI
  - `common/`: Reusable UI components
  - `BarChart.vue`: Lightweight bar chart with automatic downsampling
  - `HostCard.vue`: Host overview card with metrics
  - `MetricCard.vue`: Reusable metric display component
  - `ContainerTable.vue`: Container table with historical stat visualization

- **`assets/stores/`** - Pinia stores (auto-imported)
  - `config.ts`: App configuration and feature flags (injected from backend HTML, frozen immutable)
  - `container.ts`: Container state management with EventSource streaming (`/api/events/stream`)
  - `hosts.ts`: Multi-host state
  - `settings.ts`: User preferences (localStorage-backed via profileStorage)
  - `pinned.ts`: Pinned container logs for side-by-side viewing
  - `swarm.ts`, `k8s.ts`: Deployment mode-specific state
  - `announcements.ts`: Feature announcements

- **`assets/composable/`** - Vue composables (auto-imported)
  - `eventStreams.ts`: SSE connection management with buffer-based flushing (250ms debounce)
  - `historicalLogs.ts`: Historical log fetching
  - `logContext.ts`: Log filtering and search context (provide/inject pattern)
  - `scrollContext.ts`: Scroll state management (paused, progress, currentDate)
  - `storage.ts`: LocalStorage abstractions with reactivity
  - `visible.ts`: Log filtering by visible keys for complex logs
  - `containerActions.ts`: Container control operations
  - `duckdb.ts`: DuckDB WASM for SQL queries on logs

- **`assets/modules/`** - Vue plugins
  - `router.ts`: Vue Router configuration
  - `pinia.ts`: Pinia store setup
  - `i18n.ts`: Internationalization

### Communication Flow

1. **Real-time Logs**: Frontend establishes SSE connections to `/api/hosts/{host}/containers/{id}/logs/stream`
2. **Container Events**: SSE stream at `/api/events/stream` pushes container lifecycle events
3. **Stats**: Real-time CPU/memory stats streamed via SSE alongside events
4. **Actions**: POST to `/api/hosts/{host}/containers/{id}/actions/{action}` (start/stop/restart)
5. **Terminal**: WebSocket connections for container attach/exec at `/api/hosts/{host}/containers/{id}/attach`
6. **GraphQL**: POST to `/api/graphql` for queries and mutations (container metadata, historical logs, notifications)

### Build System

- **Frontend**: Vite builds to `dist/` with manifest
- **Backend**: Embeds `dist/` using Go embed directive
- **Hot Reload**: In development, `DEV=true` disables embedded assets, `LIVE_FS=true` serves from filesystem
- **Makefile**: Orchestrates builds and dependency generation

## Important Development Notes

### Frontend

- Auto-imports are configured for Vue composables, components, and Pinia stores (see `vite.config.ts`)
- Icons use unplugin-icons with multiple icon sets (mdi, carbon, material-symbols, etc.)
- Tailwind CSS with DaisyUI for styling
- TypeScript definitions auto-generated in `assets/auto-imports.d.ts` and `assets/components.d.ts`
- **Log Entry Types**: Three types of log messages supported
  - `SimpleLogEntry`: Single-line text logs (`string`)
  - `ComplexLogEntry`: Structured JSON logs (`JSONObject`)
  - `GroupedLogEntry`: Multi-line grouped logs (`string[]`)
- **Type consistency**: Use `LogMessage` type alias instead of `string | string[] | JSONObject` for log entry messages
- **Log Entry Factory Pattern**: Use `LogEntry.create(logEvent)` to instantiate the correct entry type based on `logEvent.t` field
- **EventSource Buffering**: Log streams use buffer-based flushing (250ms debounce, 1000ms max) to batch UI updates
- **Charts/Visualizations**: Custom lightweight implementations (no D3.js)
  - `BarChart.vue`: Self-contained bar chart with responsive downsampling
  - Downsampling algorithm: Averages data into buckets based on available screen width
  - All stat history tracked in `Container.statsHistory` (max 300 items via rolling window)
  - `chartData` is always a rolling window of max 300 items — array length stays constant
  - Uses `ref` (not `computed`) for `downsampledBars` to enable in-place mutation of the last bar, avoiding full re-renders
  - Component instance is reused when switching containers; parent must call exposed `recalculate()` to force refresh

### Backend

- The application uses Go 1.25+ with module support
- Certificate generation is required (`make generate` creates shared_key.pem and shared_cert.pem)
- Protocol buffer generation happens via `go generate` directive in `main.go`
- Docker client uses API version negotiation for compatibility
- **GraphQL API**: Uses gqlgen with schema in `graph/schema.graphqls`, generated code in `graph/generated.go`
  - Run `pnpm codegen` to regenerate GraphQL types
  - Resolvers follow-schema layout in `graph/*.resolvers.go`
- **Service Layer Architecture**:
  - `ClientService` interface abstracts Docker/K8s/Agent backends
  - `MultiHostService` orchestrates multi-host operations
  - `ClientManager` implementations: `RetriableClientManager` (server mode), `SwarmClientManager` (swarm mode)

### Authentication

- Three modes: none, simple (file-based users.yml), forward-proxy (e.g., Authelia)
- JWT tokens for simple auth with configurable TTL
- User file location: `./data/users.yml` or `./data/users.yaml`

### Testing

- Go tests use standard `testing` package with testify assertions
- Frontend uses Vitest with `@vue/test-utils`
- Integration tests with Playwright in `e2e/`
- Tests must run with `TZ=UTC` for consistent timestamps

### Container Stats & Metrics

- Stats are tracked using exponential moving average (EMA) with alpha=0.2
- History stored in rolling window (300 items max) via `useSimpleRefHistory`
- CPU metrics normalized by core count (respects `cpuLimit` or falls back to host `nCPU`)
- Memory metrics include both percentage and absolute usage (`memoryUsage` vs `memory`)
- Stats visualization uses adaptive downsampling for performance

### Container Labels

- `dev.dozzle.name`: Custom container display name
- `dev.dozzle.group`: Group containers together
- Label-based filtering throughout the application

### Deployment Modes

- **Server mode** (default): Single or multi-host Docker monitoring
  - Uses `RetriableClientManager` with local + remote agent clients
- **Swarm mode**: Automatic discovery of Swarm nodes via Docker API
  - Creates gRPC agent server on each node (port 7007)
  - Uses `SwarmClientManager` for node discovery
- **K8s mode**: Pod log monitoring in Kubernetes cluster
  - Implements `container.Client` interface via Kubernetes API
- **Agent mode**: Lightweight gRPC agent for remote log collection
  - Run with `dozzle agent` or `pnpm run agent:dev`
  - Listens on port 7007 with TLS certificate authentication

## Key Architectural Patterns

### Backend Abstraction Layers

The backend follows a clean layered architecture:

```
HTTP Handlers (internal/web)
    ↓
HostService Interface (MultiHostService)
    ↓
ClientService Interface (per host)
    ↓
container.Client Interface
    ↓
Implementation (DockerClient, K8sClient, AgentClient)
```

**When adding new container operations:**

1. Define method in `container.Client` interface (`internal/container/client.go`)
2. Implement in `internal/docker/client.go` (and `internal/k8s/client.go` if applicable)
3. Add wrapper method in `ClientService` interface (`internal/support/container/service.go`)
4. Add HTTP handler in `internal/web/` with appropriate route

### Frontend Data Flow

**Real-time Log Viewing:**

1. User navigates to `/container/{id}` route
2. Page component calls `useContainerStream(container)` composable
3. Composable creates EventSource connection to `/api/hosts/{host}/containers/{id}/logs/stream`
4. Backend streams `LogEvent` objects via SSE
5. Frontend buffers events (250ms debounce, max 1000ms)
6. Batched buffer flushes update reactive `messages` array
7. `LogViewer.vue` renders using appropriate component (`SimpleLogItem`, `ComplexLogItem`, `GroupedLogItem`)
8. When messages exceed `maxLogs` (400), oldest entries replaced or marked as `SkippedLogsEntry`

**Stats Streaming:**

1. `container.ts` store connects to `/api/events/stream` on app init
2. Backend multiplexes container events and stats into single SSE stream
3. `container-stat` events update `Container._stat` and append to `_statsHistory`
4. EMA calculation provides smoothed `movingAverageStat` (alpha=0.2)
5. `ContainerTable.vue` displays mini bar charts using `statsHistory` with downsampling

### Protocol Buffer Flow (Agent Mode)

1. Main server creates `agent.NewClient(endpoint, certs)` for each remote host
2. AgentClient implements `container.Client` interface
3. Method calls translate to gRPC requests defined in `protos/rpc.proto`
4. Remote agent receives gRPC call, delegates to local `DockerClient`
5. Streaming RPCs (logs, stats, events) use bidirectional channels
6. Responses converted back to domain models via `FromProto()` methods

### Log Parsing Pipeline

1. Docker API returns multiplexed stream (8-byte headers + payload)
2. `log_reader.go` parses headers, extracts stdout/stderr type
3. `event_generator.go` receives raw log lines
4. Detection logic identifies:
   - JSON structure → `ComplexLogEntry`
   - Multi-line patterns (stack traces) → `GroupedLogEntry`
   - Single lines → `SimpleLogEntry`
5. Log level extraction via regex patterns
6. `LogEvent` serialized to JSON and sent via SSE
7. Frontend deserializes and renders with appropriate component

## Adding New Features

### Adding a New HTTP Route

1. Define route in `internal/web/routes.go` using chi router:
   ```go
   r.Get("/api/custom-endpoint", h.customHandler)
   ```
2. Implement handler method in appropriate file (e.g., `actions.go`, `logs.go`)
3. Use `hostService` to find container/host via `FindContainer()` or `FindHost()`
4. Return JSON response or establish SSE/WebSocket stream

### Adding a New Log View Type

1. Create route file in `assets/pages/` (e.g., `custom/[id].vue`)
2. Create composable in `assets/composable/eventStreams.ts` (e.g., `useCustomStream()`)
3. Composable should:
   - Build API URL with appropriate filters
   - Create EventSource connection
   - Handle buffering and message batching
   - Return reactive `messages` array and control methods
4. Use `LogViewer.vue` component to render messages
5. Add backend API endpoint if needed (see above)

### Adding a New GraphQL Query/Mutation

1. Define in `graph/schema.graphqls`
2. Run `pnpm codegen` to regenerate types
3. Implement resolver in `graph/schema.resolvers.go`
4. Use `hostService` from resolver context to access backend services
5. Frontend calls via urql client (auto-imported via `@urql/vue`)

### Adding Container Stats/Metrics

1. Add field to `Stat` type in `internal/container/types.go`
2. Update `stats_collector.go` to extract metric from Docker API response
3. Add calculation logic in `docker/calculation.go` if needed
4. Ensure protobuf definition includes field in `protos/rpc.proto`
5. Frontend automatically receives updates via existing SSE stream
6. Update `Container` model in `assets/models/Container.ts` if UI needs access

### Working with Notifications/Alerts

**Backend** (`internal/notification/`):

- `manager.go`: Rule evaluation engine, manages alert state
- `log_listener.go`: Subscribes to container log streams, evaluates rules against incoming logs
- `types.go`: Alert rule definitions (log pattern matching, thresholds)
- `dispatcher/`: Notification channel implementations

**Frontend** (`assets/pages/notifications.vue`, `assets/components/Notification/`):

- `AlertForm.vue`, `DestinationForm.vue`: UI for creating rules
- Rules stored via GraphQL mutations
- Alert state displayed in notification cards

**Adding a new notification channel:**

1. Implement dispatcher interface in `internal/notification/dispatcher/`
2. Register in `manager.go` dispatcher factory
3. Add UI form in `assets/components/Notification/DestinationForm.vue`
4. Add GraphQL schema fields if needed

## Common Development Patterns

### Testing

- Always run Go tests with race detector: `go test -race`
- Frontend tests require `TZ=UTC` for timestamp consistency
- Integration tests use Playwright with `make int` (runs docker-compose setup)
- Use `testify/assert` for Go test assertions

### Hot Reload Development

- `make dev` runs both backend (air) and frontend (vite) with hot reload
- `DEV=true` disables embedded asset serving
- `LIVE_FS=true` serves assets from filesystem instead of embedded
- Backend changes trigger air restart automatically
- Frontend changes trigger vite HMR

### Debugging

- Backend logs: Set `--level debug` flag or `DOZZLE_LEVEL=debug` env var
- Frontend: Vue DevTools browser extension
- GraphQL: Use GraphQL Playground at `/api/graphql` (when enabled)
- SSE streams: Browser DevTools Network tab shows EventSource connections
