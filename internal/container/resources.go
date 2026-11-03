package container

import "context"

// ContainerNetwork is one network a container is attached to, as the
// container's own inspect reports it. IP is empty under host networking.
type ContainerNetwork struct {
	Name string `json:"name"`
	IP   string `json:"ip,omitempty"`
}

// Network is `docker network inspect`, trimmed to what the inspect drawer shows.
type Network struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Driver     string            `json:"driver"`
	Scope      string            `json:"scope"`
	Internal   bool              `json:"internal"`
	Subnets    []NetworkSubnet   `json:"subnets"`
	Containers []NetworkEndpoint `json:"containers"`
}

// NetworkSubnet is one IPAM pool. InUse and Available stay zero on a daemon too
// old to report IPAM status, so a zero total means "unknown", not "empty".
type NetworkSubnet struct {
	Subnet    string `json:"subnet"`
	Gateway   string `json:"gateway,omitempty"`
	InUse     uint64 `json:"inUse"`
	Available uint64 `json:"available"`
}

// NetworkEndpoint is a container attached to a network. ID is the short ID the
// rest of Dozzle uses.
type NetworkEndpoint struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	IP   string `json:"ip,omitempty"`
}

// Volume is `docker volume inspect` plus every container that mounts it.
type Volume struct {
	Name       string            `json:"name"`
	Driver     string            `json:"driver"`
	Scope      string            `json:"scope"`
	Options    map[string]string `json:"options,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
	Containers []VolumeUser      `json:"containers"`
}

// VolumeUser is a container that mounts a volume, running or not.
type VolumeUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ResourceInspector is implemented by clients that can inspect the networks and
// volumes a container uses: docker, and an agent in front of one. k8s has
// neither, and its client does not implement it.
type ResourceInspector interface {
	InspectNetwork(ctx context.Context, name string) (Network, error)
	InspectVolume(ctx context.Context, name string) (Volume, error)
}
