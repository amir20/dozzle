package cli

import (
	"context"
	"embed"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/docker"
	container_support "github.com/amir20/dozzle/internal/support/container"
	docker_support "github.com/amir20/dozzle/internal/support/docker"
	"github.com/rs/zerolog/log"
)

func CreateMultiHostService(embeddedCerts embed.FS, args Args) *docker_support.MultiHostService {
	var clients []container_support.ClientService
	if len(args.RemoteHost) > 0 {
		log.Info().Msg(`Consider using Dozzle's remote agent to manage remote hosts. See https://dozzle.dev/guide/agent for more information`)
	}

	for _, remoteHost := range args.RemoteHost {
		host, err := container.ParseConnection(remoteHost)
		if err != nil {
			log.Fatal().Err(err).Interface("host", remoteHost).Msg("Could not parse remote host")
		}

		log.Info().Interface("host", host).Msg("Adding remote host")
		if client, err := docker.NewRemoteClient(host); err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), args.Timeout)
			defer cancel()
			if _, err := client.ListContainers(ctx, args.Filter); err == nil {
				clients = append(clients, docker_support.NewDockerClientService(client, args.Filter))
			} else {
				log.Warn().Err(err).Interface("host", host).Msg("Could not connect to remote host")
			}
		} else {
			log.Warn().Err(err).Interface("host", host).Msg("Could not create remote client")
		}
	}

	localClient, err := docker.NewLocalClient(args.Hostname)
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), args.Timeout)
		defer cancel()
		_, err := localClient.ListContainers(ctx, args.Filter)
		if err != nil {
			log.Debug().Err(err).Msg("Could not connect to local Docker Engine")
		} else {
			log.Debug().Msg("Adding local Docker Engine")
			clients = append(clients, docker_support.NewDockerClientService(localClient, args.Filter))
		}
		go StartEvent(args, "server", localClient, "")
	}

	certs, err := ReadCertificates(embeddedCerts)
	if err != nil {
		log.Fatal().Err(err).Msg("Could not read certificates")
	}

	clientManager := docker_support.NewRetriableClientManager(args.RemoteAgent, args.Timeout, certs, clients...)
	return docker_support.NewMultiHostService(clientManager, args.Timeout)
}
