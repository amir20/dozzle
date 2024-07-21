package cli

import (
	"embed"

	"github.com/amir20/dozzle/internal/docker"
	docker_support "github.com/amir20/dozzle/internal/support/docker"
	log "github.com/sirupsen/logrus"
)

func CreateMultiHostService(embeddedCerts embed.FS, args Args) *docker_support.MultiHostService {
	var clients []docker_support.ClientService
	if len(args.RemoteHost) > 0 {
		log.Warnf(`Remote host flag is deprecated and will be removed in future versions. Agents will replace remote hosts as a safer and performant option. See https://github.com/amir20/dozzle/issues/3066 for discussion.`)
	}

	for _, remoteHost := range args.RemoteHost {
		host, err := docker.ParseConnection(remoteHost)
		if err != nil {
			log.Fatalf("Could not parse remote host %s: %s", remoteHost, err)
		}
		log.Debugf("creating remote client for %s with %+v", host.Name, host)
		log.Infof("Creating client for %s with %s", host.Name, host.URL.String())
		if client, err := docker.NewRemoteClient(args.Filter, host); err == nil {
			if _, err := client.ListContainers(); err == nil {
				log.Debugf("connected to local Docker Engine")
				clients = append(clients, docker_support.NewDockerClientService(client))
			} else {
				log.Warnf("Could not connect to remote host %s: %s", host.ID, err)
			}
		} else {
			log.Warnf("Could not create client for %s: %s", host.ID, err)
		}
	}

	localClient, err := docker.NewLocalClient(args.Filter, args.Hostname)
	if err == nil {
		_, err := localClient.ListContainers()
		if err != nil {
			log.Debugf("could not connect to local Docker Engine: %s", err)
			go StartEvent(args, args.Mode, nil, "")
		} else {
			log.Debugf("connected to local Docker Engine")
			go StartEvent(args, args.Mode, localClient, "")
			clients = append(clients, docker_support.NewDockerClientService(localClient))
		}
	}

	certs, err := ReadCertificates(embeddedCerts)
	if err != nil {
		log.Fatalf("Could not read certificates: %v", err)
	}

	clientManager := docker_support.NewRetriableClientManager(clients, args.RemoteAgent, certs)
	return docker_support.NewMultiHostService(clientManager)
}
