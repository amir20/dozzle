package cli

import (
	"context"
	"embed"
	"fmt"
	"os"

	"github.com/amir20/dozzle/internal/healthcheck"
	"github.com/rs/zerolog/log"
)

type HealthcheckCmd struct{}

func (h *HealthcheckCmd) Run(args Args, embeddedCerts embed.FS) error {
	const agentAddrFile = "/tmp/dozzle-agent.addr"
	if data, err := os.ReadFile(agentAddrFile); err == nil {
		agentAddress := string(data)
		certs, err := ReadCertificates(embeddedCerts, args.CertPath, args.KeyPath)
		if err != nil {
			return fmt.Errorf("failed to read certificates: %w", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), args.Timeout)
		defer cancel()
		log.Info().Str("address", agentAddress).Msg("Making RPC request to agent")
		return healthcheck.RPCRequest(ctx, agentAddress, certs)
	} else {
		log.Info().Str("address", args.Addr).Str("base", args.Base).Msg("Making HTTP request to server")
		return healthcheck.HttpRequest(args.Addr, args.Base)
	}
}
