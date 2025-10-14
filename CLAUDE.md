# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

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

# Install Go tools (protobuf, air hot-reloader)
make tools

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

- **`main.go`** - Application entry point with mode switching (server/swarm/k8s)

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
  - `ContainerViewer/`: Container-specific UI
  - `common/`: Reusable UI components

- **`assets/stores/`** - Pinia stores (auto-imported)
  - `config.ts`: App configuration and feature flags
  - `container.ts`: Container state management
  - `hosts.ts`: Multi-host state
  - `settings.ts`: User preferences

- **`assets/composable/`** - Vue composables (auto-imported)
  - `eventStreams.ts`: SSE connection management
  - `historicalLogs.ts`: Historical log fetching
  - `logContext.ts`: Log filtering and search context
  - `storage.ts`: LocalStorage abstractions

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

### Backend

- The application uses Go 1.25+ with module support
- Certificate generation is required (`make generate` creates shared_key.pem and shared_cert.pem)
- Protocol buffer generation happens via `go generate` directive in `main.go`
- Docker client uses API version negotiation for compatibility

### Authentication

- Three modes: none, simple (file-based users.yml), forward-proxy (e.g., Authelia)
- JWT tokens for simple auth with configurable TTL
- User file location: `./data/users.yml` or `./data/users.yaml`

### Testing

- Go tests use standard `testing` package with testify assertions
- Frontend uses Vitest with `@vue/test-utils`
- Integration tests with Playwright in `e2e/`
- Tests must run with `TZ=UTC` for consistent timestamps

### Container Labels

- `dev.dozzle.name`: Custom container display name
- `dev.dozzle.group`: Group containers together
- Label-based filtering throughout the application

### Deployment Modes

- **Server mode**: Single or multi-host Docker monitoring
- **Swarm mode**: Automatic discovery of Swarm nodes via Docker API
- **K8s mode**: Pod log monitoring in Kubernetes cluster
- **Agent mode**: Lightweight gRPC agent for remote log collection
