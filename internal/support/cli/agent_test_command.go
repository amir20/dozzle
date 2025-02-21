package cli

import (
	"context"
	"embed"
	"fmt"

	"github.com/amir20/dozzle/internal/agent"
	"github.com/rs/zerolog/log"
)

type AgentTestCmd struct {
	Address string `arg:"positional"`
}

func (at *AgentTestCmd) Run(args Args, embeddedCerts embed.FS) error {
	certs, err := ReadCertificates(embeddedCerts)
	if err != nil {
		return fmt.Errorf("error reading certificates: %w", err)
	}

	log.Info().Str("endpoint", args.AgentTest.Address).Msg("Connecting to agent")

	agent, err := agent.NewClient(args.AgentTest.Address, certs)
	if err != nil {
		return fmt.Errorf("error connecting to agent: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), args.Timeout)
	defer cancel()
	host, err := agent.Host(ctx)
	if err != nil {
		return fmt.Errorf("error fetching host info for agent: %w", err)
	}

	log.Info().Str("endpoint", args.AgentTest.Address).Str("version", host.AgentVersion).Str("name", host.Name).Str("id", host.ID).Msg("Successfully connected to agent")

	return nil
}
