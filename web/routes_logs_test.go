package web

import (
	"context"
	"errors"
	"io"
	"time"

	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/amir20/dozzle/docker"
	"github.com/beme/abide"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_handler_streamLogs_happy(t *testing.T) {
	id := "123456"
	req, err := http.NewRequest("GET", "/api/logs/stream", nil)
	q := req.URL.Query()
	q.Add("id", id)
	req.URL.RawQuery = q.Encode()
	require.NoError(t, err, "NewRequest should not return an error.")

	mockedClient := new(MockedClient)
	reader := ioutil.NopCloser(strings.NewReader("INFO Testing logs..."))
	mockedClient.On("FindContainer", id).Return(docker.Container{ID: id}, nil)
	mockedClient.On("ContainerLogs", mock.Anything, mock.Anything, "").Return(reader, nil)

	clients := map[string]docker.Client{
		"localhost": mockedClient,
	}
	h := handler{clients: clients, config: &Config{}}
	handler := http.HandlerFunc(h.streamLogs)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
	mockedClient.AssertExpectations(t)
}

func Test_handler_streamLogs_happy_with_id(t *testing.T) {
	id := "123456"
	req, err := http.NewRequest("GET", "/api/logs/stream", nil)
	q := req.URL.Query()
	q.Add("id", id)
	req.URL.RawQuery = q.Encode()
	require.NoError(t, err, "NewRequest should not return an error.")

	mockedClient := new(MockedClient)
	reader := ioutil.NopCloser(strings.NewReader("2020-05-13T18:55:37.772853839Z INFO Testing logs..."))
	mockedClient.On("FindContainer", id).Return(docker.Container{ID: id}, nil)
	mockedClient.On("ContainerLogs", mock.Anything, mock.Anything, "").Return(reader, nil)

	clients := map[string]docker.Client{
		"localhost": mockedClient,
	}
	h := handler{clients: clients, config: &Config{}}
	handler := http.HandlerFunc(h.streamLogs)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
	mockedClient.AssertExpectations(t)
}

func Test_handler_streamLogs_happy_container_stopped(t *testing.T) {
	id := "123456"
	req, err := http.NewRequest("GET", "/api/logs/stream", nil)
	q := req.URL.Query()
	q.Add("id", id)
	req.URL.RawQuery = q.Encode()
	require.NoError(t, err, "NewRequest should not return an error.")

	mockedClient := new(MockedClient)
	mockedClient.On("FindContainer", id).Return(docker.Container{ID: id}, nil)
	mockedClient.On("ContainerLogs", mock.Anything, id, "").Return(ioutil.NopCloser(strings.NewReader("")), io.EOF)

	clients := map[string]docker.Client{
		"localhost": mockedClient,
	}
	h := handler{clients: clients, config: &Config{}}
	handler := http.HandlerFunc(h.streamLogs)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
	mockedClient.AssertExpectations(t)
}

func Test_handler_streamLogs_error_finding_container(t *testing.T) {
	id := "123456"
	req, err := http.NewRequest("GET", "/api/logs/stream", nil)
	q := req.URL.Query()
	q.Add("id", id)
	req.URL.RawQuery = q.Encode()
	require.NoError(t, err, "NewRequest should not return an error.")

	mockedClient := new(MockedClient)
	mockedClient.On("FindContainer", id).Return(docker.Container{}, errors.New("error finding container"))

	clients := map[string]docker.Client{
		"localhost": mockedClient,
	}
	h := handler{clients: clients, config: &Config{}}
	handler := http.HandlerFunc(h.streamLogs)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
	mockedClient.AssertExpectations(t)
}

func Test_handler_streamLogs_error_reading(t *testing.T) {
	id := "123456"
	req, err := http.NewRequest("GET", "/api/logs/stream", nil)
	q := req.URL.Query()
	q.Add("id", id)
	req.URL.RawQuery = q.Encode()
	require.NoError(t, err, "NewRequest should not return an error.")

	mockedClient := new(MockedClient)
	mockedClient.On("FindContainer", id).Return(docker.Container{ID: id}, nil)
	mockedClient.On("ContainerLogs", mock.Anything, id, "").Return(ioutil.NopCloser(strings.NewReader("")), errors.New("test error"))

	clients := map[string]docker.Client{
		"localhost": mockedClient,
	}
	h := handler{clients: clients, config: &Config{}}
	handler := http.HandlerFunc(h.streamLogs)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
	mockedClient.AssertExpectations(t)
}

func Test_handler_streamEvents_happy(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/events/stream", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	mockedClient := new(MockedClient)
	messages := make(chan docker.ContainerEvent)
	errChannel := make(chan error)
	mockedClient.On("Events", mock.Anything).Return(messages, errChannel)
	mockedClient.On("ListContainers").Return([]docker.Container{}, nil)

	go func() {
		messages <- docker.ContainerEvent{
			Name:    "start",
			ActorID: "1234",
		}
		messages <- docker.ContainerEvent{
			Name:    "something-random",
			ActorID: "1234",
		}
		close(messages)
	}()

	clients := map[string]docker.Client{
		"localhost": mockedClient,
	}
	h := handler{clients: clients, config: &Config{}}
	handler := http.HandlerFunc(h.streamEvents)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
	mockedClient.AssertExpectations(t)
}

func Test_handler_streamEvents_error(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/events/stream", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	mockedClient := new(MockedClient)
	messages := make(chan docker.ContainerEvent)
	errChannel := make(chan error)
	mockedClient.On("Events", mock.Anything).Return(messages, errChannel)
	mockedClient.On("ListContainers").Return([]docker.Container{}, nil)

	go func() {
		errChannel <- errors.New("fake error")
		close(messages)
	}()

	clients := map[string]docker.Client{
		"localhost": mockedClient,
	}
	h := handler{clients: clients, config: &Config{}}
	handler := http.HandlerFunc(h.streamEvents)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
	mockedClient.AssertExpectations(t)
}

func Test_handler_streamEvents_error_request(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/events/stream", nil)
	require.NoError(t, err, "NewRequest should not return an error.")

	mockedClient := new(MockedClient)

	messages := make(chan docker.ContainerEvent)
	errChannel := make(chan error)
	mockedClient.On("Events", mock.Anything).Return(messages, errChannel)
	mockedClient.On("ListContainers").Return([]docker.Container{}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	req = req.WithContext(ctx)

	go func() {
		cancel()
	}()

	clients := map[string]docker.Client{
		"localhost": mockedClient,
	}
	h := handler{clients: clients, config: &Config{}}
	handler := http.HandlerFunc(h.streamEvents)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
	mockedClient.AssertExpectations(t)
}

// for /api/logs
func Test_handler_between_dates(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/logs", nil)
	require.NoError(t, err, "NewRequest should not return an error.")

	from, _ := time.Parse(time.RFC3339, "2018-01-01T00:00:00Z")
	to, _ := time.Parse(time.RFC3339, "2018-01-01T010:00:00Z")

	q := req.URL.Query()
	q.Add("from", from.Format(time.RFC3339))
	q.Add("to", to.Format(time.RFC3339))
	q.Add("id", "123456")
	req.URL.RawQuery = q.Encode()

	mockedClient := new(MockedClient)
	reader := ioutil.NopCloser(strings.NewReader("2020-05-13T18:55:37.772853839Z INFO Testing logs...\n2020-05-13T18:55:37.772853839Z INFO Testing logs...\n"))
	mockedClient.On("ContainerLogsBetweenDates", mock.Anything, "123456", from, to).Return(reader, nil)

	clients := map[string]docker.Client{
		"localhost": mockedClient,
	}
	h := handler{clients: clients, config: &Config{}}
	handler := http.HandlerFunc(h.fetchLogsBetweenDates)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
	mockedClient.AssertExpectations(t)
}
