package web

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	docker_support "github.com/amir20/dozzle/internal/support/docker"
	"github.com/amir20/dozzle/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// countingHostService counts Hosts() calls, which is what repairs a restarted
// agent's id via rekey.
type countingHostService struct {
	HostService
	calls atomic.Int32
}

func (c *countingHostService) Hosts() []container.Host {
	c.calls.Add(1)
	if c.HostService == nil {
		return nil
	}
	return c.HostService.Hosts()
}

func Test_reconcileHosts_throttles(t *testing.T) {
	service := &countingHostService{}
	h := &handler{hostService: service, config: &Config{}}

	h.reconcileHosts()
	require.Eventually(t, func() bool { return service.calls.Load() == 1 }, time.Second, 5*time.Millisecond)

	// a second stream connecting right away must not dial every agent again
	h.reconcileHosts()
	h.reconcileHosts()
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, int32(1), service.calls.Load())

	h.reconcileMu.Lock()
	h.reconciledAt = time.Now().Add(-hostReconcileInterval - time.Second)
	h.reconcileMu.Unlock()

	h.reconcileHosts()
	require.Eventually(t, func() bool { return service.calls.Load() == 2 }, time.Second, 5*time.Millisecond)
}

// A dashboard that was already open when an agent restarted only learns about the
// new id if something re-reads the hosts. The events stream connecting is that
// moment, so it has to reconcile rather than wait for a page load.
func Test_handler_streamEvents_reconcilesHosts(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	req, err := http.NewRequestWithContext(ctx, "GET", "/api/events/stream", nil)
	require.NoError(t, err)

	mockedClient := new(MockedClient)
	mockedClient.On("ListContainers", mock.Anything, mock.Anything).Return([]container.Container{
		{ID: "1234", Name: "test", Image: "test", Host: "localhost"},
	}, nil)
	mockedClient.On("ContainerEvents", mock.Anything, mock.AnythingOfType("chan<- container.ContainerEvent")).Return(nil).Run(func(args mock.Arguments) {
		time.Sleep(50 * time.Millisecond)
		cancel()
	})
	mockedClient.On("FindContainer", mock.Anything, "1234").Return(container.Container{
		ID:    "1234",
		Name:  "test",
		Image: "test",
		Stats: utils.NewRingBuffer[container.ContainerStat](300),
	}, nil)
	mockedClient.On("Host").Return(container.Host{ID: "localhost"})

	manager := docker_support.NewRetriableClientManager(nil, 3*time.Second, tls.Certificate{}, docker_support.NewDockerClientService(mockedClient, container.ContainerLabels{}))
	service := &countingHostService{HostService: docker_support.NewMultiHostService(manager, 3*time.Second)}

	server := CreateServer(service, nil, Config{Base: "/", Authorization: Authorization{Provider: NONE}})
	server.Handler.ServeHTTP(httptest.NewRecorder(), req)

	require.Eventually(t, func() bool { return service.calls.Load() >= 1 }, time.Second, 5*time.Millisecond)
}
