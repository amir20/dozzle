package cli

import (
	"context"
	"crypto/tls"
	"embed"
	"path/filepath"

	dozzlecerts "github.com/amir20/dozzle/internal/agentcerts"
	"github.com/amir20/dozzle/internal/config"
	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/container/docker"
	"github.com/amir20/dozzle/internal/hostservice"
	"github.com/rs/zerolog/log"
)

// CreateMultiHostService connects to every configured host. ctx bounds the
// background retry of agents and remote hosts that did not answer.
func CreateMultiHostService(ctx context.Context, embeddedCerts embed.FS, args Args) *hostservice.MultiHostService {
	var clients []container.ClientService
	if len(args.RemoteHost) > 0 {
		log.Info().Msg(`Consider using Dozzle's remote agent to manage remote hosts. See https://dozzle.dev/guide/agent for more information`)
	}

	// DOZZLE_HOST_ID names the host this process runs on, so only the local
	// client gets it. Sharing one StaticHostID with the remote engines below
	// would give them all the same id and collapse them into a single host.
	localHostIDs := container.NewHostIDResolver(args.HostID)
	remoteHostIDs := container.DerivedHostID{}

	// connectRemote builds a remote host's service, or nothing at all: the store
	// behind a service runs for the life of the process, so it is only started
	// once the engine has answered.
	connectRemote := func(ctx context.Context, host container.Host) (container.ClientService, error) {
		client, err := docker.NewRemoteClient(ctx, host, remoteHostIDs)
		if err != nil {
			return nil, err
		}
		if _, err := client.ListContainers(ctx, args.Filter); err != nil {
			_ = client.Close()
			return nil, err
		}
		return docker.NewService(client, args.Filter), nil
	}

	// Remote hosts that did not answer, retried once the manager exists.
	var downRemotes []container.Host
	for _, remoteHost := range args.RemoteHost {
		host, err := container.ParseConnection(remoteHost)
		if err != nil {
			log.Fatal().Err(err).Interface("host", remoteHost).Msg("Could not parse remote host")
		}

		log.Info().Interface("host", host).Msg("Adding remote host")
		ctx, cancel := context.WithTimeout(ctx, args.Timeout)
		service, err := connectRemote(ctx, host)
		cancel()
		if err != nil {
			log.Warn().Err(err).Interface("host", host).Msg("Could not connect to remote host, will keep retrying")
			host.Type = "remote"
			downRemotes = append(downRemotes, host)
			continue
		}
		clients = append(clients, service)
	}

	localClient, err := docker.NewLocalClient(args.Hostname, localHostIDs)
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), args.Timeout)
		defer cancel()
		_, err := localClient.ListContainers(ctx, args.Filter)
		if err != nil {
			log.Debug().Err(err).Msg("Could not connect to local Docker Engine")
		} else {
			log.Debug().Msg("Adding local Docker Engine")
			clients = append(clients, docker.NewService(localClient, args.Filter))
		}
		go StartEvent(args, "server", localClient, "")
	}

	certs, err := ReadCertificates(embeddedCerts, args.CertPath, args.KeyPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Could not read certificates")
	}

	agentCerts := map[string]tls.Certificate{}
	if len(args.PrivateAgents) > 0 {
		pair, err := dozzlecerts.LoadAgentPair(filepath.Dir(config.Path))
		var parsed tls.Certificate
		if err == nil {
			parsed, err = pair.TLS()
		}
		if err != nil {
			// They are dialed with the default pair instead, which they refuse, so
			// they show as down until the pair is restored.
			log.Error().Err(err).Msg("Could not read the private agent certificate, agents added with it will not connect")
		} else {
			for _, endpoint := range args.PrivateAgents {
				agentCerts[endpoint] = parsed
			}
		}
	}

	clientManager := hostservice.NewRetriableClientManager(args.RemoteAgent, agentCerts, args.Timeout, certs, clients...)
	for _, host := range downRemotes {
		clientManager.RetryLater(host, func(ctx context.Context) (container.ClientService, error) {
			return connectRemote(ctx, host)
		})
	}
	go clientManager.RetryFailedHosts(ctx)
	return hostservice.NewMultiHostService(clientManager, args.Timeout)
}
