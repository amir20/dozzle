package web

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/container/docker"
	"github.com/amir20/dozzle/internal/hostservice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_updateFor(t *testing.T) {
	events := []container.ContainerUpdateEvent{
		{Host: "h1", Name: "app", OldID: "old", NewID: "new", ToRef: "app:1"},
		{Host: "h2", Name: "app", OldID: "x", NewID: "new2"},
		{Host: "h1", Name: "app", OldID: "new", NewID: "new", ToRef: "app:2", RolledBack: true},
	}

	e, ok := updateFor(events, container.Container{ID: "new", Host: "h1"})
	require.True(t, ok)
	assert.True(t, e.RolledBack, "the newest event for the container wins")

	_, ok = updateFor(events, container.Container{ID: "new2", Host: "h1"})
	assert.False(t, ok, "an id on another host is not this container")

	_, ok = updateFor(events, container.Container{ID: "old", Host: "h1"})
	assert.False(t, ok, "the container an update replaced has no marker")
}

// historyService is a client service whose host keeps update events.
type historyService struct {
	container.ClientService
	events []container.ContainerUpdateEvent
}

func (s historyService) RecentUpdates() []container.ContainerUpdateEvent { return s.events }
func (s historyService) SubscribeUpdates(context.Context, chan<- container.ContainerUpdateEvent) {
}

// historyHosts serves the hosts of a real MultiHostService, with update history.
type historyHosts struct {
	*hostservice.MultiHostService
	services []container.ClientService
}

func (h historyHosts) ClientServices(bool) []container.ClientService { return h.services }

func Test_handler_streamLogs_sends_update_marker(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	id := "123456"
	started := time.Date(2026, time.October, 3, 3, 0, 0, 0, time.UTC)

	mockedClient := new(MockedClient)
	mockedClient.On("FindContainer", mock.Anything, id).Return(container.Container{ID: id, Name: "app", Host: "localhost", StartedAt: started}, nil)
	mockedClient.On("ContainerLogs", mock.Anything, mock.Anything, started, container.STDALL).
		Return(io.NopCloser(bytes.NewReader(makeMessage("INFO up\n", container.STDOUT))), nil).
		Run(func(mock.Arguments) {
			go func() {
				time.Sleep(50 * time.Millisecond)
				cancel()
			}()
		})
	mockedClient.On("Host").Return(container.Host{ID: "localhost"})
	mockedClient.On("ListContainers", mock.Anything, mock.Anything).Return([]container.Container{
		{ID: id, Name: "app", Host: "localhost", State: "running"},
	}, nil)
	mockedClient.On("ContainerEvents", mock.Anything, mock.AnythingOfType("chan<- container.ContainerEvent")).Return(nil).
		Run(func(mock.Arguments) { time.Sleep(50 * time.Millisecond) })

	service := docker.NewService(mockedClient, container.ContainerLabels{})
	manager := hostservice.NewRetriableClientManager(nil, nil, 3*time.Second, tls.Certificate{}, service)
	hosts := historyHosts{
		MultiHostService: hostservice.NewMultiHostService(manager, 3*time.Second),
		services: []container.ClientService{historyService{ClientService: service, events: []container.ContainerUpdateEvent{
			{Host: "localhost", Name: "app", OldID: "abcdef", NewID: id, FromRef: "app:1.4.1", ToRef: "app:1.4.2", Source: "schedule", At: started},
			{Host: "localhost", Name: "other", OldID: "zzz", NewID: "yyy", Source: "dozzle", At: started},
		}}},
	}
	router := createRouter(&handler{
		hostService: hosts,
		config:      &Config{Base: "/", Authorization: Authorization{Provider: NONE}},
	})

	req, err := http.NewRequestWithContext(ctx, "GET", "/api/hosts/localhost/containers/"+id+"/logs/stream", nil)
	require.NoError(t, err)
	q := req.URL.Query()
	q.Add("stdout", "true")
	q.Add("stderr", "true")
	addAllLogLevels(q)
	req.URL.RawQuery = q.Encode()

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	body := rr.Body.String()
	assert.Contains(t, body, "event: container-update\n")
	assert.Contains(t, body, `"newId":"123456"`)
	assert.Contains(t, body, `"toRef":"app:1.4.2"`)
	assert.NotContains(t, body, `"newId":"yyy"`, "only the streamed container's update is sent")
	assert.Less(t, bytes.Index(rr.Body.Bytes(), []byte("container-update")), bytes.Index(rr.Body.Bytes(), []byte("INFO up")),
		"the marker comes before the container's first line")
}
