# Build assets. Pinned to the build platform: the output is platform independent,
# and bun ships no arm/v6 or arm/v7 image at all, so a cross build cannot run it
# on the target.
#
# No node in this stage. `bun run build` runs vite and compress-dist.js on the
# bun runtime (see the --bun flags in package.json).
FROM --platform=$BUILDPLATFORM oven/bun:1.4.2-alpine AS assets

ENV CI=true

WORKDIR /build

# Install dependencies from lock file
COPY package.json bun.lock bunfig.toml ./
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

# Copy assets built in the assets stage
COPY --from=assets /build/dist ./dist

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
