package docker

import (
	"context"
	"errors"
	"testing"

	"github.com/amir20/dozzle/internal/container"
	cerrdefs "github.com/containerd/errdefs"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/api/types/system"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (m *mockedProxy) ServiceInspect(_ context.Context, serviceID string, _ client.ServiceInspectOptions) (client.ServiceInspectResult, error) {
	args := m.Called(serviceID)
	return client.ServiceInspectResult{Service: args.Get(0).(swarm.Service)}, args.Error(1)
}

func (m *mockedProxy) ServiceUpdate(_ context.Context, serviceID string, options client.ServiceUpdateOptions) (client.ServiceUpdateResult, error) {
	args := m.Called(serviceID, options)
	return client.ServiceUpdateResult{}, args.Error(0)
}

func replicatedService(replicas uint64) swarm.Service {
	return swarm.Service{
		ID:      "svc1",
		Version: swarm.Version{Index: 42},
		Spec: swarm.ServiceSpec{
			Annotations:  swarm.Annotations{Name: "shop_web", Labels: map[string]string{"keep": "me"}},
			Mode:         swarm.ServiceMode{Replicated: &swarm.ReplicatedService{Replicas: &replicas}},
			TaskTemplate: swarm.TaskSpec{ContainerSpec: &swarm.ContainerSpec{Image: "nginx:1"}},
		},
	}
}

// Only the replica count changes, and the inspected version goes back so a
// concurrent update makes this one fail instead of being undone by it.
func TestScaleService_changesOnlyReplicas(t *testing.T) {
	proxy := &mockedProxy{}
	proxy.On("ServiceInspect", "svc1").Return(replicatedService(2), nil)
	proxy.On("ServiceUpdate", "svc1", mock.Anything).Return(nil)

	require.NoError(t, managerClient(proxy).ScaleService(context.Background(), "svc1", 5))

	options := proxy.Calls[1].Arguments.Get(1).(client.ServiceUpdateOptions)
	assert.Equal(t, uint64(42), options.Version.Index)
	assert.Equal(t, uint64(5), *options.Spec.Mode.Replicated.Replicas)
	assert.Equal(t, "nginx:1", options.Spec.TaskTemplate.ContainerSpec.Image)
	assert.Equal(t, map[string]string{"keep": "me"}, options.Spec.Labels)
}

func TestScaleService_refusesGlobalService(t *testing.T) {
	proxy := &mockedProxy{}
	global := replicatedService(0)
	global.Spec.Mode = swarm.ServiceMode{Global: &swarm.GlobalService{}}
	proxy.On("ServiceInspect", "svc2").Return(global, nil)

	err := managerClient(proxy).ScaleService(context.Background(), "svc2", 3)
	assert.ErrorIs(t, err, container.ErrServiceNotReplicated)
	proxy.AssertNotCalled(t, "ServiceUpdate", mock.Anything, mock.Anything)
}

// A worker, a node outside the swarm (503) and another swarm's manager (404)
// all mean "ask the next host". The engine says so, not the Info read at
// startup, which a promotion or demotion since then would make wrong.
func TestScaleService_refusalsMeanTryTheNextHost(t *testing.T) {
	for name, err := range map[string]error{
		"worker":                  cerrdefs.ErrUnavailable,
		"another swarm's manager": cerrdefs.ErrNotFound,
	} {
		t.Run(name, func(t *testing.T) {
			proxy := &mockedProxy{}
			proxy.On("ServiceInspect", "svc1").Return(swarm.Service{}, err)
			// A worker's Info, as read at startup, must not decide this.
			c := &Client{cli: proxy, host: container.Host{ID: "n"}, info: system.Info{}}

			assert.ErrorIs(t, c.ScaleService(context.Background(), "svc1", 3), container.ErrNotSwarmManager)
			proxy.AssertNotCalled(t, "ServiceUpdate", mock.Anything, mock.Anything)
		})
	}
}

// Any other failure is the answer, and the caller stops asking.
func TestScaleService_otherInspectErrorIsReturned(t *testing.T) {
	proxy := &mockedProxy{}
	proxy.On("ServiceInspect", "svc1").Return(swarm.Service{}, errors.New("boom"))

	err := managerClient(proxy).ScaleService(context.Background(), "svc1", 3)
	assert.ErrorContains(t, err, "boom")
	assert.NotErrorIs(t, err, container.ErrNotSwarmManager)
}
