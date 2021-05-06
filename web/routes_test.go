package web

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
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
	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "index.html", []byte("index page"), 0644), "WriteFile should have no error.")
	handler := createHandler(nil, afero.NewIOFS(fs), Config{Base: "/"})
	req, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
}

func Test_createRoutes_redirect(t *testing.T) {
	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "index.html", []byte("index page"), 0644), "WriteFile should have no error.")

	handler := createHandler(nil, afero.NewIOFS(fs), Config{Base: "/foobar"})
	req, err := http.NewRequest("GET", "/foobar", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
}

func Test_createRoutes_redirect_with_auth(t *testing.T) {
	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "index.html", []byte("index page"), 0644), "WriteFile should have no error.")

	handler := createHandler(nil, afero.NewIOFS(fs), Config{Base: "/foobar", Username: "amir", Password: "password", Key: "key"})
	req, err := http.NewRequest("GET", "/foobar/", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
}

func Test_createRoutes_foobar(t *testing.T) {
	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "index.html", []byte("foo page"), 0644), "WriteFile should have no error.")
	handler := createHandler(nil, afero.NewIOFS(fs), Config{Base: "/foobar"})
	req, err := http.NewRequest("GET", "/foobar/", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
}

func Test_createRoutes_foobar_file(t *testing.T) {
	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "index.html", []byte("index page"), 0644), "WriteFile should have no error.")
	require.NoError(t, afero.WriteFile(fs, "test", []byte("test page"), 0644), "WriteFile should have no error.")

	handler := createHandler(nil, afero.NewIOFS(fs), Config{Base: "/foobar"})
	req, err := http.NewRequest("GET", "/foobar/test", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	assert.Equal(t, rr.Body.String(), "test page", "page doesn't match")
}

func Test_createRoutes_version(t *testing.T) {
	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "index.html", []byte("index page"), 0644), "WriteFile should have no error.")
	handler := createHandler(nil, afero.NewIOFS(fs), Config{Base: "/", Version: "dev"})
	req, err := http.NewRequest("GET", "/version", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
}

func Test_createRoutes_username_password(t *testing.T) {

	handler := createHandler(nil, nil, Config{Base: "/", Username: "amir", Password: "password", Key: "key"})
	req, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
}

func Test_createRoutes_username_password_invalid(t *testing.T) {
	handler := createHandler(nil, nil, Config{Base: "/", Username: "amir", Password: "password", Key: "key"})
	req, err := http.NewRequest("GET", "/api/logs/stream?id=123", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
}

func Test_createRoutes_username_password_login_happy(t *testing.T) {
	handler := createHandler(nil, nil, Config{Base: "/", Username: "amir", Password: "password", Key: "key"})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	fw, err := writer.CreateFormField("username")
	require.NoError(t, err, "Creating field should not be error.")
	_, err = io.Copy(fw, strings.NewReader("amir"))
	require.NoError(t, err, "Copying field should not result in error.")

	fw, err = writer.CreateFormField("password")
	require.NoError(t, err, "Creating field should not be error.")
	_, err = io.Copy(fw, strings.NewReader("password"))
	require.NoError(t, err, "Copying field should not result in error.")

	writer.Close()

	req, err := http.NewRequest("POST", "/api/validateCredentials", bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", writer.FormDataContentType())

	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, rr.Code, 200)
	cookie := rr.Header().Get("Set-Cookie")
	assert.Matches(t, cookie, "session=.+")
}

func Test_createRoutes_username_password_login_failed(t *testing.T) {
	handler := createHandler(nil, nil, Config{Base: "/", Username: "amir", Password: "password", Key: "key"})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	fw, err := writer.CreateFormField("username")
	require.NoError(t, err, "Creating field should not be error.")
	_, err = io.Copy(fw, strings.NewReader("amir"))
	require.NoError(t, err, "Copying field should not result in error.")

	fw, err = writer.CreateFormField("password")
	require.NoError(t, err, "Creating field should not be error.")
	_, err = io.Copy(fw, strings.NewReader("bad"))
	require.NoError(t, err, "Copying field should not result in error.")

	writer.Close()

	req, err := http.NewRequest("POST", "/api/validateCredentials", bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", writer.FormDataContentType())

	require.NoError(t, err, "NewRequest should not return an error.")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, rr.Code, 401)
}

func Test_createRoutes_username_password_valid_session(t *testing.T) {
	mockedClient := new(MockedClient)
	mockedClient.On("FindContainer", "123").Return(docker.Container{ID: "123"}, nil)
	mockedClient.On("ContainerLogs", mock.Anything, "123", 0).Return(ioutil.NopCloser(strings.NewReader("test data")), io.EOF)
	handler := createHandler(mockedClient, nil, Config{Base: "/", Username: "amir", Password: "password", Key: "key"})

	// Get cookie first
	req, err := http.NewRequest("GET", "/api/logs/stream?id=123", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	session, _ := store.Get(req, sessionName)
	session.Values[authorityKey] = time.Now().Unix()
	recorder := httptest.NewRecorder()
	session.Save(req, recorder)
	cookies := recorder.Result().Cookies()

	// Test with cookie
	req, err = http.NewRequest("GET", "/api/logs/stream?id=123", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	req.AddCookie(cookies[0])
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
}

func Test_createRoutes_username_password_invalid_session(t *testing.T) {
	mockedClient := new(MockedClient)
	mockedClient.On("FindContainer", "123").Return(docker.Container{ID: "123"}, nil)
	mockedClient.On("ContainerLogs", mock.Anything, "123", 0).Return(ioutil.NopCloser(strings.NewReader("test data")), io.EOF)
	handler := createHandler(mockedClient, nil, Config{Base: "/", Username: "amir", Password: "password", Key: "key"})
	req, err := http.NewRequest("GET", "/api/logs/stream?id=123", nil)
	require.NoError(t, err, "NewRequest should not return an error.")
	req.AddCookie(&http.Cookie{Name: "session", Value: "baddata"})
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, rr.Code, 401)
}

func createHandler(client docker.Client, content fs.FS, config Config) *mux.Router {
	if client == nil {
		client = new(MockedClient)
	}

	if content == nil {
		fs := afero.NewMemMapFs()
		afero.WriteFile(fs, "index.html", []byte("index page"), 0644)
		content = afero.NewIOFS(fs)
	}

	return createRouter(&handler{
		client:  client,
		content: content,
		config:  &config,
	})
}

func TestMain(m *testing.M) {
	exit := m.Run()
	abide.Cleanup()
	os.Exit(exit)
}
