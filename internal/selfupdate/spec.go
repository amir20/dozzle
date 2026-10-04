package selfupdate

import (
	"path"
	"slices"
	"strings"

	dcontainer "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/client"
)

const (
	// HelperLabel marks the short-lived container that performs the swap.
	HelperLabel = "dev.dozzle.self-update"

	defaultSocket = "/var/run/docker.sock"
)

func shortID(id string) string {
	id = strings.TrimPrefix(id, "sha256:")
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

func helperName(selfID string) string {
	return "dozzle-self-update-" + shortID(selfID)
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
