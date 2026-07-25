package web

import (
	"archive/zip"
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

func Test_handler_download_logs_inverse_filter(t *testing.T) {
	id := "123456"
	data := append(
		makeMessage("2020-05-13T18:55:37.772853839Z ERROR boom\n", container.STDOUT),
		makeMessage("2020-05-13T18:56:37.772853839Z INFO all good\n", container.STDOUT)...,
	)

	newHandler := func() http.Handler {
		mockedClient := new(MockedClient)
		mockedClient.On("FindContainer", mock.Anything, id).Return(container.Container{ID: id, Tty: false}, nil)
		mockedClient.On("ContainerLogsBetweenDates", mock.Anything, id, mock.Anything, mock.Anything, container.STDOUT).Return(io.NopCloser(bytes.NewReader(data)), nil)
		mockedClient.On("Host").Return(container.Host{ID: "localhost"})
		mockedClient.On("ContainerEvents", mock.Anything, mock.AnythingOfType("chan<- container.ContainerEvent")).Return(nil).Run(func(args mock.Arguments) {
			time.Sleep(1 * time.Second)
		})
		mockedClient.On("ListContainers", mock.Anything, mock.Anything).Return([]container.Container{
			{ID: id, Name: "test", State: "running"},
		}, nil)
		return createDefaultHandler(mockedClient)
	}

	logContents := func(t *testing.T, body []byte) string {
		zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
		require.NoError(t, err)
		require.Len(t, zr.File, 1, "zip should contain a single log file")
		f, err := zr.File[0].Open()
		require.NoError(t, err)
		defer f.Close()
		content, err := io.ReadAll(f)
		require.NoError(t, err)
		return string(content)
	}

	t.Run("inverse excludes matching lines", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/containers/localhost~"+id+"/download?stdout=1&filter=ERROR&inverse=true", nil)
		require.NoError(t, err)
		rr := httptest.NewRecorder()
		newHandler().ServeHTTP(rr, req)
		require.Equal(t, http.StatusOK, rr.Code)
		out := logContents(t, rr.Body.Bytes())
		require.Contains(t, out, "all good")
		require.NotContains(t, out, "boom")
	})

	t.Run("non-inverse keeps only matching lines", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/containers/localhost~"+id+"/download?stdout=1&filter=ERROR", nil)
		require.NoError(t, err)
		rr := httptest.NewRecorder()
		newHandler().ServeHTTP(rr, req)
		require.Equal(t, http.StatusOK, rr.Code)
		out := logContents(t, rr.Body.Bytes())
		require.Contains(t, out, "boom")
		require.NotContains(t, out, "all good")
	})
}
