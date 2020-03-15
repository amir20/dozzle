FROM golang:alpine AS builder
RUN apk --update add ca-certificates
RUN mkdir /dozzle
WORKDIR /dozzle
COPY go.* .
RUN go mod download
COPY . .
RUN go build -o dozzle main.go

FROM scratch
ENV PATH=/bin
ENV DOCKER_API_VERSION 1.38
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /dozzle/dozzle /
ENTRYPOINT ["/dozzle"]
