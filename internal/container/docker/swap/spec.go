package swap

import (
	"maps"
	"path"
	"reflect"
	"regexp"
	"slices"
	"strings"

	dcontainer "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

// ImageRefLabel keeps the tag a container was created from when rollback had
// to recreate it from a bare image id (the tag by then names the broken
// image). Without it the container reads as pinned and never updates again.
const ImageRefLabel = "dev.dozzle.self-update.image"

// imageIDRef matches a container created from a bare image id.
var imageIDRef = regexp.MustCompile(`^(sha256:)?[0-9a-f]{12,64}$`)

// IsImageID reports whether ref is a bare image id rather than a reference a
// registry could serve.
func IsImageID(ref string) bool {
	return imageIDRef.MatchString(ref)
}

// ImageRef is the image reference cfg's container follows: Config.Image, or the
// tag remembered by a rollback that recreated it from an image id.
func ImageRef(cfg *dcontainer.Config) string {
	if cfg == nil {
		return ""
	}
	if ref := cfg.Labels[ImageRefLabel]; ref != "" && IsImageID(cfg.Image) {
		return ref
	}
	return cfg.Image
}

// OldName is the name the outgoing container is renamed to while its
// replacement takes over its own.
func OldName(name, id string) string {
	return name + "-dozzle-old-" + shortID(id)
}

func shortID(id string) string {
	id = strings.TrimPrefix(id, "sha256:")
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

// ReplacementSpec turns the inspect of the running container into the create
// request for its replacement on newImage. Both self-update and the ordinary
// container Update action (internal/container/docker) recreate through it.
//
// It differs from a verbatim replay in three ways:
//   - every volume survives, including anonymous ones (docker run -v /data):
//     those are rewritten as mounts naming the existing volume, otherwise the
//     engine would hand the new container a fresh empty one;
//   - settings the old image supplied (env, labels, cmd, healthcheck, ...) are
//     dropped so the new image's own defaults apply;
//   - runtime state (hostname from the old id, dynamic IPs and MACs, the
//     short-id alias) is not carried over. A static IP is, and so is the MAC
//     beside it.
func ReplacementSpec(old dcontainer.InspectResponse, oldImage *image.InspectResponse, name string) client.ContainerCreateOptions {
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
	delete(cfg.Labels, ImageRefLabel)
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
			settings := &network.EndpointSettings{
				// Older engines list the container's short id as an alias; the
				// new container gets its own.
				Aliases:    slices.DeleteFunc(slices.Clone(ep.Aliases), func(a string) bool { return strings.HasPrefix(old.ID, a) && len(a) >= 12 }),
				IPAMConfig: ep.IPAMConfig,
				Links:      ep.Links,
				DriverOpts: ep.DriverOpts,
				GwPriority: ep.GwPriority,
			}
			// Inspect shows the MAC in use, not whether it was asked for, so it
			// is kept only beside a static IP: the same IP behind a new MAC
			// reads as an address conflict to routers and IDSes (#5652). The
			// old container is stopped before the replacement starts, so the
			// engine never sees the MAC twice.
			if hasStaticIP(ep.IPAMConfig) {
				settings.MacAddress = ep.MacAddress
			}
			endpoints[netName] = settings
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

// ForPodman adjusts a replacement's host config for a Podman engine, given
// the inspect of the container it replaces. On Docker neither applies.
//
// Podman's compat API reports a swappiness of 0 where Docker reports null for
// "never set" (containers/podman#23824), and crun refuses to start a container
// that sets it on cgroup v2 (#5662), so the 0 is dropped and the engine default
// applies. On Docker a 0 was set on purpose.
//
// A bind mounted with :Z is relabelled for the old container's private SELinux
// level, and Podman reports the bind without the :Z. The replacement would get
// a fresh level and be denied its own files (#5662), so it keeps the old one.
// The :Z is not put back: which binds had it is unknown, and relabelling a bind
// that did not would change files on the host.
func ForPodman(hc *dcontainer.HostConfig, old dcontainer.InspectResponse) {
	if hc.MemorySwappiness != nil && *hc.MemorySwappiness == 0 {
		hc.MemorySwappiness = nil
	}
	if level := selinuxLevel(old.ProcessLabel); level != "" && !setsLabelLevel(hc.SecurityOpt) {
		hc.SecurityOpt = append(slices.Clone(hc.SecurityOpt), "label=level:"+level)
	}
}

// selinuxLevel is the MLS/MCS level of an SELinux label
// (user:role:type:level, where the level itself may hold colons), or "" for
// none.
func selinuxLevel(label string) string {
	parts := strings.SplitN(label, ":", 4)
	if len(parts) < 4 {
		return ""
	}
	return parts[3]
}

// setsLabelLevel reports whether security options already fix the SELinux
// level or turn labelling off, in either the label= or the older label: form.
func setsLabelLevel(opts []string) bool {
	return slices.ContainsFunc(opts, func(o string) bool {
		for _, p := range []string{"label=level:", "label:level:", "label=disable", "label:disable"} {
			if strings.HasPrefix(o, p) {
				return true
			}
		}
		return false
	})
}

// IsPodman reports whether a server version was answered by Podman.
func IsPodman(v client.ServerVersionResult) bool {
	for _, c := range v.Components {
		if strings.Contains(strings.ToLower(c.Name), "podman") {
			return true
		}
	}
	return strings.Contains(strings.ToLower(v.Platform.Name), "podman")
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

// sanitizeNetworkMode strips what inspect reports but create rejects for a
// container sharing a namespace (a VPN sidecar, say). The rules mirror the
// daemon's own validateNetMode. Returns whether the namespace is shared, which
// also rules out attaching networks.
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

// hasStaticIP reports whether an endpoint was given a fixed IPv4 or IPv6 address.
func hasStaticIP(cfg *network.EndpointIPAMConfig) bool {
	return cfg != nil && (cfg.IPv4Address.IsValid() || cfg.IPv6Address.IsValid())
}
