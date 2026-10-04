package web

import (
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/amir20/dozzle/internal/config"
	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/updatepolicy"
	"github.com/rs/zerolog/log"
)

// Which containers the auto-update schedule updates: the instance-wide mode in
// dozzle.yml, and the dev.dozzle.update label on each container. See
// updatepolicy for the rules; this file reads the dozzle.yml half and lists
// what the schedule will do, for Settings → Updates.

// loadUpdateMode reads the mode from dozzle.yml, with the flag or env var
// winning. An unreadable file is the default, which is how Dozzle behaved
// before the mode existed.
func loadUpdateMode(setup SetupConfig) updatepolicy.Mode {
	if setup.UpdateContainers != nil {
		return updatepolicy.ParseMode(*setup.UpdateContainers)
	}
	file, err := config.Load(setupConfigPath)
	if err != nil {
		log.Warn().Err(err).Msg("auto update: could not read dozzle.yml, using the default mode")
	}
	return updateModeFrom(setup, file)
}

func updateModeFrom(setup SetupConfig, file config.File) updatepolicy.Mode {
	if setup.UpdateContainers != nil {
		return updatepolicy.ParseMode(*setup.UpdateContainers)
	}
	if file.UpdateContainers == nil {
		return updatepolicy.DefaultMode
	}
	return updatepolicy.ParseMode(*file.UpdateContainers)
}

// anonymousVolume is the 64 hex name docker gives a volume nobody named.
var anonymousVolume = regexp.MustCompile(`^[0-9a-f]{64}$`)

// namedVolumes lists the named volumes c mounts: the data a newer image may
// migrate to a format the older one can no longer read. Anonymous volumes are
// left out: nobody chose to keep data in them.
func namedVolumes(c container.Container) []string {
	var names []string
	for _, m := range c.Mounts {
		if m.Type != "volume" {
			continue
		}
		name := ""
		// /var/lib/docker/volumes/<name>/_data
		parts := strings.Split(strings.TrimSuffix(m.Source, "/"), "/")
		for i := 0; i+1 < len(parts); i++ {
			if parts[i] == "volumes" {
				name = parts[i+1]
			}
		}
		if name == "" || anonymousVolume.MatchString(name) {
			continue
		}
		names = append(names, name)
	}
	return names
}

// containerUpdatePolicy is one container as Settings → Updates lists it. The
// page works out what each mode would do from Label, so picking a mode shows
// its effect before it is saved.
type containerUpdatePolicy struct {
	Host   string `json:"host"`
	ID     string `json:"id"`
	Name   string `json:"name"`
	Image  string `json:"image"`
	State  string `json:"state"`
	Health string `json:"health,omitempty"`
	// Label is what the container's labels choose: auto, manual, off, or
	// empty when they choose nothing.
	Label updatepolicy.Policy `json:"label,omitempty"`
	// Volumes are its named volumes, for the warning that comes with All.
	Volumes []string `json:"volumes,omitempty"`
}

type updatePoliciesResponse struct {
	Mode       updatepolicy.Mode       `json:"mode"`
	Containers []containerUpdatePolicy `json:"containers"`
}

// getUpdatePolicies lists every container the caller can see that Dozzle could
// update, with what its labels say. Dozzle's own container is left out: it
// follows the schedule by itself, and the page says so separately.
func (h *handler) getUpdatePolicies(w http.ResponseWriter, r *http.Request) {
	containers, errs := h.hostService.ListAllContainers(h.resolveLabels(r))
	for _, err := range errs {
		log.Debug().Err(err).Msg("update policies: host unavailable")
	}
	selfService := selfSwarmService(containers)

	resp := updatePoliciesResponse{
		Mode:       loadUpdateMode(h.config.Setup),
		Containers: make([]containerUpdatePolicy, 0, len(containers)),
	}
	for _, c := range containers {
		if !container.Updatable(c) || isSelfContainer(c, selfService) {
			continue
		}
		label, _ := updatepolicy.FromLabels(c.Labels)
		resp.Containers = append(resp.Containers, containerUpdatePolicy{
			Host:    c.Host,
			ID:      c.ID,
			Name:    c.Name,
			Image:   c.Image,
			State:   c.State,
			Health:  c.Health,
			Label:   label,
			Volumes: namedVolumes(c),
		})
	}
	slices.SortFunc(resp.Containers, func(a, b containerUpdatePolicy) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	writeJSON(w, http.StatusOK, resp)
}
