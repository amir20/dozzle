package cli

import (
	"github.com/amir20/dozzle/internal/analytics"
	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/types"
	"github.com/rs/zerolog/log"
)

func StartEvent(args Args, mode string, client container.Client, subCommand string) {
	if args.NoAnalytics {
		return
	}
	event := types.BeaconEvent{
		Name:             "start",
		Version:          args.Version(),
		Mode:             mode,
		RemoteAgents:     len(args.RemoteAgent),
		RemoteClients:    len(args.RemoteHost),
		SubCommand:       subCommand,
		HasActions:       args.EnableActions,
		HasShell:         args.EnableShell,
		HasCustomAddress: args.Addr != ":8080",
		HasCustomBase:    args.Base != "/",
		HasHostname:      args.Hostname != "",
		FilterLength:     len(args.Filter),
	}

	if client != nil {
		host := client.Host()
		event.ServerID = host.ID
		event.ServerVersion = host.DockerVersion
		event.IsSwarmMode = host.Swarm
	} else {
		event.ServerID = "n/a"
	}

	log.Trace().Interface("event", event).Msg("Sending analytics event")
	if err := analytics.SendBeacon(event); err != nil {
		log.Debug().Err(err).Msg("Failed to send analytics event")
	}
}
