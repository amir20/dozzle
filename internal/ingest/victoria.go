package ingest

import "github.com/amir20/dozzle/internal/container"

type VictoriaIngestor struct {
	client container.Client
}

func NewVictoriaIngestor(client container.Client) *VictoriaIngestor {
	return &VictoriaIngestor{
		client: client,
	}
}
