package cli

import (
	"embed"
	"fmt"

	dozzle_mcp "github.com/amir20/dozzle/internal/mcp"
	"github.com/rs/zerolog/log"
)

type MCPCmd struct{}

func (m *MCPCmd) Run(args Args, embeddedCerts embed.FS) error {
	if args.Mode != "server" {
		return fmt.Errorf("mcp command is only available in server mode")
	}

	multiHostService := CreateMultiHostService(embeddedCerts, args)
	if multiHostService.TotalClients() == 0 {
		return fmt.Errorf("could not connect to any Docker Engine")
	}

	log.Info().Msgf("Dozzle MCP server version %s", args.Version())
	log.Info().Int("clients", multiHostService.TotalClients()).Msg("Connected to Docker")

	mcpServer := dozzle_mcp.NewServer(multiHostService, args.Filter, args.Version())
	return mcpServer.ServeStdio()
}
