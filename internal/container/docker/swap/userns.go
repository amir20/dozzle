package swap

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"time"

	dcontainer "github.com/moby/moby/api/types/container"
	"github.com/rs/zerolog/log"
)

// WithPodmanKeepID returns old with --userns keep-id put back in its host
// config. Podman's compat inspect reports keep-id as "private" and leaves the ID
// maps out, and recreating with "private" fails with "must provide at least one
// UID or GID mapping" (#5662). Only the libpod inspect carries the maps, so it
// is read over cli's own connection. Anything else, or a libpod API that cannot
// be reached, returns old unchanged and the swap fails and rolls back as before.
func WithPodmanKeepID(ctx context.Context, cli any, old dcontainer.InspectResponse) dcontainer.InspectResponse {
	if old.HostConfig == nil || old.HostConfig.UsernsMode != "private" {
		return old
	}
	uidMap, gidMap := libpodIDMaps(ctx, cli, old.ID)
	mode := KeepIDMode(uidMap, gidMap)
	if mode == "" {
		return old
	}
	hc := *old.HostConfig
	hc.UsernsMode = dcontainer.UsernsMode(mode)
	old.HostConfig = &hc
	return old
}

// KeepIDMode rebuilds keep-id from a container's ID maps as the libpod inspect
// reports them ("container:host:size"). keep-id is the one mapping that puts a
// container id on intermediate id 0, the rootless user itself, so that entry
// gives back the uid and gid it was created with. Any other map returns "".
func KeepIDMode(uidMap, gidMap []string) string {
	uid, gid := keptID(uidMap), keptID(gidMap)
	if uid == "" || gid == "" {
		return ""
	}
	return "keep-id:uid=" + uid + ",gid=" + gid
}

func keptID(maps []string) string {
	for _, m := range maps {
		parts := strings.Split(m, ":")
		if len(parts) == 3 && parts[1] == "0" && parts[2] == "1" && parts[0] != "0" {
			return parts[0]
		}
	}
	return ""
}

func libpodIDMaps(ctx context.Context, cli any, id string) (uidMap, gidMap []string) {
	var inspect struct {
		HostConfig struct {
			IDMappings *struct {
				UIDMap []string `json:"UidMap"`
				GIDMap []string `json:"GidMap"`
			}
		}
	}
	if !libpodGet(ctx, cli, "/libpod/containers/"+id+"/json", &inspect) || inspect.HostConfig.IDMappings == nil {
		return nil, nil
	}
	return inspect.HostConfig.IDMappings.UIDMap, inspect.HostConfig.IDMappings.GIDMap
}

// libpodGet decodes the libpod API's answer to path into out, over cli's own
// connection. It returns false when cli has no dialer or the call fails.
func libpodGet(ctx context.Context, cli any, path string, out any) bool {
	dialer, ok := cli.(interface {
		Dialer() func(context.Context) (net.Conn, error)
	})
	if !ok {
		return false
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	httpClient := &http.Client{Transport: &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) { return dialer.Dialer()(ctx) },
	}}
	defer httpClient.CloseIdleConnections()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://podman/v4.0.0"+path, nil)
	if err != nil {
		return false
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Debug().Err(err).Str("path", path).Msg("could not read the podman libpod api")
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}
	return json.NewDecoder(resp.Body).Decode(out) == nil
}
