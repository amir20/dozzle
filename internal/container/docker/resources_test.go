package docker

import (
	"context"
	"net/netip"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	docker "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/api/types/volume"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (m *mockedProxy) NetworkInspect(ctx context.Context, name string, _ client.NetworkInspectOptions) (client.NetworkInspectResult, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(client.NetworkInspectResult), args.Error(1)
}

func (m *mockedProxy) VolumeInspect(ctx context.Context, name string, _ client.VolumeInspectOptions) (client.VolumeInspectResult, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(client.VolumeInspectResult), args.Error(1)
}

// volumeProxy records the filter ContainerList is called with, which the shared
// mock ignores.
type volumeProxy struct {
	*mockedProxy
	listed client.ContainerListOptions
	items  []docker.Summary
}

func (p *volumeProxy) ContainerList(_ context.Context, options client.ContainerListOptions) (client.ContainerListResult, error) {
	p.listed = options
	return client.ContainerListResult{Items: p.items}, nil
}

func TestInspectNetwork(t *testing.T) {
	subnet := netip.MustParsePrefix("172.18.0.0/16")
	proxy := new(mockedProxy)
	proxy.On("NetworkInspect", mock.Anything, "app_default").Return(client.NetworkInspectResult{Network: network.Inspect{
		Network: network.Network{
			Name:   "app_default",
			Driver: "bridge",
			Scope:  "local",
			IPAM:   network.IPAM{Config: []network.IPAMConfig{{Subnet: subnet, Gateway: netip.MustParseAddr("172.18.0.1")}}},
		},
		Containers: map[string]network.EndpointResource{
			"bbbbbbbbbbbbbbbbbbbb": {Name: "web", IPv4Address: netip.MustParsePrefix("172.18.0.3/16")},
			"aaaaaaaaaaaaaaaaaaaa": {Name: "db", IPv4Address: netip.MustParsePrefix("172.18.0.2/16")},
		},
		Status: &network.Status{IPAM: network.IPAMStatus{Subnets: network.SubnetStatuses{subnet: {IPsInUse: 5, DynamicIPsAvailable: 65531}}}},
	}}, nil)
	c := &Client{cli: proxy}

	got, err := c.InspectNetwork(context.Background(), "app_default")

	require.NoError(t, err)
	assert.Equal(t, container.Network{
		Name:    "app_default",
		Driver:  "bridge",
		Scope:   "local",
		Subnets: []container.NetworkSubnet{{Subnet: "172.18.0.0/16", Gateway: "172.18.0.1", InUse: 5, Available: 65531}},
		Containers: []container.NetworkEndpoint{
			{ID: "aaaaaaaaaaaa", Name: "db", IP: "172.18.0.2"},
			{ID: "bbbbbbbbbbbb", Name: "web", IP: "172.18.0.3"},
		},
	}, got)
}

// An older daemon sends no IPAM status; the counts stay zero rather than inventing one.
func TestInspectNetwork_withoutStatus(t *testing.T) {
	proxy := new(mockedProxy)
	proxy.On("NetworkInspect", mock.Anything, "n").Return(client.NetworkInspectResult{Network: network.Inspect{
		Network: network.Network{Name: "n", IPAM: network.IPAM{Config: []network.IPAMConfig{{Subnet: netip.MustParsePrefix("10.0.0.0/24")}}}},
	}}, nil)

	got, err := (&Client{cli: proxy}).InspectNetwork(context.Background(), "n")

	require.NoError(t, err)
	assert.Equal(t, []container.NetworkSubnet{{Subnet: "10.0.0.0/24"}}, got.Subnets)
}

func TestInspectVolume(t *testing.T) {
	proxy := new(mockedProxy)
	proxy.On("VolumeInspect", mock.Anything, "data").Return(client.VolumeInspectResult{Volume: volume.Volume{
		Name:       "data",
		Driver:     "local",
		Scope:      "local",
		Mountpoint: "/var/lib/docker/volumes/data/_data",
		CreatedAt:  "2026-09-11T11:30:53-07:00",
		Options:    map[string]string{"type": "nfs"},
	}}, nil)
	vp := &volumeProxy{mockedProxy: proxy, items: []docker.Summary{
		{ID: "bbbbbbbbbbbbbbbbbbbb", Names: []string{"/web"}},
		{ID: "aaaaaaaaaaaaaaaaaaaa", Names: []string{"/db"}},
	}}

	got, err := (&Client{cli: vp}).InspectVolume(context.Background(), "data")

	require.NoError(t, err)
	assert.True(t, vp.listed.All, "a stopped container still holds the volume")
	assert.Contains(t, vp.listed.Filters["volume"], "data")
	assert.Equal(t, container.Volume{
		Name:       "data",
		Driver:     "local",
		Scope:      "local",
		Mountpoint: "/var/lib/docker/volumes/data/_data",
		Created:    time.Date(2026, 9, 11, 18, 30, 53, 0, time.UTC),
		Options:    map[string]string{"type": "nfs"},
		Containers: []container.VolumeUser{{ID: "aaaaaaaaaaaa", Name: "db"}, {ID: "bbbbbbbbbbbb", Name: "web"}},
	}, got)
}
