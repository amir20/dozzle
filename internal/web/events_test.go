package web

import (
	"context"
	"crypto/tls"
	"time"

	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amir20/dozzle/internal/container"
	docker_support "github.com/amir20/dozzle/internal/support/docker"
	"github.com/amir20/dozzle/internal/utils"
	"github.com/beme/abide"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_handler_streamEvents_happy(t *testing.T) {
	context, cancel := context.WithCancel(context.Background())
	req, err := http.NewRequestWithContext(context, "GET", "/api/events/stream", nil)
	require.NoError(t, err, "NewRequest should not return an error.")

	mockedClient := new(MockedClient)

	mockedClient.On("ListContainers", mock.Anything, mock.Anything).Return([]container.Container{
		{ID: "1234", Name: "test", Image: "test", Host: "localhost"},
	}, nil)
	mockedClient.On("ContainerEvents", mock.Anything, mock.AnythingOfType("chan<- container.ContainerEvent")).Return(nil).Run(func(args mock.Arguments) {
		messages := args.Get(1).(chan<- container.ContainerEvent)

		time.Sleep(50 * time.Millisecond)
		messages <- container.ContainerEvent{
			Name:    "start",
			ActorID: "1234",
			Host:    "localhost",
		}
		messages <- container.ContainerEvent{
			Name:    "something-random",
			ActorID: "1234",
			Host:    "localhost",
		}
		time.Sleep(50 * time.Millisecond)
		cancel()
	})
	mockedClient.On("FindContainer", mock.Anything, "1234").Return(container.Container{
		ID:    "1234",
		Name:  "test",
		Image: "test",
		Stats: utils.NewRingBuffer[container.ContainerStat](300), // 300 seconds of stats
	}, nil)

	mockedClient.On("Host").Return(container.Host{
		ID: "localhost",
	})

	// This is needed so that the server is initialized for store
	manager := docker_support.NewRetriableClientManager(nil, 3*time.Second, tls.Certificate{}, docker_support.NewDockerClientService(mockedClient, container.ContainerLabels{}))
	multiHostService := docker_support.NewMultiHostService(manager, 3*time.Second)

	server := CreateServer(multiHostService, nil, Config{Base: "/", Authorization: Authorization{Provider: NONE}})

	handler := server.Handler
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
	mockedClient.AssertExpectations(t)
}

// Test_handler_streamEvents_filtered asserts that container-event for a container
// outside the caller's label scope is not forwarded. The mocked ListContainers
// only ever returns the in-scope container ("visible"), so the out-of-scope
// container ("secret") is never added to the visible set and its lifecycle event
// must be dropped. See GHSA-xcw9-qmmf-vqxj.
func Test_handler_streamEvents_filtered(t *testing.T) {
	context, cancel := context.WithCancel(context.Background())
	req, err := http.NewRequestWithContext(context, "GET", "/api/events/stream", nil)
	require.NoError(t, err, "NewRequest should not return an error.")

	mockedClient := new(MockedClient)

	mockedClient.On("ListContainers", mock.Anything, mock.Anything).Return([]container.Container{
		{ID: "visible", Name: "visible", Image: "test", Host: "localhost"},
	}, nil)
	mockedClient.On("ContainerEvents", mock.Anything, mock.AnythingOfType("chan<- container.ContainerEvent")).Return(nil).Run(func(args mock.Arguments) {
		messages := args.Get(1).(chan<- container.ContainerEvent)

		time.Sleep(50 * time.Millisecond)
		// out-of-scope container: must be dropped
		messages <- container.ContainerEvent{
			Name:    "start",
			ActorID: "secret",
			Host:    "localhost",
		}
		// in-scope container: must be forwarded
		messages <- container.ContainerEvent{
			Name:    "start",
			ActorID: "visible",
			Host:    "localhost",
		}
		time.Sleep(50 * time.Millisecond)
		cancel()
	})

	mockedClient.On("FindContainer", mock.Anything, mock.Anything).Return(container.Container{
		ID:    "visible",
		Name:  "visible",
		Image: "test",
		Stats: utils.NewRingBuffer[container.ContainerStat](300),
	}, nil)

	mockedClient.On("Host").Return(container.Host{
		ID: "localhost",
	})

	manager := docker_support.NewRetriableClientManager(nil, 3*time.Second, tls.Certificate{}, docker_support.NewDockerClientService(mockedClient, container.ContainerLabels{}))
	multiHostService := docker_support.NewMultiHostService(manager, 3*time.Second)

	server := CreateServer(multiHostService, nil, Config{Base: "/", Authorization: Authorization{Provider: NONE}})

	handler := server.Handler
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
	mockedClient.AssertExpectations(t)
}
