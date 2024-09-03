package cli

import (
	"embed"

	"github.com/amir20/dozzle/internal/docker"
	docker_support "github.com/amir20/dozzle/internal/support/docker"
	"github.com/rs/zerolog/log"
)

func CreateMultiHostService(embeddedCerts embed.FS, args Args) (docker.Client, *docker_support.MultiHostService) {
	var clients []docker_support.ClientService
	if len(args.RemoteHost) > 0 {
		log.Info().Msg(`Consider using Dozzle's remote agent to manage remote hosts. See https://dozzle.dev/guide/agent for more information`)
	}

	for _, remoteHost := range args.RemoteHost {
		host, err := docker.ParseConnection(remoteHost)
		if err != nil {
			log.Fatal().Err(err).Interface("host", remoteHost).Msg("Could not parse remote host")
		}

		log.Info().Interface("host", host).Msg("Adding remote host")
		if client, err := docker.NewRemoteClient(args.Filter, host); err == nil {
			if _, err := client.ListContainers(); err == nil {
				clients = append(clients, docker_support.NewDockerClientService(client))
			} else {
				log.Warn().Err(err).Interface("host", host).Msg("Could not connect to remote host")
			}
		} else {
			log.Warn().Err(err).Interface("host", host).Msg("Could not create remote client")
		}
	}

	localClient, err := docker.NewLocalClient(args.Filter, args.Hostname)
	if err == nil {
		_, err := localClient.ListContainers()
		if err != nil {
			log.Debug().Err(err).Msg("Could not connect to local Docker Engine")
		} else {
			log.Debug().Msg("Adding local Docker Engine")
			clients = append(clients, docker_support.NewDockerClientService(localClient))
		}
	}

	certs, err := ReadCertificates(embeddedCerts)
	if err != nil {
		log.Fatal().Err(err).Msg("Could not read certificates")
	}

	clientManager := docker_support.NewRetriableClientManager(args.RemoteAgent, certs, clients...)
	return localClient, docker_support.NewMultiHostService(clientManager)
}
