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

	mockedClient.On("ListContainers", mock.Anything, mock.Anything).Return([]container.Container{}, nil)
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
