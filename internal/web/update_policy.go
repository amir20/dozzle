package web

import (
	"net/http"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/amir20/dozzle/internal/config"
	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/updatepolicy"
	"github.com/rs/zerolog/log"
)

// Which containers the auto-update schedule updates. Each container is auto,
// manual or off: by its dev.dozzle.update label, by a choice saved from the UI,
// or by the instance-wide mode. See updatepolicy for the rules; this file reads
// and writes the dozzle.yml half of them.

// updatePolicies is what dozzle.yml says, read once per decision batch.
type updatePolicies struct {
	mode    updatepolicy.Mode
	choices map[string]string
}

// loadUpdatePolicies reads dozzle.yml. An unreadable file is the defaults,
// which is how Dozzle behaved before any of this existed.
func loadUpdatePolicies() updatePolicies {
	file, err := config.Load(setupConfigPath)
	if err != nil {
		log.Warn().Err(err).Msg("auto update: could not read dozzle.yml, using the default for every container")
	}
	return updatePoliciesFrom(file)
}

func updatePoliciesFrom(file config.File) updatePolicies {
	p := updatePolicies{mode: updatepolicy.DefaultMode, choices: file.ContainerUpdates}
	if file.UpdateContainers != nil {
		p.mode = updatepolicy.ParseMode(*file.UpdateContainers)
	}
	return p
}

// updatePolicyKey is where a container's choice is saved: its host and the
// name the engine knows it by. An update recreates the container under a new
// id, and dev.dozzle.name can give two containers the same display name, so
// neither would do.
func updatePolicyKey(c container.Container) string {
	name := c.EngineName
	if name == "" {
		// An agent older than EngineName.
		name = c.Name
	}
	return c.Host + "/" + name
}

func (p updatePolicies) choice(c container.Container) updatepolicy.Policy {
	if choice, ok := updatepolicy.Parse(p.choices[updatePolicyKey(c)]); ok {
		return choice
	}
	return ""
}

func (p updatePolicies) decide(c container.Container) updatepolicy.Decision {
	return updatepolicy.Resolve(c.Labels, p.choice(c), p.mode)
}

// UpdatePolicy is a container's effective policy, reading dozzle.yml now. The
// cloud client asks it which containers are on the schedule.
func UpdatePolicy(c container.Container) updatepolicy.Policy {
	return loadUpdatePolicies().decide(c).Policy
}

// lastAutoUpdate is how a container's last scheduled update ended. It lives in
// memory: a restart forgets it, which only costs the line on the Updates page.
type lastAutoUpdate struct {
	Status string    `json:"status"`
	Error  string    `json:"error,omitempty"`
	At     time.Time `json:"at"`
}

var lastAutoUpdates = struct {
	sync.Mutex
	byKey map[string]lastAutoUpdate
}{byKey: make(map[string]lastAutoUpdate)}

// recordAutoUpdateResults keeps how each container of a scheduled run ended.
func recordAutoUpdateResults(job *bulkUpdateJob) {
	if job.Trigger != "schedule" || job.FinishedAt == nil {
		return
	}
	lastAutoUpdates.Lock()
	defer lastAutoUpdates.Unlock()
	for _, item := range job.Items {
		if item.service == nil {
			continue
		}
		lastAutoUpdates.byKey[updatePolicyKey(item.service.Container)] = lastAutoUpdate{Status: item.Status, Error: item.Error, At: *job.FinishedAt}
	}
}

func lastAutoUpdateFor(c container.Container) *lastAutoUpdate {
	lastAutoUpdates.Lock()
	defer lastAutoUpdates.Unlock()
	if last, ok := lastAutoUpdates.byKey[updatePolicyKey(c)]; ok {
		return &last
	}
	return nil
}

// anonymousVolume is the 64 hex name docker gives a volume nobody named.
var anonymousVolume = regexp.MustCompile(`^[0-9a-f]{64}$`)

// namedVolumes lists the named volumes c mounts: the data a newer image may
// migrate to a format the older one can no longer read. Anonymous volumes are
// left out, since recreating the container loses them anyway.
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

type containerUpdatePolicy struct {
	Host   string              `json:"host"`
	ID     string              `json:"id"`
	Name   string              `json:"name"`
	Image  string              `json:"image"`
	State  string              `json:"state"`
	Health string              `json:"health,omitempty"`
	Self   bool                `json:"self,omitempty"`
	Policy updatepolicy.Policy `json:"policy"`
	Source updatepolicy.Source `json:"source"`
	// Choice is what the UI saved, even when a label or Dozzle only
	// overrides it, so the page can say so.
	Choice  updatepolicy.Policy `json:"choice,omitempty"`
	Volumes []string            `json:"volumes,omitempty"`
	Last    *lastAutoUpdate     `json:"last,omitempty"`
}

type updatePoliciesResponse struct {
	Mode updatepolicy.Mode `json:"mode"`
	// Persisted is false when /data is not on a volume, so nothing chosen in
	// the UI would survive a recreate. Labels still work.
	Persisted bool `json:"persisted"`
	// CanChoose is whether this user may change a container's choice.
	CanChoose  bool                    `json:"canChoose"`
	Containers []containerUpdatePolicy `json:"containers"`
}

// canChooseUpdates is who may change a container's choice: anyone who may
// update containers by hand, since auto only does on a schedule what they can
// already do with a click, or anyone who may change the other settings.
func (h *handler) canChooseUpdates(r *http.Request) bool {
	return (h.config.EnableActions && h.permitActions(r)) || h.setupCanWrite(r)
}

