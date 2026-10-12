package web

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// finderHostService answers FindContainer from a func and nothing else.
type finderHostService struct {
	HostService
	find func(host, id string) (*container.ContainerService, error)
}

func (s *finderHostService) FindContainer(host, id string, _ container.ContainerLabels) (*container.ContainerService, error) {
	return s.find(host, id)
}

func Test_resolveContainers_parallel_in_request_order(t *testing.T) {
	// Every lookup waits until all three are in flight, so a serial resolver
	// never gets past the first.
	var arrived sync.WaitGroup
	arrived.Add(3)
	allIn := make(chan struct{})
	go func() { arrived.Wait(); close(allIn) }()

	h := &handler{hostService: &finderHostService{find: func(host, id string) (*container.ContainerService, error) {
		arrived.Done()
		select {
		case <-allIn:
		case <-time.After(2 * time.Second):
			return nil, fmt.Errorf("lookups ran one at a time")
		}
		if id == "gone" {
			return nil, container.ErrContainerNotFound
		}
		return &container.ContainerService{Container: container.Container{Host: host, ID: id}}, nil
	}}}

	var found sync.Map
	services, err := h.resolveContainers([]containerRef{{"pi", "db"}, {"nas", "gone"}, {"nas", "web"}}, nil, func(s *container.ContainerService) {
		found.Store(s.Container.ID, true)
	})
	require.NoError(t, err, "a container that is gone is skipped, not an error")
	require.Len(t, services, 2)
	assert.Equal(t, "db", services[0].Container.ID)
	assert.Equal(t, "web", services[1].Container.ID)
	_, ok := found.Load("web")
	assert.True(t, ok, "found is called for each container that resolves")
	_, ok = found.Load("gone")
	assert.False(t, ok)
}

func Test_resolveContainers_returns_other_errors(t *testing.T) {
	h := &handler{hostService: &finderHostService{find: func(host, id string) (*container.ContainerService, error) {
		if host == "down" {
			return nil, fmt.Errorf("host %s not found", host)
		}
		return &container.ContainerService{Container: container.Container{Host: host, ID: id}}, nil
	}}}

	services, err := h.resolveContainers([]containerRef{{"down", "a"}, {"nas", "web"}}, nil, nil)
	require.Error(t, err)
	require.Len(t, services, 1, "what did resolve still comes back")
	assert.Equal(t, "web", services[0].Container.ID)
}
