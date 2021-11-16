# Build assets
FROM node:17-alpine as node

RUN apk add --no-cache git openssh make g++ util-linux curl python3 && curl -f https://get.pnpm.io/v6.16.js | node - add --global pnpm

WORKDIR /build

# Install dependencies from lock file
COPY pnpm-lock.yaml ./
RUN pnpm fetch --prod

# Copy files
COPY package.json .* vite.config.ts index.html ./

# Copy assets to build
COPY assets ./assets

# Install dependencies
RUN pnpm install -r --offline --prod

# Do the build
RUN pnpm build

FROM golang:1.17.3-alpine AS builder

RUN apk add --no-cache git ca-certificates && mkdir /dozzle

WORKDIR /dozzle

# Copy go mod files
COPY go.* ./
RUN go mod download

# Copy assets built with node
COPY --from=node /build/dist ./dist

# Copy all other files
COPY analytics ./analytics
COPY docker ./docker
COPY web ./web
COPY main.go ./

# Args
ARG TAG=dev

# Build binary
RUN CGO_ENABLED=0 go build -ldflags "-s -w -X main.version=$TAG"  -o dozzle

FROM scratch

ENV PATH=/bin

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /dozzle/dozzle /dozzle

EXPOSE 8080

ENTRYPOINT ["/dozzle"]
