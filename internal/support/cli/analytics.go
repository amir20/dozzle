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
		Name:             "start",
		Version:          args.Version(),
		Mode:             mode,
		RemoteAgents:     len(args.RemoteAgent),
		RemoteClients:    len(args.RemoteHost),
		SubCommand:       subCommand,
		HasActions:       args.EnableActions,
		HasCustomAddress: args.Addr != ":8080",
		HasCustomBase:    args.Base != "/",
		HasHostname:      args.Hostname != "",
		FilterLength:     len(args.Filter),
	}

	if client != nil {
		host := client.Host()
		event.ServerID = host.ID
		event.ServerVersion = host.DockerVersion
		event.IsSwarmMode = client.SystemInfo().Swarm.NodeID != ""
	} else {
		event.ServerID = "n/a"
	}

	log.Tracef("sending beacon event: %+v", event)
	if err := analytics.SendBeacon(event); err != nil {
		log.Debug(err)
	}
}
