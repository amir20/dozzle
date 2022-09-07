# Build assets
FROM --platform=$BUILDPLATFORM node:18-alpine as node

RUN npm install -g pnpm


WORKDIR /build

# Install dependencies from lock file
COPY pnpm-lock.yaml ./
RUN pnpm fetch --prod

# Copy files
COPY package.json .* vite.config.ts index.html ./

# Copy assets to build
COPY assets ./assets

# Install dependencies
RUN pnpm install -r --offline --prod --ignore-scripts && pnpm build

FROM --platform=$BUILDPLATFORM golang:1.19.1-alpine AS builder

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
