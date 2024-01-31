package web

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"time"

	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/amir20/dozzle/internal/docker"
	"github.com/beme/abide"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_handler_streamLogs_happy(t *testing.T) {
	id := "123456"
	req, err := http.NewRequest("GET", "/api/logs/stream/localhost/"+id, nil)
	q := req.URL.Query()
	q.Add("stdout", "true")
	q.Add("stderr", "true")

	req.URL.RawQuery = q.Encode()
	require.NoError(t, err, "NewRequest should not return an error.")

	mockedClient := new(MockedClient)

	data := makeMessage("INFO Testing logs...", docker.STDOUT)

	mockedClient.On("FindContainer", id).Return(docker.Container{ID: id, Tty: false}, nil)
	mockedClient.On("ContainerLogs", mock.Anything, mock.Anything, "", docker.STDALL).Return(io.NopCloser(bytes.NewReader(data)), nil)

	handler := createDefaultHandler(mockedClient)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
	mockedClient.AssertExpectations(t)
}

func Test_handler_streamLogs_happy_with_id(t *testing.T) {
	id := "123456"
	req, err := http.NewRequest("GET", "/api/logs/stream/localhost/"+id, nil)
	q := req.URL.Query()
	q.Add("stdout", "true")
	q.Add("stderr", "true")

	req.URL.RawQuery = q.Encode()
	require.NoError(t, err, "NewRequest should not return an error.")

	mockedClient := new(MockedClient)

	data := makeMessage("2020-05-13T18:55:37.772853839Z INFO Testing logs...", docker.STDOUT)

	mockedClient.On("FindContainer", id).Return(docker.Container{ID: id}, nil)
	mockedClient.On("ContainerLogs", mock.Anything, mock.Anything, "", docker.STDALL).Return(io.NopCloser(bytes.NewReader(data)), nil)

	handler := createDefaultHandler(mockedClient)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
	mockedClient.AssertExpectations(t)
}

func Test_handler_streamLogs_happy_container_stopped(t *testing.T) {
	id := "123456"
	req, err := http.NewRequest("GET", "/api/logs/stream/localhost/"+id, nil)
	q := req.URL.Query()
	q.Add("stdout", "true")
	q.Add("stderr", "true")

	req.URL.RawQuery = q.Encode()
	require.NoError(t, err, "NewRequest should not return an error.")

	mockedClient := new(MockedClient)
	mockedClient.On("FindContainer", id).Return(docker.Container{ID: id}, nil)
	mockedClient.On("ContainerLogs", mock.Anything, id, "", docker.STDALL).Return(io.NopCloser(strings.NewReader("")), io.EOF)

	handler := createDefaultHandler(mockedClient)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
	mockedClient.AssertExpectations(t)
}

func Test_handler_streamLogs_error_finding_container(t *testing.T) {
	id := "123456"
	req, err := http.NewRequest("GET", "/api/logs/stream/localhost/"+id, nil)
	q := req.URL.Query()
	q.Add("stdout", "true")
	q.Add("stderr", "true")

	req.URL.RawQuery = q.Encode()
	require.NoError(t, err, "NewRequest should not return an error.")

	mockedClient := new(MockedClient)
	mockedClient.On("FindContainer", id).Return(docker.Container{}, errors.New("error finding container"))

	handler := createDefaultHandler(mockedClient)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
	mockedClient.AssertExpectations(t)
}

func Test_handler_streamLogs_error_reading(t *testing.T) {
	id := "123456"
	req, err := http.NewRequest("GET", "/api/logs/stream/localhost/"+id, nil)
	q := req.URL.Query()
	q.Add("stdout", "true")
	q.Add("stderr", "true")

	req.URL.RawQuery = q.Encode()
	require.NoError(t, err, "NewRequest should not return an error.")

	mockedClient := new(MockedClient)
	mockedClient.On("FindContainer", id).Return(docker.Container{ID: id}, nil)
	mockedClient.On("ContainerLogs", mock.Anything, id, "", docker.STDALL).Return(io.NopCloser(strings.NewReader("")), errors.New("test error"))

	handler := createDefaultHandler(mockedClient)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
	mockedClient.AssertExpectations(t)
}

func Test_handler_streamLogs_error_std(t *testing.T) {
	id := "123456"
	req, err := http.NewRequest("GET", "/api/logs/stream/localhost/"+id, nil)

	require.NoError(t, err, "NewRequest should not return an error.")

	mockedClient := new(MockedClient)

	handler := createDefaultHandler(mockedClient)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
	mockedClient.AssertExpectations(t)
}

// for /api/logs
func Test_handler_between_dates(t *testing.T) {
	id := "123456"
	req, err := http.NewRequest("GET", "/api/logs/localhost/"+id, nil)
	require.NoError(t, err, "NewRequest should not return an error.")

	from, _ := time.Parse(time.RFC3339, "2018-01-01T00:00:00Z")
	to, _ := time.Parse(time.RFC3339, "2018-01-01T010:00:00Z")

	q := req.URL.Query()
	q.Add("from", from.Format(time.RFC3339))
	q.Add("to", to.Format(time.RFC3339))
	q.Add("stdout", "true")
	q.Add("stderr", "true")

	req.URL.RawQuery = q.Encode()

	mockedClient := new(MockedClient)

	first := makeMessage("2020-05-13T18:55:37.772853839Z INFO Testing stdout logs...\n", docker.STDOUT)
	second := makeMessage("2020-05-13T18:56:37.772853839Z INFO Testing stderr logs...\n", docker.STDERR)
	data := append(first, second...)

	mockedClient.On("ContainerLogsBetweenDates", mock.Anything, id, from, to, docker.STDALL).Return(io.NopCloser(bytes.NewReader(data)), nil)
	mockedClient.On("FindContainer", id).Return(docker.Container{ID: id}, nil)

	handler := createDefaultHandler(mockedClient)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	abide.AssertHTTPResponse(t, t.Name(), rr.Result())
	mockedClient.AssertExpectations(t)
}

func makeMessage(message string, stream docker.StdType) []byte {
	data := make([]byte, 8)
	binary.BigEndian.PutUint32(data[4:], uint32(len(message)))
	data[0] = byte(stream / 2)
	data = append(data, []byte(message)...)

	return data
}
