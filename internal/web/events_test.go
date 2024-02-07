package web

import (
	"context"
	"time"

	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amir20/dozzle/internal/docker"
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
	errChannel := make(chan error)

	mockedClient.On("ListContainers").Return([]docker.Container{}, nil)
	mockedClient.On("Events", mock.Anything, mock.AnythingOfType("chan<- docker.ContainerEvent")).Return(errChannel).Run(func(args mock.Arguments) {
		messages := args.Get(1).(chan<- docker.ContainerEvent)
		go func() {
			time.Sleep(50 * time.Millisecond)
			messages <- docker.ContainerEvent{
				Name:    "start",
				ActorID: "1234",
				Host:    "localhost",
			}
			messages <- docker.ContainerEvent{
				Name:    "something-random",
				ActorID: "1234",
				Host:    "localhost",
			}
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()
	})
	mockedClient.On("FindContainer", "1234").Return(docker.Container{
		ID:    "1234",
		Name:  "test",
		Image: "test",
		Stats: utils.NewRingBuffer[docker.ContainerStat](300), // 300 seconds of stats
	}, nil)

	clients := map[string]docker.Client{
		"localhost": mockedClient,
	}

	// This is needed so that the server is initialized for store
	server := CreateServer(clients, nil, Config{Base: "/", Authorization: Authorization{Provider: NONE}})

	handler := server.Handler
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
	mockedClient.AssertExpectations(t)
}
