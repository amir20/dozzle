package docker

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"github.com/amir20/dozzle/internal/container"
	"github.com/moby/moby/client"
)

// InspectNetwork reads one network and the containers attached to it.
func (d *Client) InspectNetwork(ctx context.Context, name string) (container.Network, error) {
	result, err := d.cli.NetworkInspect(ctx, name, client.NetworkInspectOptions{})
	if err != nil {
		return container.Network{}, err
	}
	n := result.Network

	network := container.Network{
		ID:         n.ID,
		Name:       n.Name,
		Driver:     n.Driver,
		Scope:      n.Scope,
		Internal:   n.Internal,
		Subnets:    make([]container.NetworkSubnet, 0, len(n.IPAM.Config)),
		Containers: make([]container.NetworkEndpoint, 0, len(n.Containers)),
	}
	for _, cfg := range n.IPAM.Config {
		if !cfg.Subnet.IsValid() {
			continue
		}
		subnet := container.NetworkSubnet{Subnet: cfg.Subnet.String()}
		if cfg.Gateway.IsValid() {
			subnet.Gateway = cfg.Gateway.String()
		}
		if n.Status != nil {
			if status, ok := n.Status.IPAM.Subnets[cfg.Subnet]; ok {
				subnet.InUse = status.IPsInUse
				subnet.Available = status.DynamicIPsAvailable
			}
		}
		network.Subnets = append(network.Subnets, subnet)
	}
	for id, endpoint := range n.Containers {
		e := container.NetworkEndpoint{ID: shortID(id), Name: endpoint.Name}
		if endpoint.IPv4Address.IsValid() {
			e.IP = endpoint.IPv4Address.Addr().String()
		} else if endpoint.IPv6Address.IsValid() {
			e.IP = endpoint.IPv6Address.Addr().String()
		}
		network.Containers = append(network.Containers, e)
	}
	slices.SortFunc(network.Containers, func(a, b container.NetworkEndpoint) int { return cmp.Compare(a.Name, b.Name) })
	return network, nil
}

// InspectVolume reads one volume and lists every container that mounts it,
// stopped ones included, since a stopped container still holds the volume.
func (d *Client) InspectVolume(ctx context.Context, name string) (container.Volume, error) {
	result, err := d.cli.VolumeInspect(ctx, name, client.VolumeInspectOptions{})
	if err != nil {
		return container.Volume{}, err
	}
	v := result.Volume

	list, err := d.cli.ContainerList(ctx, client.ContainerListOptions{All: true, Filters: make(client.Filters).Add("volume", v.Name)})
	if err != nil {
		return container.Volume{}, err
	}

	volume := container.Volume{
		Name:       v.Name,
		Driver:     v.Driver,
		Scope:      v.Scope,
		Options:    v.Options,
		Labels:     v.Labels,
		Containers: make([]container.VolumeUser, 0, len(list.Items)),
	}
	for _, c := range list.Items {
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}
		volume.Containers = append(volume.Containers, container.VolumeUser{ID: shortID(c.ID), Name: name})
	}
	slices.SortFunc(volume.Containers, func(a, b container.VolumeUser) int { return cmp.Compare(a.Name, b.Name) })
	return volume, nil
}

func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}
