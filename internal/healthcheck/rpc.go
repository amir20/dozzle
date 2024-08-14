package healthcheck

import (
	"crypto/tls"

	"github.com/amir20/dozzle/internal/agent"
	"github.com/rs/zerolog/log"
)

func RPCRequest(addr string, certs tls.Certificate) error {
	client, err := agent.NewClient(addr, certs)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create agent client")
	}
	containers, err := client.ListContainers()
	log.Trace().Int("containers", len(containers)).Msg("Healtcheck RPC request completed")
	return err
}
