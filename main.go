package main

import (
	"embed"

	// The image has no zoneinfo, so without this TZ is ignored and the
	// auto-update time silently means UTC.
	_ "time/tzdata"

	"github.com/amir20/dozzle/internal/cli"
)

//go:embed all:dist
var content embed.FS

//go:embed shared_cert.pem shared_key.pem
var certs embed.FS

//go:generate protoc --go_out=. --go-grpc_out=. --proto_path=./protos ./protos/rpc.proto ./protos/types.proto
//go:generate protoc --go_out=. --go-grpc_out=. --proto_path=./protos --go_opt=module=github.com/amir20/dozzle --go-grpc_opt=module=github.com/amir20/dozzle ./protos/cloud.proto
//go:generate go tool tygo generate
func main() {
	cli.Main(content, certs)
}
