# Build assets
FROM --platform=$BUILDPLATFORM oven/bun:1-slim as bun

WORKDIR /build

# Install dependencies from lock file
COPY bun.lockb package.json ./
RUN bun install --frozen-lockfile

# Copy assets and translations to build
COPY .* *.config.ts *.config.js *.config.cjs ./
COPY assets ./assets
COPY locales ./locales
COPY public ./public

# Build assets
RUN bun run build

FROM --platform=$BUILDPLATFORM golang:1.21.6-alpine AS builder

RUN apk add --no-cache ca-certificates && mkdir /dozzle

WORKDIR /dozzle

# Copy go mod files
COPY go.* ./
RUN go mod download

# Copy assets built with node
COPY --from=bun /build/dist ./dist

# Copy all other files
COPY internal ./internal
COPY main.go ./

# Args
ARG TAG=dev
ARG TARGETOS TARGETARCH

# Build binary
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH CGO_ENABLED=0 go build -ldflags "-s -w -X main.version=$TAG"  -o dozzle

RUN mkdir /data

FROM scratch

ENV PATH /bin
COPY --from=builder /data /data
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /dozzle/dozzle /dozzle

EXPOSE 8080

ENTRYPOINT ["/dozzle"]
