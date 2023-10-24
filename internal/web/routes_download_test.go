package web

import (
	"bytes"
	"compress/gzip"
	"io"

	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amir20/dozzle/internal/docker"
	"github.com/beme/abide"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_handler_download_logs(t *testing.T) {
	id := "123456"
	req, err := http.NewRequest("GET", "/api/logs/download/localhost/"+id, nil)
	require.NoError(t, err, "NewRequest should not return an error.")

	mockedClient := new(MockedClient)

	data := makeMessage("INFO Testing logs...", docker.STDOUT)

	mockedClient.On("FindContainer", id).Return(docker.Container{ID: id, Tty: false}, nil)
	mockedClient.On("ContainerLogsBetweenDates", mock.Anything, id, mock.Anything, mock.Anything, docker.STDALL).Return(io.NopCloser(bytes.NewReader(data)), nil)

	handler := createDefaultHandler(mockedClient)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	reader, _ := gzip.NewReader(rr.Body)
	abide.AssertReader(t, t.Name(), reader)
	mockedClient.AssertExpectations(t)
}
