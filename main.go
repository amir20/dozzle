package main

import (
	"embed"

	// The image has no zoneinfo, so without this TZ is ignored and the
	// auto-update time silently means UTC.
	_ "time/tzdata"

	"github.com/amir20/dozzle/internal/cli"
	"github.com/rs/zerolog/log"
)

//go:embed all:dist
var content embed.FS

//go:embed shared_cert.pem shared_key.pem
var certs embed.FS

//go:generate protoc --go_out=. --go-grpc_out=. --proto_path=./protos ./protos/rpc.proto ./protos/types.proto
//go:generate protoc --go_out=. --go-grpc_out=. --proto_path=./protos --go_opt=module=github.com/amir20/dozzle --go-grpc_opt=module=github.com/amir20/dozzle ./protos/cloud.proto
//go:generate go tool tygo generate
func main() {
	cli.ValidateEnvVars(cli.Args{}, cli.AgentCmd{})
	args, subcommand := cli.ParseArgs()

	// No subcommand means the server, whose flags are the top-level ones.
	var cmd cli.Runnable = cli.ServerCmd{Content: content}
	if subcommand != nil {
		runnable, ok := subcommand.(cli.Runnable)
		if !ok {
			log.Fatal().Msg("Invalid command")
		}
		cmd = runnable
	}
	if err := cmd.Run(args, certs); err != nil {
		log.Fatal().Err(err).Msg("Failed to run command")
	}
}
