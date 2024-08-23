package cli

import (
	"context"
	"embed"
	"io"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/amir20/dozzle/internal/agent"
	"github.com/amir20/dozzle/internal/docker"
	"github.com/rs/zerolog/log"
)

func StartAgent(args Args, embedCerts embed.FS) {
	client, err := docker.NewLocalClient(args.Filter, args.Hostname)
	if err != nil {
		log.Fatal().Err(err).Msg("Could not create docker client")
	}
	certs, err := ReadCertificates(embedCerts)
	if err != nil {
		log.Fatal().Err(err).Msg("Could not read certificates")
	}

	listener, err := net.Listen("tcp", ":7007")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to listen")
	}
	tempFile, err := os.CreateTemp("./", "agent-*.addr")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create temp file")
	}
	io.WriteString(tempFile, listener.Addr().String())
	go StartEvent(args, "", client, "agent")
	server, err := agent.NewServer(client, certs, args.Version())
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create agent server")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		log.Info().Msgf("Dozzle agent version %s", args.Version())
		log.Info().Msgf("Agent listening on %s", listener.Addr().String())

		if err := server.Serve(listener); err != nil {
			log.Error().Err(err).Msg("failed to serve")
		}
	}()
	<-ctx.Done()
	stop()
	log.Info().Msg("Shutting down agent")
	server.Stop()
	log.Debug().Str("file", tempFile.Name()).Msg("Removing temp file")
	os.Remove(tempFile.Name())
}
