package cli

import (
	"github.com/amir20/dozzle/internal/analytics"
	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/types"
	"github.com/rs/zerolog/log"
)

// BeaconBase is what every beacon from this process repeats: how it was
// started, not what it is doing.
func BeaconBase(args Args, mode string) types.BeaconEvent {
	return types.BeaconEvent{
		Version:          args.Version(),
		Mode:             mode,
		RemoteAgents:     len(args.RemoteAgent),
		RemoteClients:    len(args.RemoteHost),
		FileAgents:       len(args.FileAgents),
		HasActions:       args.EnableActions,
		HasShell:         args.EnableShell,
		HasCustomAddress: args.Addr != ":8080",
		HasCustomBase:    args.Base != "/",
		HasHostname:      args.Hostname != "",
		FilterLength:     len(args.Filter),
	}
}

func StartEvent(args Args, mode string, client container.Client, subCommand string) {
	if args.NoAnalytics {
		return
	}
	event := BeaconBase(args, mode)
	event.Name = "start"
	event.SubCommand = subCommand

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
