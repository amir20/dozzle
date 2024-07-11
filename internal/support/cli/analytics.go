package cli

import (
	"github.com/amir20/dozzle/internal/analytics"
	"github.com/amir20/dozzle/internal/docker"
	log "github.com/sirupsen/logrus"
)

func StartEvent(args Args, mode string, client docker.Client, subCommand string) {
	if args.NoAnalytics {
		return
	}
	event := analytics.BeaconEvent{
		Name:          "start",
		Version:       args.Version(),
		Mode:          mode,
		RemoteAgents:  len(args.RemoteAgent),
		RemoteClients: len(args.RemoteHost),
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
