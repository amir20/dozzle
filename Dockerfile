# Build assets
FROM node:current-alpine as node

RUN apk add --no-cache git openssh python make g++ util-linux

WORKDIR /build

# Install dependencies
COPY package*.json yarn.lock ./
RUN yarn install --ignore-scripts --network-timeout 1000000

# Copy config files
COPY .* webpack*.js ./

# Copy assets to build
COPY assets ./assets

# Do the build
RUN yarn build

FROM golang:1.16-alpine AS builder

RUN apk add --no-cache git ca-certificates
RUN mkdir /dozzle

WORKDIR /dozzle

# Needed for assets
RUN go get -u github.com/gobuffalo/packr/packr

# Copy go mod files
COPY go.* ./
RUN go mod download

# Copy assets built with node
COPY --from=node /build/static ./static

# Copy all other files
COPY . .

# Compile static files
RUN packr -z

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
