package hostservice

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/stretchr/testify/assert"
)

// scalerService is a host that lists containers and answers a scale like a
// manager (err nil) or a worker (ErrNotSwarmManager). A manager given owns
// only answers for those service ids, as a manager of one swarm does.
type scalerService struct {
	containersService
	err    error
	owns   []string
	scaled []string
}

func (s *scalerService) ScaleService(_ context.Context, id string, replicas uint64) error {
	if s.owns != nil && !slices.Contains(s.owns, id) {
		return container.ErrNotSwarmManager
	}
	s.scaled = append(s.scaled, fmt.Sprintf("%s=%d", id, replicas))
	return s.err
}

func task(serviceID, name string) container.Container {
	return container.Container{ID: serviceID + "-t", Labels: map[string]string{
		container.SwarmServiceIDLabel:   serviceID,
		container.SwarmServiceNameLabel: name,
	}}
}

func webTask() container.Container { return task("svc-a", "shop_web") }

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
	assert.Equal(t, []string{"svc-a=4"}, manager.scaled)
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
		task("svc-db", "shop_db"),
	}}
	service := NewMultiHostService(managerWith(map[string]container.ClientService{"manager": manager}), time.Second)

	err := service.ScaleService(context.Background(), "shop_web", 2, nil)
	assert.True(t, errors.Is(err, ErrServiceNotFound))
	assert.Empty(t, manager.scaled)
}

// Two swarms can each run a shop_web. Only the one whose tasks the caller can
// see is scaled, however the hosts are ordered.
func TestMultiHostService_ScaleServiceStaysInTheVisibleSwarm(t *testing.T) {
	visibleSwarm := &scalerService{containers: []container.Container{task("svc-a", "shop_web")}, owns: []string{"svc-a"}}
	hiddenSwarm := &scalerService{owns: []string{"svc-b"}}
	service := NewMultiHostService(managerWith(map[string]container.ClientService{
		"a": visibleSwarm,
		"b": hiddenSwarm,
	}), time.Second)

	for range 10 { // map order varies; each run must land on svc-a only
		assert.NoError(t, service.ScaleService(context.Background(), "shop_web", 2, nil))
	}
	assert.Empty(t, hiddenSwarm.scaled)
	assert.Len(t, visibleSwarm.scaled, 10)
}

// An unreachable host answers like any other host that cannot scale, so the
// manager is still reached whatever order the hosts come in. The agent and
// docker clients map their transport failures to ErrNotSwarmManager.
func TestMultiHostService_ScaleServiceSkipsAnUnreachableHost(t *testing.T) {
	down := &scalerService{containers: []container.Container{webTask()},
		err: fmt.Errorf("agent unreachable: %w: connection refused", container.ErrNotSwarmManager)}
	manager := &scalerService{owns: []string{"svc-a"}}
	service := NewMultiHostService(managerWith(map[string]container.ClientService{"down": down, "manager": manager}), time.Second)

	for range 10 {
		assert.NoError(t, service.ScaleService(context.Background(), "shop_web", 2, nil))
	}
	assert.Len(t, manager.scaled, 10)
}
