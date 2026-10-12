package web

import (
	"archive/zip"
	"bytes"
	"encoding/json"
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

func Test_writeRawText(t *testing.T) {
	long := strings.Repeat("x", 200*1024)
	in := "2026-10-12T10:00:00.000000001Z plain line\n" +
		"2026-10-12T10:00:01.5Z level=info msg=\"kept as printed\"\n" +
		"2026-10-12T10:00:02Z \x1b[31mred\x1b[0m\n" +
		"2026-10-12T10:00:03Z \n" +
		"2026-10-12T10:00:04Z " + long + "\n" +
		"2026-10-12T10:00:05Z no newline at the end"
	var out bytes.Buffer
	require.NoError(t, writeRawText(&out, strings.NewReader(in)))
	require.Equal(t, "plain line\n"+
		"level=info msg=\"kept as printed\"\n"+
		"red\n"+
		"\n"+
		long+"\n"+
		"no newline at the end\n", out.String())
}

func Test_handler_container_logs_text_filtered_parses(t *testing.T) {
	h := textHandler(t, map[string]string{"aaa111": "web"})
	req, err := http.NewRequest("GET", "/api/containers/logs?stdout=1&container=localhost/aaa111&filter=nomatch", nil)
	require.NoError(t, err)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "", rr.Body.String(), "a search that matches nothing copies nothing")
}

// jsonlHandler serves containers on localhost, each with the given raw log lines.
func jsonlHandler(t *testing.T, logs map[string][]byte) http.Handler {
	t.Helper()
	mockedClient := new(MockedClient)
	list := []container.Container{}
	for id, data := range logs {
		mockedClient.On("FindContainer", mock.Anything, id).Return(container.Container{ID: id, Name: id}, nil)
		mockedClient.On("ContainerLogsBetweenDates", mock.Anything, id, mock.Anything, mock.Anything, container.STDALL).
			Return(io.NopCloser(bytes.NewReader(data)), nil).
			Once()
		list = append(list, container.Container{ID: id, Name: id, State: "running"})
	}
	mockedClient.On("Host").Return(container.Host{ID: "localhost"})
	mockedClient.On("ContainerEvents", mock.Anything, mock.AnythingOfType("chan<- container.ContainerEvent")).Return(nil)
	mockedClient.On("ListContainers", mock.Anything, mock.Anything).Return(list, nil)
	return createDefaultHandler(mockedClient)
}

// getJSONL fetches format=jsonl for query and decodes the events it wrote.
func getJSONL(t *testing.T, h http.Handler, query string) []*container.LogEvent {
	t.Helper()
	req, err := http.NewRequest("GET", "/api/containers/logs?format=jsonl&stdout=1&stderr=1&"+query, nil)
	require.NoError(t, err)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "application/x-jsonl; charset=UTF-8", rr.Header().Get("Content-Type"))

	var events []*container.LogEvent
	decoder := json.NewDecoder(rr.Body)
	for decoder.More() {
		event := &container.LogEvent{}
		require.NoError(t, decoder.Decode(event))
		events = append(events, event)
	}
	return events
}

// jsonOnly feeds the SQL analytics view, which builds a DuckDB table with
// `unnest(m)`. That only works when every `m` is an object, so anything that is
// not a complex event has to be dropped here. Filtering on "the message is not a
// string" is not enough: grouped events carry a []string, which makes DuckDB
// infer the column as JSON and fails the whole table with a binder error.
func Test_handler_container_logs_jsonl_jsonOnly(t *testing.T) {
	// A single line, a two line group (an ERROR followed by an unlevelled
	// continuation close in time), and a complex JSON line.
	data := concatMessages(
		"2020-05-13T18:55:37.772853839Z INFO Testing stdout logs...\n",
		"2020-05-13T18:55:38.772853839Z ERROR something blew up\n",
		"2020-05-13T18:55:38.782853839Z     at foo.bar(baz.go:1)\n",
		"2020-05-13T18:56:37.772853839Z {\"msg\":\"a complex log message\"}\n",
	)

	tests := []struct {
		name      string
		query     string
		wantTypes []container.LogType
	}{
		{
			name:      "jsonOnly keeps complex events only",
			query:     "jsonOnly=1",
			wantTypes: []container.LogType{container.LogTypeComplex},
		},
		{
			name:      "without jsonOnly every event is sent",
			wantTypes: []container.LogType{container.LogTypeSingle, container.LogTypeGroup, container.LogTypeComplex},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := jsonlHandler(t, map[string][]byte{"123456": data})
			events := getJSONL(t, h, "container=localhost/123456&"+tt.query)

			types := make([]container.LogType, 0, len(events))
			for _, event := range events {
				types = append(types, event.Type)
			}
			require.Equal(t, tt.wantTypes, types)

			if tt.query == "" {
				return
			}
			for _, event := range events {
				// unnest(m) rejects anything that is not a struct, so every
				// message the analytics view sees must decode to an object.
				require.IsType(t, map[string]any{}, event.Message)
			}
		})
	}
}

// The SQL view loads a merged view in one request and takes `_container` from `c`,
// so every event has to name its container and containers must not interleave.
func Test_handler_container_logs_jsonl_several_containers(t *testing.T) {
	h := jsonlHandler(t, map[string][]byte{
		"aaa111": concatMessages(
			"2020-05-13T18:55:37.772853839Z {\"from\":\"a\",\"n\":1}\n",
			"2020-05-13T18:56:37.772853839Z {\"from\":\"a\",\"n\":2}\n",
		),
		"bbb222": concatMessages("2020-05-13T18:55:00.000000000Z {\"from\":\"b\"}\n"),
	})

	events := getJSONL(t, h, "jsonOnly=1&container=localhost/bbb222&container=localhost/aaa111")

	got := make([]string, 0, len(events))
	for _, event := range events {
		got = append(got, event.ContainerID+":"+event.Message.(map[string]any)["from"].(string))
	}
	require.Equal(t, []string{"bbb222:b", "aaa111:a", "aaa111:a"}, got)
}
