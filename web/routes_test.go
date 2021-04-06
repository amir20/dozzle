package web

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/magiconair/properties/assert"

	"github.com/amir20/dozzle/docker"
	"github.com/beme/abide"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/spf13/afero"
)

type MockedClient struct {
	mock.Mock
	docker.Client
}

func (m *MockedClient) FindContainer(id string) (docker.Container, error) {
	args := m.Called(id)
	return args.Get(0).(docker.Container), args.Error(1)
}

func (m *MockedClient) ListContainers() ([]docker.Container, error) {
	args := m.Called()
	return args.Get(0).([]docker.Container), args.Error(1)
}

func (m *MockedClient) ContainerLogs(ctx context.Context, id string, tailSize int, since string) (io.ReadCloser, error) {
	args := m.Called(ctx, id, tailSize)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockedClient) Events(ctx context.Context) (<-chan docker.ContainerEvent, <-chan error) {
	args := m.Called(ctx)
	channel, ok := args.Get(0).(chan docker.ContainerEvent)
	if !ok {
		panic("channel is not of type chan events.Message")
	}

	err, ok := args.Get(1).(chan error)
	if !ok {
		panic("error is not of type chan error")
	}
	return channel, err
}

func (m *MockedClient) ContainerStats(context.Context, string, chan<- docker.ContainerStat) error {
	return nil
}

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
	mockedClient.On("ContainerLogs", mock.Anything, mock.Anything, 300).Return(reader, nil)

	h := handler{client: mockedClient, config: &Config{TailSize: 300}}
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
	mockedClient.On("ContainerLogs", mock.Anything, mock.Anything, 300).Return(reader, nil)

	h := handler{client: mockedClient, config: &Config{TailSize: 300}}
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
	mockedClient.On("ContainerLogs", mock.Anything, id, 300).Return(ioutil.NopCloser(strings.NewReader("")), io.EOF)

	h := handler{client: mockedClient, config: &Config{TailSize: 300}}
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

	h := handler{client: mockedClient, config: &Config{TailSize: 300}}
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
	mockedClient.On("ContainerLogs", mock.Anything, id, 300).Return(ioutil.NopCloser(strings.NewReader("")), errors.New("test error"))

	h := handler{client: mockedClient, config: &Config{TailSize: 300}}
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

	h := handler{client: mockedClient, config: &Config{TailSize: 300}}
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

	h := handler{client: mockedClient, config: &Config{TailSize: 300}}
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

	h := handler{client: mockedClient, config: &Config{TailSize: 300}}
	handler := http.HandlerFunc(h.streamEvents)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
	mockedClient.AssertExpectations(t)
}

func Test_createRoutes_index(t *testing.T) {
	mockedClient := new(MockedClient)
	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "index.html", []byte("index page"), 0644), "WriteFile should have no error.")
	handler := createRouter(&handler{
		client:  mockedClient,
		content: afero.NewIOFS(fs),
		config:  &Config{Base: "/"},
	})
	req, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
}

func Test_createRoutes_redirect(t *testing.T) {
	mockedClient := new(MockedClient)
	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "index.html", []byte("index page"), 0644), "WriteFile should have no error.")

	handler := createRouter(&handler{
		client:  mockedClient,
		content: afero.NewIOFS(fs),
		config:  &Config{Base: "/foobar"},
	})
	req, err := http.NewRequest("GET", "/foobar", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
}

func Test_createRoutes_foobar(t *testing.T) {
	mockedClient := new(MockedClient)
	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "index.html", []byte("foo page"), 0644), "WriteFile should have no error.")

	handler := createRouter(&handler{
		client:  mockedClient,
		content: afero.NewIOFS(fs),
		config:  &Config{Base: "/foobar"},
	})
	req, err := http.NewRequest("GET", "/foobar/", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
}

func Test_createRoutes_foobar_file(t *testing.T) {
	mockedClient := new(MockedClient)
	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "index.html", []byte("index page"), 0644), "WriteFile should have no error.")
	require.NoError(t, afero.WriteFile(fs, "test", []byte("test page"), 0644), "WriteFile should have no error.")

	handler := createRouter(&handler{
		client:     mockedClient,
		content:    afero.NewIOFS(fs),
		config:     &Config{Base: "/foobar"},
	})
	req, err := http.NewRequest("GET", "/foobar/test", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	assert.Equal(t, rr.Body.String(), "test page", "page doesn't match")
}

func Test_createRoutes_version(t *testing.T) {
	mockedClient := new(MockedClient)
	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "index.html", []byte("index page"), 0644), "WriteFile should have no error.")

	handler := createRouter(&handler{
		client:  mockedClient,
		content: afero.NewIOFS(fs),
		config:  &Config{Base: "/", Version: "dev"},
	})
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
