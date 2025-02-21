package cli

import (
	"context"
	"embed"
	"os"
	"os/signal"
	"syscall"

	"github.com/amir20/dozzle/internal/docker"
	"github.com/amir20/dozzle/internal/ingest"
	docker_support "github.com/amir20/dozzle/internal/support/docker"
	"github.com/rs/zerolog/log"
)

type IngestCmd struct {
	Addr string `arg:"--ingest-addr,env:DOZZLE_INGEST_ADDR" default:"localhost:9428" help:"sets the host:port to bind for the ingest"`
}

func (c IngestCmd) Run(args Args, embeddedCerts embed.FS) error {
	client, err := docker.NewLocalClient(args.Hostname)
	service := docker_support.NewDockerClientService(client, args.Filter)
	if err != nil {
		return err
	}
	ingestor := ingest.NewVictoriaIngestor(service)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Info().Msgf("Dozzle ingestor version %s", args.Version())
		if err := ingestor.Start(ctx); err != nil {
			log.Error().Err(err).Msg("Ingestor failed")
		}
	}()
	<-ctx.Done()
	log.Info().Msg("Ingestor stopped")
	stop()

	return nil
}
