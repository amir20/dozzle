package docker

import (
	"context"
	"errors"
	"maps"
	"testing"

	"github.com/amir20/dozzle/internal/container"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/api/types/system"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (m *mockedProxy) ServiceList(ctx context.Context, options client.ServiceListOptions) (client.ServiceListResult, error) {
	args := m.Called(ctx, options)
	services, ok := args.Get(0).([]swarm.Service)
	if !ok && args.Get(0) != nil {
		panic("services is not of type []swarm.Service")
	}
	return client.ServiceListResult{Items: services}, args.Error(1)
}

func service(id string, labels map[string]string) swarm.Service {
	return swarm.Service{ID: id, Spec: swarm.ServiceSpec{Annotations: swarm.Annotations{Labels: labels}}}
}

func managerClient(proxy *mockedProxy) *DockerClient {
	return &DockerClient{
		cli:  proxy,
		host: container.Host{ID: "localhost"},
		info: system.Info{Swarm: swarm.Info{ControlAvailable: true}},
	}
}

func task(serviceID string, labels map[string]string) *container.Container {
	all := map[string]string{"com.docker.swarm.service.id": serviceID}
	maps.Copy(all, labels)
	return &container.Container{ID: "abc", Name: "svc.1", Labels: all}
}

func Test_mergeServiceLabels_merges_deploy_labels(t *testing.T) {
	proxy := new(mockedProxy)
	proxy.On("ServiceList", mock.Anything, mock.Anything).Return([]swarm.Service{
		service("svc1", map[string]string{
			"traefik.http.routers.ui.rule": "Host(`cloud.example.com`)",
			"dev.dozzle.url":               "https://cloud.example.com",
		}),
	}, nil)

	c := task("svc1", nil)
	managerClient(proxy).mergeServiceLabels(context.Background(), c)

	assert.Equal(t, "Host(`cloud.example.com`)", c.Labels["traefik.http.routers.ui.rule"])
	assert.Equal(t, "https://cloud.example.com", c.Labels["dev.dozzle.url"])
	proxy.AssertExpectations(t)
}

func Test_mergeServiceLabels_container_label_wins(t *testing.T) {
	proxy := new(mockedProxy)
	proxy.On("ServiceList", mock.Anything, mock.Anything).Return([]swarm.Service{
		service("svc1", map[string]string{"dev.dozzle.url": "https://service.example.com"}),
	}, nil)

	c := task("svc1", map[string]string{"dev.dozzle.url": "https://container.example.com"})
	managerClient(proxy).mergeServiceLabels(context.Background(), c)

	assert.Equal(t, "https://container.example.com", c.Labels["dev.dozzle.url"])
}

func Test_mergeServiceLabels_redoes_name_and_group(t *testing.T) {
	proxy := new(mockedProxy)
	proxy.On("ServiceList", mock.Anything, mock.Anything).Return([]swarm.Service{
		service("svc1", map[string]string{"dev.dozzle.name": "Cloud UI", "dev.dozzle.group": "cloud"}),
	}, nil)

	c := task("svc1", nil)
	managerClient(proxy).mergeServiceLabels(context.Background(), c)

	assert.Equal(t, "Cloud UI", c.Name)
	assert.Equal(t, "cloud", c.Group)
}

func Test_mergeServiceLabels_leaves_unmatched_containers_alone(t *testing.T) {
	proxy := new(mockedProxy)
	proxy.On("ServiceList", mock.Anything, mock.Anything).Return([]swarm.Service{
		service("other", map[string]string{"dev.dozzle.url": "https://other.example.com"}),
	}, nil)

	c := task("svc1", nil)
	managerClient(proxy).mergeServiceLabels(context.Background(), c)

	assert.NotContains(t, c.Labels, "dev.dozzle.url")
	assert.Equal(t, "svc.1", c.Name)
}

func Test_mergeServiceLabels_skips_plain_containers(t *testing.T) {
	proxy := new(mockedProxy)

	c := &container.Container{ID: "abc", Name: "plain", Labels: map[string]string{}}
	managerClient(proxy).mergeServiceLabels(context.Background(), c)

	// No swarm task in the batch, so the manager never gets asked for services.
	proxy.AssertNotCalled(t, "ServiceList", mock.Anything, mock.Anything)
}

func Test_mergeServiceLabels_skips_workers(t *testing.T) {
	proxy := new(mockedProxy)
	d := &DockerClient{cli: proxy, host: container.Host{ID: "localhost"}, info: system.Info{}}

	c := task("svc1", nil)
	d.mergeServiceLabels(context.Background(), c)

	// Listing services is manager-only; a worker must not even try.
	proxy.AssertNotCalled(t, "ServiceList", mock.Anything, mock.Anything)
	assert.Len(t, c.Labels, 1)
}

func Test_mergeServiceLabels_lists_once_for_a_batch(t *testing.T) {
	proxy := new(mockedProxy)
	proxy.On("ServiceList", mock.Anything, mock.Anything).Return([]swarm.Service{
		service("svc1", map[string]string{"dev.dozzle.group": "cloud"}),
	}, nil).Once()

	d := managerClient(proxy)
	first, second := task("svc1", nil), task("svc1", nil)
	d.mergeServiceLabels(context.Background(), first, second)
	// A second pass inside the TTL is served from the cache.
	d.mergeServiceLabels(context.Background(), task("svc1", nil))

	assert.Equal(t, "cloud", first.Group)
	assert.Equal(t, "cloud", second.Group)
	proxy.AssertExpectations(t)
}

func Test_mergeServiceLabels_survives_a_list_error(t *testing.T) {
	proxy := new(mockedProxy)
	proxy.On("ServiceList", mock.Anything, mock.Anything).Return(nil, errors.New("permission denied")).Once()

	d := managerClient(proxy)
	c := task("svc1", nil)
	require.NotPanics(t, func() { d.mergeServiceLabels(context.Background(), c) })

	assert.Len(t, c.Labels, 1)
	// The failure is cached for a window so every container does not retry.
	d.mergeServiceLabels(context.Background(), task("svc1", nil))
	proxy.AssertExpectations(t)
}
