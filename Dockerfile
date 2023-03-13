# Build assets
FROM --platform=$BUILDPLATFORM node:19-alpine as node

RUN npm install -g pnpm

WORKDIR /build

# Install dependencies from lock file
COPY pnpm-*.yaml ./
RUN pnpm fetch --ignore-scripts --no-optional

# Copy package.json and install dependencies
COPY package.json ./
RUN pnpm install --offline --ignore-scripts --no-optional

# Copy assets and translations to build
COPY .* vite.config.ts index.html ./
COPY assets ./assets
COPY locales ./locales

# Build assets
RUN pnpm build

FROM --platform=$BUILDPLATFORM golang:1.20.2-alpine AS builder

RUN apk add --no-cache ca-certificates && mkdir /dozzle

WORKDIR /dozzle

# Copy go mod files
COPY go.* ./
RUN go mod download

# Copy assets built with node
COPY --from=node /build/dist ./dist

# Copy all other files
COPY analytics ./analytics
COPY healthcheck ./healthcheck
COPY docker ./docker
COPY web ./web
COPY main.go ./

# Args
ARG TAG=dev
ARG TARGETOS TARGETARCH

# Build binary
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH CGO_ENABLED=0 go build -ldflags "-s -w -X main.version=$TAG"  -o dozzle


FROM scratch

ENV PATH /bin

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /dozzle/dozzle /dozzle

EXPOSE 8080

ENTRYPOINT ["/dozzle"]
