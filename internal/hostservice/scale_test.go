package hostservice

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/stretchr/testify/assert"
)

// scalerService is a host that lists containers and answers a scale like a
// manager (err nil) or a worker (ErrNotSwarmManager).
type scalerService struct {
	containersService
	err    error
	scaled []uint64
}

func (s *scalerService) ScaleService(_ context.Context, _ string, replicas uint64) error {
	s.scaled = append(s.scaled, replicas)
	return s.err
}

func webTask() container.Container {
	return container.Container{ID: "t1", Labels: map[string]string{container.SwarmServiceNameLabel: "shop_web"}}
}

// The node serving the request is often a worker, so its refusal must hand
// the call on to a host that can do it.
func TestMultiHostService_ScaleServiceFindsAManager(t *testing.T) {
	worker := &scalerService{containers: []container.Container{webTask()}, err: container.ErrNotSwarmManager}
	manager := &scalerService{}
	service := NewMultiHostService(managerWith(map[string]container.ClientService{
		"worker":  worker,
		"manager": manager,
		"plain":   &containersService{},
	}), time.Second)

	assert.NoError(t, service.ScaleService(context.Background(), "shop_web", 4, nil))
	assert.Equal(t, []uint64{4}, manager.scaled)
}

func TestMultiHostService_ScaleServiceWithoutAManager(t *testing.T) {
	worker := &scalerService{containers: []container.Container{webTask()}, err: container.ErrNotSwarmManager}
	service := NewMultiHostService(managerWith(map[string]container.ClientService{"worker": worker}), time.Second)

	assert.ErrorIs(t, service.ScaleService(context.Background(), "shop_web", 2, nil), container.ErrNotSwarmManager)
}

// A manager's own refusal (a global service, a version conflict) is the
// answer; asking the next manager would only get it again.
func TestMultiHostService_ScaleServiceReturnsTheManagersError(t *testing.T) {
	manager := &scalerService{containers: []container.Container{webTask()}, err: container.ErrServiceNotReplicated}
	service := NewMultiHostService(managerWith(map[string]container.ClientService{"manager": manager}), time.Second)

	assert.ErrorIs(t, service.ScaleService(context.Background(), "shop_web", 2, nil), container.ErrServiceNotReplicated)
	assert.Len(t, manager.scaled, 1)
}

// A label filter that hides every task of a service also keeps it from being scaled.
func TestMultiHostService_ScaleServiceHiddenByFilter(t *testing.T) {
	manager := &scalerService{containers: []container.Container{
		{ID: "other", Labels: map[string]string{container.SwarmServiceNameLabel: "shop_db"}},
	}}
	service := NewMultiHostService(managerWith(map[string]container.ClientService{"manager": manager}), time.Second)

	err := service.ScaleService(context.Background(), "shop_web", 2, nil)
	assert.True(t, errors.Is(err, ErrServiceNotFound))
	assert.Empty(t, manager.scaled)
}
