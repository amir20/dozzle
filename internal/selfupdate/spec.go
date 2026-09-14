package selfupdate

import (
	"maps"
	"path"
	"reflect"
	"slices"
	"strings"

	dcontainer "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

const (
	// HelperLabel marks the short-lived container that performs the swap.
	HelperLabel = "dev.dozzle.self-update"

	// imageRefLabel keeps the tag a container was created from when rollback had
	// to recreate it from a bare image id (the tag by then names the broken
	// image). Without it the container reads as pinned and never updates again.
	imageRefLabel = "dev.dozzle.self-update.image"

	defaultSocket = "/var/run/docker.sock"
	swarmLabel    = "com.docker.swarm.service.name"
)

func shortID(id string) string {
	id = strings.TrimPrefix(id, "sha256:")
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

// ImageRef is the image reference cfg's container follows: Config.Image, or the
// tag remembered by a rollback that recreated it from an image id.
func ImageRef(cfg *dcontainer.Config) string {
	if cfg == nil {
		return ""
	}
	if ref := cfg.Labels[imageRefLabel]; ref != "" && imageIDRef.MatchString(cfg.Image) {
		return ref
	}
	return cfg.Image
}

func helperName(selfID string) string {
	return "dozzle-self-update-" + shortID(selfID)
}

func oldName(name, id string) string {
	return name + "-dozzle-old-" + shortID(id)
}

// helperSpec is the container Start launches: the new image running
// "dozzle self-update --target <self>", with the same way into the engine that
// Dozzle itself has.
func helperSpec(self dcontainer.InspectResponse, newImageID string) client.ContainerCreateOptions {
	var selfEnv []string
	if self.Config != nil {
		selfEnv = self.Config.Env
	}

	env := []string{}
	dockerHost, certPath := "", ""
	for _, kv := range selfEnv {
		key, value, _ := strings.Cut(kv, "=")
		switch key {
		case "DOCKER_HOST":
			dockerHost = value
			env = append(env, kv)
		case "DOCKER_CERT_PATH":
			certPath = value
			env = append(env, kv)
		case "DOCKER_TLS_VERIFY", "DOCKER_API_VERSION", "DOZZLE_LEVEL":
			env = append(env, kv)
		}
	}

	hostConfig := &dcontainer.HostConfig{
		AutoRemove:  true,
		NetworkMode: "none",
	}

	if dockerHost == "" || strings.HasPrefix(dockerHost, "unix://") {
		socket := strings.TrimPrefix(dockerHost, "unix://")
		if socket == "" {
			socket = defaultSocket
		}
		source := ""
		for _, m := range self.Mounts {
			if m.Type == mount.TypeBind && path.Clean(m.Destination) == path.Clean(socket) {
				source = m.Source
				break
			}
		}
		if source == "" {
			source, socket = defaultSocket, defaultSocket
			// DOCKER_HOST named a socket we could not find mounted; the fallback
			// bind is at the default path, so point the helper there.
			env = slices.DeleteFunc(env, func(kv string) bool { return strings.HasPrefix(kv, "DOCKER_HOST=") })
		}
		hostConfig.Binds = []string{source + ":" + socket}
	} else {
		// A TCP engine (socket proxy, remote daemon) needs a network to reach
		// it, and the certificates Dozzle has mounted. Only the bind holding
		// DOCKER_CERT_PATH is passed on, never Dozzle's data or other binds.
		hostConfig.NetworkMode = helperNetwork(self)
		for _, m := range self.Mounts {
			dest := path.Clean(m.Destination)
			if m.Type == mount.TypeBind && certPath != "" && (path.Clean(certPath) == dest || strings.HasPrefix(path.Clean(certPath), dest+"/")) {
				hostConfig.Binds = append(hostConfig.Binds, m.Source+":"+m.Destination+":ro")
			}
		}
	}

	return client.ContainerCreateOptions{
		Name: helperName(self.ID),
		Config: &dcontainer.Config{
			Image:      newImageID,
			Entrypoint: []string{"/dozzle"},
			Cmd:        []string{"self-update", "--target", self.ID},
			Env:        env,
			Labels:     map[string]string{HelperLabel: "true"},
			// The image may inherit a healthcheck meant for the server.
			Healthcheck: &dcontainer.HealthConfig{Test: []string{"NONE"}},
		},
		HostConfig: hostConfig,
	}
}

func helperNetwork(self dcontainer.InspectResponse) dcontainer.NetworkMode {
	if self.HostConfig != nil {
		mode := self.HostConfig.NetworkMode
		if mode != "" && !mode.IsContainer() && mode != "none" {
			return mode
		}
	}
	return "bridge"
}

// replacementSpec turns the inspect of the running container into the create
// request for its replacement on newImage.
//
// It differs from a verbatim replay in three ways that matter for Dozzle:
//   - every volume survives, including anonymous ones (docker run -v /data):
//     those are rewritten as mounts naming the existing volume, otherwise the
//     engine would hand the new container a fresh empty one;
//   - settings the old image supplied (env, labels, cmd, healthcheck, ...) are
//     dropped so the new image's own defaults apply;
//   - runtime state (hostname from the old id, IPs, the short-id alias) is not
//     carried over.
func replacementSpec(old dcontainer.InspectResponse, oldImage *image.InspectResponse, name string) client.ContainerCreateOptions {
	cfg := dcontainer.Config{}
	if old.Config != nil {
		cfg = *old.Config
	}
	hc := dcontainer.HostConfig{}
	if old.HostConfig != nil {
		hc = *old.HostConfig
	}
	cfg.Labels = maps.Clone(cfg.Labels)
	cfg.Image = ImageRef(&cfg)
	delete(cfg.Labels, imageRefLabel)
	cfg.Volumes = maps.Clone(cfg.Volumes)
	cfg.Env = slices.Clone(cfg.Env)
	hc.Binds = slices.Clone(hc.Binds)

	if cfg.Hostname != "" && strings.HasPrefix(old.ID, cfg.Hostname) {
		cfg.Hostname = ""
	}

	if oldImage != nil && oldImage.Config != nil {
		ic := oldImage.Config
		cfg.Env = slices.DeleteFunc(cfg.Env, func(kv string) bool { return slices.Contains(ic.Env, kv) })
		for k, v := range ic.Labels {
			if cfg.Labels[k] == v {
				delete(cfg.Labels, k)
			}
		}
		if slices.Equal(cfg.Entrypoint, ic.Entrypoint) {
			cfg.Entrypoint = nil
		}
		if slices.Equal(cfg.Cmd, ic.Cmd) {
			cfg.Cmd = nil
		}
		if cfg.WorkingDir == ic.WorkingDir {
			cfg.WorkingDir = ""
		}
		if cfg.User == ic.User {
			cfg.User = ""
		}
		if cfg.StopSignal == ic.StopSignal {
			cfg.StopSignal = ""
		}
		if cfg.Healthcheck != nil && ic.Healthcheck != nil && reflect.DeepEqual(*cfg.Healthcheck, *ic.Healthcheck) {
			cfg.Healthcheck = nil
		}
	}
	if len(cfg.Env) == 0 {
		cfg.Env = nil
	}
	if len(cfg.Labels) == 0 {
		cfg.Labels = nil
	}

	sharesNamespace := sanitizeNetworkMode(&cfg, &hc)
	hc.Mounts = preserveVolumes(old, &cfg, hc.Binds, hc.Mounts)

	var networking *network.NetworkingConfig
	if !sharesNamespace && old.NetworkSettings != nil && len(old.NetworkSettings.Networks) > 0 {
		endpoints := make(map[string]*network.EndpointSettings, len(old.NetworkSettings.Networks))
		for netName, ep := range old.NetworkSettings.Networks {
			if ep == nil {
				endpoints[netName] = &network.EndpointSettings{}
				continue
			}
			endpoints[netName] = &network.EndpointSettings{
				// Older engines list the container's short id as an alias; the
				// new container gets its own.
				Aliases:    slices.DeleteFunc(slices.Clone(ep.Aliases), func(a string) bool { return strings.HasPrefix(old.ID, a) && len(a) >= 12 }),
				IPAMConfig: ep.IPAMConfig,
				Links:      ep.Links,
				DriverOpts: ep.DriverOpts,
				GwPriority: ep.GwPriority,
			}
		}
		networking = &network.NetworkingConfig{EndpointsConfig: endpoints}
	}

	return client.ContainerCreateOptions{
		Name:             name,
		Config:           &cfg,
		HostConfig:       &hc,
		NetworkingConfig: networking,
	}
}

// preserveVolumes returns the mounts for the replacement. Named volumes and
// binds already reference their data by name or path and are left alone. An
// anonymous volume only has a destination in the create request, so it is
// replaced by a mount naming the volume the old container actually got.
func preserveVolumes(old dcontainer.InspectResponse, cfg *dcontainer.Config, binds []string, mounts []mount.Mount) []mount.Mount {
	covered := map[string]bool{}
	for _, b := range binds {
		parts := strings.Split(b, ":")
		if len(parts) >= 2 {
			covered[path.Clean(parts[1])] = true
		}
	}

	var result []mount.Mount
	for _, m := range mounts {
		if m.Type == mount.TypeVolume && m.Source == "" {
			continue
		}
		covered[path.Clean(m.Target)] = true
		result = append(result, m)
	}

	for _, mp := range old.Mounts {
		if mp.Type != mount.TypeVolume || mp.Name == "" || covered[path.Clean(mp.Destination)] {
			continue
		}
		covered[path.Clean(mp.Destination)] = true
		result = append(result, mount.Mount{
			Type:     mount.TypeVolume,
			Source:   mp.Name,
			Target:   mp.Destination,
			ReadOnly: !mp.RW,
		})
		delete(cfg.Volumes, mp.Destination)
	}
	if len(cfg.Volumes) == 0 {
		cfg.Volumes = nil
	}
	return result
}

// sanitizeNetworkMode mirrors internal/docker's sanitizeForRecreate: inspect
// reports fields that create rejects for containers sharing a namespace.
func sanitizeNetworkMode(cfg *dcontainer.Config, hc *dcontainer.HostConfig) bool {
	mode := string(hc.NetworkMode)
	isContainerMode := mode == "container" || strings.HasPrefix(mode, "container:")
	isHostMode := mode == "host"

	if isContainerMode {
		cfg.Hostname = ""
		cfg.ExposedPorts = nil
		hc.Links = nil
		hc.DNS = nil
		hc.ExtraHosts = nil
		hc.PortBindings = nil
		hc.PublishAllPorts = false
	}
	if isHostMode {
		cfg.Hostname = ""
		hc.Links = nil
	}
	if hc.UTSMode.IsHost() {
		cfg.Hostname = ""
	}
	return isContainerMode || isHostMode
}
