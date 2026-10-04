package docker

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// podmanServer answers /events with one record and /containers/{id}/json with
// the given health block, both byte for byte in Podman 5.x's shape.
func podmanServer(t *testing.T, event string, inspect string) *Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/events"):
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, event+"\n")
			w.(http.Flusher).Flush()
			<-r.Context().Done()
		case strings.HasSuffix(r.URL.Path, "/json") && strings.Contains(r.URL.Path, "/containers/"):
			io.WriteString(w, inspect)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	cli, err := client.New(client.WithHost(strings.Replace(server.URL, "http://", "tcp://", 1)))
	require.NoError(t, err)
	t.Cleanup(func() { cli.Close() })

	return &Client{cli: cli, host: container.Host{ID: "local"}}
}

func firstEvent(t *testing.T, c *Client) container.ContainerEvent {
	t.Helper()
	messages := make(chan container.ContainerEvent, 1)
	go c.ContainerEvents(t.Context(), messages)

	select {
	case event := <-messages:
		return event
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for the event")
		return container.ContainerEvent{}
	}
}

const podmanHealthEvent = `{"status":"health_status","id":"abc123def4567890","Type":"container","Action":"health_status","Actor":{"ID":"abc123def4567890","Attributes":{"image":"nginx","name":"web","podId":""}},"scope":"local","time":1700000000,"timeNano":1700000000000000000,"HealthStatus":"healthy"}`

func TestContainerEventsReadsPodmanHealthStatus(t *testing.T) {
	c := podmanServer(t, podmanHealthEvent, `{"Id":"abc123def4567890","State":{"Status":"running","Health":{"Status":"unhealthy","FailingStreak":3}}}`)

	event := firstEvent(t, c)
	assert.Equal(t, "health_status: unhealthy", event.Name)
	assert.Equal(t, "abc123def456", event.ActorID)
	assert.Equal(t, "web", event.ActorAttributes["name"])
}

func TestContainerEventsKeepsBareHealthStatusWithoutHealth(t *testing.T) {
	c := podmanServer(t, podmanHealthEvent, `{"Id":"abc123def4567890","State":{"Status":"running"}}`)

	assert.Equal(t, "health_status", firstEvent(t, c).Name)
}

func TestContainerEventsLeavesDockerHealthStatusAlone(t *testing.T) {
	c := podmanServer(t, `{"Type":"container","Action":"health_status: healthy","Actor":{"ID":"abc123def4567890"}}`, `{}`)

	assert.Equal(t, "health_status: healthy", firstEvent(t, c).Name)
}
