package docker

import (
	"context"
	"testing"

	"github.com/amir20/dozzle/internal/container"
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
	proxy.On("ServiceInspect", "shop_web").Return(replicatedService(2), nil)
	proxy.On("ServiceUpdate", "svc1", mock.Anything).Return(nil)

	require.NoError(t, managerClient(proxy).ScaleService(context.Background(), "shop_web", 5))

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
	proxy.On("ServiceInspect", "shop_metrics").Return(global, nil)

	err := managerClient(proxy).ScaleService(context.Background(), "shop_metrics", 3)
	assert.ErrorIs(t, err, container.ErrServiceNotReplicated)
	proxy.AssertNotCalled(t, "ServiceUpdate", mock.Anything, mock.Anything)
}

// A worker's engine would refuse anyway; the client says so without asking it.
func TestScaleService_workerIsNotAManager(t *testing.T) {
	proxy := &mockedProxy{}
	worker := &Client{cli: proxy, host: container.Host{ID: "worker"}, info: system.Info{Swarm: swarm.Info{ControlAvailable: false}}}

	assert.ErrorIs(t, worker.ScaleService(context.Background(), "shop_web", 3), container.ErrNotSwarmManager)
	proxy.AssertNotCalled(t, "ServiceInspect", mock.Anything)
}
