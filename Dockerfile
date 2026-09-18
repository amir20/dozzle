# Build assets
FROM --platform=$BUILDPLATFORM node:25.9.0-alpine AS node

# bun only installs packages. vite and the build scripts still run on node.
COPY --from=oven/bun:1.3.14-alpine /usr/local/bin/bun /usr/local/bin/bun

ENV CI=true

WORKDIR /build

# docs/package.json is a workspace member, so the frozen lockfile needs it present
COPY package.json bun.lock bunfig.toml ./
COPY docs/package.json ./docs/
RUN --mount=type=cache,target=/root/.bun/install/cache \
  bun install --frozen-lockfile --ignore-scripts

# Copy assets and translations to build
COPY vite.config.ts tsconfig.json .prettierrc.cjs ./
COPY assets ./assets
COPY locales ./locales
COPY public ./public
COPY scripts ./scripts

# Build assets
RUN bun run build

FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS builder

RUN apk add --no-cache ca-certificates && mkdir /dozzle

WORKDIR /dozzle

# Copy go mod files
COPY go.* ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# Copy all other files
COPY internal ./internal
COPY proto ./proto
COPY types ./types
COPY main.go ./
COPY protos ./protos
COPY shared_key.pem shared_cert.pem ./

# Copy assets built with node
COPY --from=node /build/dist ./dist

# Args
ARG TAG=dev
ARG TARGETOS TARGETARCH

# Build binary
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
  GOOS=$TARGETOS GOARCH=$TARGETARCH CGO_ENABLED=0 go build -ldflags "-s -w -X github.com/amir20/dozzle/internal/cli.Version=$TAG" -o dozzle

RUN mkdir /data

# Optional variant published as :alpine for platforms that bind-mount a shell
# wrapper over the entrypoint. Must stay above the scratch stage so that the
# last stage remains the default build target.
FROM alpine:3.24 AS alpine

COPY --from=builder /data /data
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /dozzle/dozzle /dozzle

EXPOSE 8080

ENTRYPOINT ["/dozzle"]

FROM scratch

COPY --from=builder /data /data
COPY --from=builder /tmp /tmp
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /dozzle/dozzle /dozzle

EXPOSE 8080

ENTRYPOINT ["/dozzle"]
