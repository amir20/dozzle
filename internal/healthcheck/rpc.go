package healthcheck

import (
	"context"
	"crypto/tls"

	"github.com/amir20/dozzle/internal/agent"
	"github.com/amir20/dozzle/internal/container"
	"github.com/rs/zerolog/log"
)

func RPCRequest(ctx context.Context, addr string, certs tls.Certificate) error {
	client, err := agent.NewClient(addr, certs)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create agent client")
	}
	containers, err := client.ListContainers(ctx, container.ContainerLabels{})
	log.Trace().Int("containers", len(containers)).Msg("Healtcheck RPC request completed")
	return err
}
