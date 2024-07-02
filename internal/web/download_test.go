package web

import (
	"bytes"
	"compress/gzip"
	"io"
	"time"

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
	req, err := http.NewRequest("GET", "/api/hosts/localhost/containers/"+id+"/logs/download?stdout=1", nil)
	require.NoError(t, err, "NewRequest should not return an error.")

	mockedClient := new(MockedClient)

	data := makeMessage("INFO Testing logs...", docker.STDOUT)

	mockedClient.On("FindContainer", id).Return(docker.Container{ID: id, Tty: false}, nil).Once()
	mockedClient.On("ContainerLogsBetweenDates", mock.Anything, id, mock.Anything, mock.Anything, docker.STDOUT).Return(io.NopCloser(bytes.NewReader(data)), nil)
	mockedClient.On("Host").Return(docker.Host{
		ID: "localhost",
	})
	mockedClient.On("ContainerEvents", mock.Anything, mock.AnythingOfType("chan<- docker.ContainerEvent")).Return(nil).Run(func(args mock.Arguments) {
		time.Sleep(1 * time.Second)
	})
	mockedClient.On("ListContainers").Return([]docker.Container{
		{ID: id, Name: "test"},
	}, nil)

	handler := createDefaultHandler(mockedClient)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	reader, _ := gzip.NewReader(rr.Body)
	abide.AssertReader(t, t.Name(), reader)
	mockedClient.AssertExpectations(t)
}