// getUpdatePolicies lists every container the caller can see with what the
// schedule will do to it.
func (h *handler) getUpdatePolicies(w http.ResponseWriter, r *http.Request) {
	policies := loadUpdatePolicies()
	containers, errs := h.hostService.ListAllContainers(h.resolveLabels(r))
	for _, err := range errs {
		log.Debug().Err(err).Msg("update policies: host unavailable")
	}
	selfService := selfSwarmService(containers)

	resp := updatePoliciesResponse{
		Mode:       policies.mode,
		Persisted:  setupPersisted(),
		CanChoose:  h.canChooseUpdates(r),
		Containers: make([]containerUpdatePolicy, 0, len(containers)),
	}
	for _, c := range containers {
		if !container.Updatable(c) {
			continue
		}
		decision := policies.decide(c)
		self := isSelfContainer(c, selfService)
		if self && decision.Source != updatepolicy.SourceLabel {
			// Dozzle follows the schedule itself; only a label can stop it.
			decision = updatepolicy.Decision{Policy: updatepolicy.Auto, Source: updatepolicy.SourceDefault}
		}
		resp.Containers = append(resp.Containers, containerUpdatePolicy{
			Host:    c.Host,
			ID:      c.ID,
			Name:    c.Name,
			Image:   c.Image,
			State:   c.State,
			Health:  c.Health,
			Self:    self,
			Policy:  decision.Policy,
			Source:  decision.Source,
			Choice:  policies.choice(c),
			Volumes: namedVolumes(c),
			Last:    lastAutoUpdateFor(c),
		})
	}
	slices.SortFunc(resp.Containers, func(a, b containerUpdatePolicy) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	writeJSON(w, http.StatusOK, resp)
}

type setUpdatePolicyRequest struct {
	Containers []struct {
		Host string `json:"host"`
		ID   string `json:"id"`
	} `json:"containers"`
	// Policy is auto, manual or off. Empty forgets the choice, so the
	// container follows the mode again.
	Policy string `json:"policy"`
}

type setUpdatePolicyResponse struct {
	Mode updatepolicy.Mode `json:"mode"`
	// Saved counts the containers whose choice was written. A container a
	// label decides is left out, since its label would win anyway.
	Saved    int `json:"saved"`
	Labelled int `json:"labelled"`
}

// setUpdatePolicy saves the choice for one or more containers. Choosing auto
// under Dozzle only moves the instance to "Dozzle and containers I pick":
// picking a container is the clearest way anyone could say they want that, and
// it widens nothing beyond the container just picked.
func (h *handler) setUpdatePolicy(w http.ResponseWriter, r *http.Request) {
	if !h.canChooseUpdates(r) {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	if !setupPersisted() {
		agentError(w, http.StatusPreconditionFailed, "not-persisted", "data directory is not persisted")
		return
	}

	var req setUpdatePolicyRequest
	if !decodeSetupBody(w, r, &req) {
		return
	}
	policy, ok := updatepolicy.Parse(req.Policy)
	if req.Policy != "" && (!ok || string(policy) != req.Policy) {
		agentError(w, http.StatusBadRequest, "invalid", "policy must be auto, manual or off")
		return
	}
	if len(req.Containers) == 0 || len(req.Containers) > 1000 {
		agentError(w, http.StatusBadRequest, "invalid", "no containers")
		return
	}

	// Resolved against the caller's own labels, so nobody can choose for a
	// container they cannot see.
	labels := h.resolveLabels(r)
	keys := make([]string, 0, len(req.Containers))
	labelled := 0
	for _, c := range req.Containers {
		service, err := h.hostService.FindContainer(c.Host, c.ID, labels)
		if err != nil {
			continue
		}
		if _, ok := updatepolicy.FromLabels(service.Container.Labels); ok {
			labelled++
			continue
		}
		keys = append(keys, updatePolicyKey(service.Container))
	}
	if len(keys) == 0 {
		if labelled > 0 {
			agentError(w, http.StatusConflict, "set-by-label", "set by the dev.dozzle.update label")
			return
		}
		agentError(w, http.StatusNotFound, "not-found", "no containers found")
		return
	}

	var mode updatepolicy.Mode
	err := config.Update(setupConfigPath, func(f *config.File) {
		if f.ContainerUpdates == nil {
			f.ContainerUpdates = make(map[string]string)
		}
		for _, key := range keys {
			if policy == "" {
				delete(f.ContainerUpdates, key)
			} else {
				f.ContainerUpdates[key] = string(policy)
			}
		}
		if len(f.ContainerUpdates) == 0 {
			f.ContainerUpdates = nil
		}
		mode = updatePoliciesFrom(*f).mode
		if policy == updatepolicy.Auto && mode == updatepolicy.ModeDozzle {
			picked := string(updatepolicy.ModePicked)
			f.UpdateContainers = &picked
			mode = updatepolicy.ModePicked
		}
	})
	if err != nil {
		log.Error().Err(err).Msg("could not update dozzle.yml")
		http.Error(w, "could not write dozzle.yml", http.StatusInternalServerError)
		return
	}
	log.Info().Strs("containers", keys).Str("policy", string(policy)).Msg("auto update: container choice saved")
	writeJSON(w, http.StatusOK, setUpdatePolicyResponse{Mode: mode, Saved: len(keys), Labelled: labelled})
}
