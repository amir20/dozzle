package cli

import (
	"context"
	"embed"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/amir20/dozzle/internal/agent"
	"github.com/amir20/dozzle/internal/docker"
	"github.com/amir20/dozzle/internal/notification"
	container_support "github.com/amir20/dozzle/internal/support/container"
	docker_support "github.com/amir20/dozzle/internal/support/docker"
	"github.com/rs/zerolog/log"
)

type AgentCmd struct {
	Addr string `arg:"--agent-addr,env:DOZZLE_AGENT_ADDR" default:":7007" help:"sets the host:port to bind for the agent"`
}

func (a *AgentCmd) Run(args Args, embeddedCerts embed.FS) error {
	if args.Mode != "server" {
		return fmt.Errorf("agent command is only available in server mode")
	}
	client, err := docker.NewLocalClient(args.Hostname)
	if err != nil {
		return fmt.Errorf("failed to create docker client: %w", err)
	}
	certs, err := ReadCertificates(embeddedCerts, args.CertPath, args.KeyPath)
	if err != nil {
		return fmt.Errorf("failed to read certificates: %w", err)
	}

	listener, err := net.Listen("tcp", args.Agent.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	tempFile, err := os.CreateTemp("", "agent-*.addr")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	io.WriteString(tempFile, listener.Addr().String())
	log.Debug().Str("file", tempFile.Name()).Msg("Created temp file")
	go StartEvent(args, "", client, "agent")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Create shared client service (single ContainerStore for both agent server and notifications)
	clientService := docker_support.NewDockerClientService(client, args.Filter)

	// Create notification manager using the shared client service
	clients := []container_support.ClientService{clientService}
	notifListener := notification.NewContainerLogListener(ctx, clients)
	notificationManager := notification.NewManager(notifListener)
	if err := notificationManager.Start(); err != nil {
		return fmt.Errorf("failed to start notification manager: %w", err)
	}

	// Create agent server using the same shared client service
	server, err := agent.NewServer(clientService, certs, args.Version(), notificationManager)
	if err != nil {
		return fmt.Errorf("failed to create agent server: %w", err)
	}
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
	return nil
}
