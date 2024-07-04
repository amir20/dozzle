package healthcheck

import (
	"crypto/tls"

	"github.com/amir20/dozzle/internal/agent"
	log "github.com/sirupsen/logrus"
)

func RPCRequest(addr string, certs tls.Certificate) error {
	client, err := agent.NewClient(addr, certs)
	if err != nil {
		log.Fatalf("Failed to create agent client: %v", err)
	}
	containers, err := client.ListContainers()
	log.Tracef("Found %d containers.", len(containers))
	return err
}
