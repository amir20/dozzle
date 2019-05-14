package main

import (
	"context"
	"errors"
	"github.com/magiconair/properties/assert"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/amir20/dozzle/docker"
	"github.com/beme/abide"
	"github.com/docker/docker/api/types/events"
	"github.com/gobuffalo/packr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockedClient struct {
	mock.Mock
	docker.Client
}

func (m *MockedClient) ListContainers() ([]docker.Container, error) {
	args := m.Called()
	containers, ok := args.Get(0).([]docker.Container)
	if !ok {
		panic("containers is not of type []docker.Container")
	}
	return containers, args.Error(1)
}

func (m *MockedClient) ContainerLogs(ctx context.Context, id string, tailSize int) (<-chan string, <-chan error) {
	args := m.Called(ctx, id, tailSize)
	channel, ok := args.Get(0).(chan string)
	if !ok {
		panic("channel is not of type chan string")
	}

	err, ok := args.Get(1).(chan error)
	if !ok {
		panic("error is not of type chan error")
	}
	return channel, err
}

func (m *MockedClient) Events(ctx context.Context) (<-chan events.Message, <-chan error) {
	args := m.Called(ctx)
	channel, ok := args.Get(0).(chan events.Message)
	if !ok {
		panic("channel is not of type chan events.Message")
	}

	err, ok := args.Get(1).(chan error)
	if !ok {
		panic("error is not of type chan error")
	}
	return channel, err
}

func Test_handler_listContainers_happy(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/containers.json", nil)
	require.NoError(t, err, "NewRequest should not return an error.")

	rr := httptest.NewRecorder()

	mockedClient := new(MockedClient)
	containers := []docker.Container{
		{
			ID:      "1234567890",
			Status:  "status",
			State:   "state",
			Name:    "test",
			Created: 0,
			Command: "command",
			ImageID: "image_id",
			Image:   "image",
		},
	}
	mockedClient.On("ListContainers", mock.Anything).Return(containers, nil)

	h := handler{client: mockedClient}
	handler := http.HandlerFunc(h.listContainers)
	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
	mockedClient.AssertExpectations(t)
}

func Test_handler_streamLogs_happy(t *testing.T) {
	id := "123456"
	req, err := http.NewRequest("GET", "/api/logs/stream", nil)
	q := req.URL.Query()
	q.Add("id", id)
	req.URL.RawQuery = q.Encode()
	require.NoError(t, err, "NewRequest should not return an error.")

	rr := httptest.NewRecorder()

	mockedClient := new(MockedClient)

	messages := make(chan string)
	errChannel := make(chan error)
	mockedClient.On("ContainerLogs", mock.Anything, id, 300).Return(messages, errChannel)
	go func() {
		messages <- "INFO Testing logs..."
		close(messages)
	}()

	h := handler{client: mockedClient}
	handler := http.HandlerFunc(h.streamLogs)
	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
	mockedClient.AssertExpectations(t)
}

func Test_handler_streamLogs_error_reading(t *testing.T) {
	id := "123456"
	req, err := http.NewRequest("GET", "/api/logs/stream", nil)
	q := req.URL.Query()
	q.Add("id", "123456")
	req.URL.RawQuery = q.Encode()
	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()

	mockedClient := new(MockedClient)
	messages := make(chan string)
	errChannel := make(chan error)
	mockedClient.On("ContainerLogs", mock.Anything, id, 300).Return(messages, errChannel)

	go func() {
		errChannel <- errors.New("test error")
	}()

	h := handler{client: mockedClient}
	handler := http.HandlerFunc(h.streamLogs)
	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
	mockedClient.AssertExpectations(t)
}

func Test_handler_streamEvents_happy(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/events/stream", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()
	mockedClient := new(MockedClient)
	messages := make(chan events.Message)
	errChannel := make(chan error)
	mockedClient.On("Events", mock.Anything).Return(messages, errChannel)

	go func() {
		messages <- events.Message{
			Action: "start",
		}
		messages <- events.Message{
			Action: "something-random",
		}
		close(messages)
	}()

	h := handler{client: mockedClient}
	handler := http.HandlerFunc(h.streamEvents)
	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
	mockedClient.AssertExpectations(t)
}

func Test_handler_streamEvents_error(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/events/stream", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()
	mockedClient := new(MockedClient)
	messages := make(chan events.Message)
	errChannel := make(chan error)
	mockedClient.On("Events", mock.Anything).Return(messages, errChannel)

	go func() {
		errChannel <- errors.New("fake error")
		close(messages)
	}()

	h := handler{client: mockedClient}
	handler := http.HandlerFunc(h.streamEvents)
	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
	mockedClient.AssertExpectations(t)
}

func Test_handler_streamEvents_error_request(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/events/stream", nil)
	require.NoError(t, err, "NewRequest should not return an error.")

	rr := httptest.NewRecorder()

	mockedClient := new(MockedClient)

	messages := make(chan events.Message)
	errChannel := make(chan error)
	mockedClient.On("Events", mock.Anything).Return(messages, errChannel)

	ctx, cancel := context.WithCancel(context.Background())
	req = req.WithContext(ctx)

	go func() {
		cancel()
	}()

	h := handler{client: mockedClient}
	handler := http.HandlerFunc(h.streamEvents)
	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
	mockedClient.AssertExpectations(t)
}

func Test_createRoutes_index(t *testing.T) {
	mockedClient := new(MockedClient)
	box := packr.NewBox("./virtual")
	require.NoError(t, box.AddString("index.html", "index page"), "AddString should have no error.")

	handler := createRoutes("/", &handler{mockedClient, box})
	req, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
}

func Test_createRoutes_redirect(t *testing.T) {
	mockedClient := new(MockedClient)
	box := packr.NewBox("./virtual")

	handler := createRoutes("/foobar", &handler{mockedClient, box})
	req, err := http.NewRequest("GET", "/foobar", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
}

func Test_createRoutes_foobar(t *testing.T) {
	mockedClient := new(MockedClient)
	box := packr.NewBox("./virtual")
	require.NoError(t, box.AddString("index.html", "foo page"), "AddString should have no error.")

	handler := createRoutes("/foobar", &handler{mockedClient, box})
	req, err := http.NewRequest("GET", "/foobar/", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
}

func Test_createRoutes_foobar_file(t *testing.T) {
	mockedClient := new(MockedClient)
	box := packr.NewBox("./virtual")
	require.NoError(t, box.AddString("/test", "test page"), "AddString should have no error.")

	handler := createRoutes("/foobar", &handler{mockedClient, box})
	req, err := http.NewRequest("GET", "/foobar/test", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	assert.Equal(t, rr.Body.String(), "test page", "page doesn't match")
}

func Test_createRoutes_version(t *testing.T) {
	mockedClient := new(MockedClient)
	box := packr.NewBox("./virtual")

	handler := createRoutes("/", &handler{mockedClient, box})
	req, err := http.NewRequest("GET", "/version", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
}

func TestMain(m *testing.M) {
	exit := m.Run()
	abide.Cleanup()
	os.Exit(exit)
}
