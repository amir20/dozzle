package web

import (
	"bytes"
	"io"
	"time"

	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amir20/dozzle/internal/container"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_handler_download_logs(t *testing.T) {
	id := "123456"
	req, err := http.NewRequest("GET", "/api/containers/localhost~"+id+"/download?stdout=1", nil)
	require.NoError(t, err, "NewRequest should not return an error.")

	mockedClient := new(MockedClient)

	data := makeMessage("INFO Testing logs...", container.STDOUT)

	mockedClient.On("FindContainer", mock.Anything, id).Return(container.Container{ID: id, Tty: false}, nil)
	mockedClient.On("ContainerLogsBetweenDates", mock.Anything, id, mock.Anything, mock.Anything, container.STDOUT).Return(io.NopCloser(bytes.NewReader(data)), nil)
	mockedClient.On("Host").Return(container.Host{
		ID: "localhost",
	})
	mockedClient.On("ContainerEvents", mock.Anything, mock.AnythingOfType("chan<- container.ContainerEvent")).Return(nil).Run(func(args mock.Arguments) {
		time.Sleep(1 * time.Second)
	})
	mockedClient.On("ListContainers", mock.Anything, mock.Anything).Return([]container.Container{
		{ID: id, Name: "test", State: "running"},
	}, nil)

	handler := createDefaultHandler(mockedClient)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code, "Status code should be 200.")
	mockedClient.AssertExpectations(t)
}
