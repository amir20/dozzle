FROM --platform=$BUILDPLATFORM node:25.9.0-alpine AS node

RUN npm install -g --force corepack && corepack enable

WORKDIR /build

COPY pnpm-*.yaml ./
RUN pnpm fetch --ignore-scripts --no-optional

COPY package.json ./
RUN pnpm install --offline --ignore-scripts --no-optional

COPY vite.config.ts tsconfig.json .prettierrc.cjs .npmrc ./
COPY assets ./assets
COPY locales ./locales
COPY public ./public

RUN pnpm build

FROM docker/compose-bin:v2.36.0 AS compose

FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder

RUN apk add --no-cache ca-certificates upx && mkdir /dozzle

WORKDIR /dozzle

COPY go.* ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY internal ./internal
COPY proto ./proto
COPY types ./types
COPY main.go ./
COPY protos ./protos
COPY shared_key.pem shared_cert.pem ./

COPY --from=node /build/dist ./dist

ARG TAG=dev
ARG TARGETOS TARGETARCH

RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
  GOOS=$TARGETOS GOARCH=$TARGETARCH CGO_ENABLED=0 go build -ldflags "-s -w -X github.com/amir20/dozzle/internal/support/cli.Version=$TAG" -o dozzle

# Compress both binaries
COPY --from=compose /docker-compose ./docker-compose
RUN upx --best dozzle docker-compose

RUN mkdir /data

FROM scratch

COPY --from=builder /data /data
COPY --from=builder /tmp /tmp
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /dozzle/dozzle /dozzle
COPY --from=builder /dozzle/docker-compose /usr/local/bin/docker-compose

EXPOSE 8080

ENTRYPOINT ["/dozzle"]
