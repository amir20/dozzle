package selfupdate

import (
	"context"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/amir20/dozzle/internal/profile"
	"github.com/moby/moby/client"
	"github.com/rs/zerolog/log"
)

// SelfID is the id of the container Dozzle runs in, or "" when it cannot be
// told apart. It starts from profile.SelfContainerID, which reads the id off
// the /etc/hostname mount, and checks it with the engine: a container joined to
// another's network namespace (network_mode: service:x) gets those files from
// that other container, so the hint names the wrong one. #5289
var SelfID = sync.OnceValue(func() string {
	hint := profile.SelfContainerID()
	if hint == "" {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cli, err := newClient(ctx)
	if err != nil {
		return hint
	}
	defer cli.Close()
	return resolveSelf(ctx, cli, hint, os.TempDir())
})

// resolveSelf returns hint unless other containers share its network
// namespace, in which case Dozzle may be any of them. It then drops a marker
// file in markerDir and keeps the one candidate whose filesystem has it. Any
// doubt returns "", which callers already treat as "could be anything".
func resolveSelf(ctx context.Context, cli dockerAPI, hint string, markerDir string) string {
	inspect, err := cli.ContainerInspect(ctx, hint, client.ContainerInspectOptions{})
	if err != nil {
		return hint
	}
	list, err := cli.ContainerList(ctx, client.ContainerListOptions{All: true})
	if err != nil {
		return hint
	}
	candidates := []string{hint}
	for _, c := range list.Items {
		if c.ID != hint && JoinsNetworkOf(c.HostConfig.NetworkMode, hint, trimName(inspect.Container.Name)) {
			candidates = append(candidates, c.ID)
		}
	}
	if len(candidates) == 1 {
		return hint
	}

	marker, err := os.CreateTemp(markerDir, ".dozzle-self-*")
	if err != nil {
		log.Warn().Err(err).Msg("unable to write a marker to find Dozzle's own container, self-update is disabled")
		return ""
	}
	marker.Close()
	defer os.Remove(marker.Name())

	found := ""
	for _, id := range candidates {
		if _, err := cli.ContainerStatPath(ctx, id, client.ContainerStatPathOptions{Path: marker.Name()}); err == nil {
			if found != "" {
				return ""
			}
			found = id
		}
	}
	if found == "" {
		log.Warn().Msg("unable to find Dozzle's own container among those sharing its network, self-update is disabled")
	}
	return found
}

// JoinsNetworkOf reports whether a container with this network mode shares the
// namespace of the container id named name. The engine keeps the reference as
// it was given: compose writes the full id, docker run whatever was typed.
func JoinsNetworkOf(mode string, id string, name string) bool {
	ref, ok := strings.CutPrefix(mode, "container:")
	if !ok || ref == "" {
		return false
	}
	return ref == name || (len(ref) >= 12 && strings.HasPrefix(id, ref))
}
