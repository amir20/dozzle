package web

import (
	"context"

	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amir20/dozzle/internal/docker"
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
			cancel()
		}()
	})

	handler := createDefaultHandler(mockedClient)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
	mockedClient.AssertExpectations(t)
}

func Test_handler_streamEvents_error_request(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/events/stream", nil)
	require.NoError(t, err, "NewRequest should not return an error.")

	mockedClient := new(MockedClient)

	errChannel := make(chan error)
	mockedClient.On("Events", mock.Anything, mock.Anything).Return(errChannel)
	mockedClient.On("ListContainers").Return([]docker.Container{}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	req = req.WithContext(ctx)

	go func() {
		cancel()
	}()

	handler := createDefaultHandler(mockedClient)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
	mockedClient.AssertExpectations(t)
}
