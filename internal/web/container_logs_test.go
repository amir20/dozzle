package web

import (
	"archive/zip"
	"bytes"
	"io"
	"time"

	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/amir20/dozzle/internal/container"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_handler_download_logs(t *testing.T) {
	id := "123456"
	req, err := http.NewRequest("GET", "/api/containers/logs?format=zip&container=localhost/"+id+"&stdout=1", nil)
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
		req, err := http.NewRequest("GET", "/api/containers/logs?format=zip&container=localhost/"+id+"&stdout=1&filter=ERROR&inverse=true", nil)
		require.NoError(t, err)
		rr := httptest.NewRecorder()
		newHandler().ServeHTTP(rr, req)
		require.Equal(t, http.StatusOK, rr.Code)
		out := logContents(t, rr.Body.Bytes())
		require.Contains(t, out, "all good")
		require.NotContains(t, out, "boom")
	})

	t.Run("non-inverse keeps only matching lines", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/containers/logs?format=zip&container=localhost/"+id+"&stdout=1&filter=ERROR", nil)
		require.NoError(t, err)
		rr := httptest.NewRecorder()
		newHandler().ServeHTTP(rr, req)
		require.Equal(t, http.StatusOK, rr.Code)
		out := logContents(t, rr.Body.Bytes())
		require.Contains(t, out, "boom")
		require.NotContains(t, out, "all good")
	})
}

func Test_handler_download_logs_sanitizes_entry_names(t *testing.T) {
	id := "123456"
	name := "../../etc/cron.d/evil"
	req, err := http.NewRequest("GET", "/api/containers/logs?format=zip&container=localhost/"+id+"&stdout=1", nil)
	require.NoError(t, err)

	mockedClient := new(MockedClient)
	data := makeMessage("INFO Testing logs...", container.STDOUT)
	mockedClient.On("FindContainer", mock.Anything, id).Return(container.Container{ID: id, Name: name, Tty: false}, nil)
	mockedClient.On("ContainerLogsBetweenDates", mock.Anything, id, mock.Anything, mock.Anything, container.STDOUT).Return(io.NopCloser(bytes.NewReader(data)), nil)
	mockedClient.On("Host").Return(container.Host{ID: "localhost"})
	mockedClient.On("ContainerEvents", mock.Anything, mock.AnythingOfType("chan<- container.ContainerEvent")).Return(nil).Run(func(args mock.Arguments) {
		time.Sleep(1 * time.Second)
	})
	mockedClient.On("ListContainers", mock.Anything, mock.Anything).Return([]container.Container{
		{ID: id, Name: name, State: "running"},
	}, nil)

	rr := httptest.NewRecorder()
	createDefaultHandler(mockedClient).ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	body := rr.Body.Bytes()
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	require.NoError(t, err)
	require.Len(t, zr.File, 1)
	entry := zr.File[0].Name
	require.NotContains(t, entry, "/")
	require.NotContains(t, entry, "\\")
	require.False(t, strings.HasPrefix(entry, "."), "entry %q should not start with a dot", entry)
}

// textHandler serves the given containers, all on localhost, each with one line of logs.
func textHandler(t *testing.T, containers map[string]string) http.Handler {
	mockedClient := new(MockedClient)
	list := []container.Container{}
	for id, name := range containers {
		mockedClient.On("FindContainer", mock.Anything, id).Return(container.Container{ID: id, Name: name, Tty: false}, nil)
		mockedClient.On("ContainerLogsBetweenDates", mock.Anything, id, mock.Anything, mock.Anything, container.STDOUT).
			Return(io.NopCloser(bytes.NewReader(makeMessage("2020-05-13T18:55:37.772853839Z line from "+name+"\n", container.STDOUT))), nil)
		list = append(list, container.Container{ID: id, Name: name, State: "running"})
	}
	mockedClient.On("Host").Return(container.Host{ID: "localhost"})
	mockedClient.On("ContainerEvents", mock.Anything, mock.AnythingOfType("chan<- container.ContainerEvent")).Return(nil).Run(func(args mock.Arguments) {
		time.Sleep(1 * time.Second)
	})
	mockedClient.On("ListContainers", mock.Anything, mock.Anything).Return(list, nil)
	return createDefaultHandler(mockedClient)
}

func Test_handler_container_logs_text(t *testing.T) {
	get := func(t *testing.T, h http.Handler, query string) *httptest.ResponseRecorder {
		req, err := http.NewRequest("GET", "/api/containers/logs?stdout=1&"+query, nil)
		require.NoError(t, err)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		return rr
	}

	t.Run("one container is its lines with no header", func(t *testing.T) {
		rr := get(t, textHandler(t, map[string]string{"aaa111": "web"}), "container=localhost/aaa111")
		require.Equal(t, http.StatusOK, rr.Code)
		require.Equal(t, "text/plain; charset=UTF-8", rr.Header().Get("Content-Type"))
		require.Equal(t, "line from web\n", rr.Body.String())
	})

	t.Run("several containers get a header each, in request order", func(t *testing.T) {
		h := textHandler(t, map[string]string{"aaa111": "web", "bbb222": "db"})
		rr := get(t, h, "container=localhost/bbb222&container=localhost/aaa111")
		require.Equal(t, http.StatusOK, rr.Code)
		require.Equal(t, "── db ──\nline from db\n\n── web ──\nline from web\n", rr.Body.String())
	})

	t.Run("no container is a bad request", func(t *testing.T) {
		rr := get(t, textHandler(t, map[string]string{"aaa111": "web"}), "")
		require.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("a container without a host is a bad request", func(t *testing.T) {
		rr := get(t, textHandler(t, map[string]string{"aaa111": "web"}), "container=aaa111")
		require.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("an unknown host is not found, before anything is written", func(t *testing.T) {
		rr := get(t, textHandler(t, map[string]string{"aaa111": "web"}), "container=localhost/aaa111&container=elsewhere/aaa111")
		require.Equal(t, http.StatusNotFound, rr.Code)
		require.NotContains(t, rr.Body.String(), "line from")
	})
}

func Test_copyHeaders(t *testing.T) {
	service := func(name, host string) *container.ContainerService {
		return container.NewContainerService(nil, container.Container{Name: name, Host: host})
	}
	asked := 0
	hosts := func() []container.Host {
		asked++
		return []container.Host{{ID: "h1", Name: "prod-1"}, {ID: "h2", Name: "prod-2"}}
	}

	t.Run("names alone never ask for hosts", func(t *testing.T) {
		asked = 0
		require.Equal(t, []string{"web", "db"}, copyHeaders([]*container.ContainerService{service("web", "h1"), service("db", "h1")}, hosts))
		require.Zero(t, asked)
	})

	t.Run("a shared name adds the host, asking once", func(t *testing.T) {
		asked = 0
		got := copyHeaders([]*container.ContainerService{service("web", "h1"), service("web", "h2"), service("db", "h1")}, hosts)
		require.Equal(t, []string{"web (prod-1)", "web (prod-2)", "db"}, got)
		require.Equal(t, 1, asked)
	})
}
