package cli

import (
	"github.com/amir20/dozzle/internal/analytics"
	"github.com/amir20/dozzle/internal/docker"
	log "github.com/sirupsen/logrus"
)

func StartEvent(version string, mode string, agents []string, remoteClients []string, client docker.Client, subCommand string) {
	event := analytics.BeaconEvent{
		Name:          "start",
		Version:       version,
		Mode:          mode,
		RemoteAgents:  len(agents),
		RemoteClients: len(remoteClients),
		SubCommand:    subCommand,
	}

	if client != nil {
		event.ServerID = client.SystemInfo().ID
		event.ServerVersion = client.SystemInfo().ServerVersion
	} else {
		event.ServerID = "n/a"
	}

	if err := analytics.SendBeacon(event); err != nil {
		log.Debug(err)
	}
}
