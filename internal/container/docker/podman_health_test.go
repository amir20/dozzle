package docker

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/moby/moby/api/types/events"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDockerHealthShape(t *testing.T) {
	tests := []struct {
		name   string
		record string
		want   string
	}{
		{
			name:   "podman health event gains docker's action",
			record: `{"Type":"container","Action":"health_status","Actor":{"ID":"abc","Attributes":{"name":"web"}},"HealthStatus":"healthy"}`,
			want:   "health_status: healthy",
		},
		{
			name:   "podman unhealthy",
			record: `{"Type":"container","Action":"health_status","Actor":{"ID":"abc"},"HealthStatus":"unhealthy"}`,
			want:   "health_status: unhealthy",
		},
		{
			name:   "podman starting",
			record: `{"Type":"container","Action":"health_status","Actor":{"ID":"abc"},"HealthStatus":"starting"}`,
			want:   "health_status: starting",
		},
		{
			name:   "docker health event is left alone",
			record: `{"Type":"container","Action":"health_status: healthy","Actor":{"ID":"abc"}}`,
			want:   "health_status: healthy",
		},
		{
			name:   "bare health_status with no status stays bare",
			record: `{"Type":"container","Action":"health_status","Actor":{"ID":"abc"}}`,
			want:   "health_status",
		},
		{
			name:   "other events are left alone",
			record: `{"Type":"container","Action":"die","Actor":{"ID":"abc"},"HealthStatus":"healthy"}`,
			want:   "die",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var message events.Message
			require.NoError(t, json.Unmarshal(dockerHealthShape(json.RawMessage(tt.record)), &message))
			assert.Equal(t, tt.want, string(message.Action))
		})
	}
}

// The rest of the event has to survive the round trip, or a rewritten health
// event would arrive without the container it belongs to.
func TestDockerHealthShapeKeepsTheRestOfTheEvent(t *testing.T) {
	record := `{"Type":"container","Action":"health_status","Actor":{"ID":"abc123","Attributes":{"name":"web","image":"nginx"}},"scope":"local","time":1700000000,"timeNano":1700000000000000000,"HealthStatus":"healthy"}`

	var message events.Message
	require.NoError(t, json.Unmarshal(dockerHealthShape(json.RawMessage(record)), &message))

	assert.Equal(t, "health_status: healthy", string(message.Action))
	assert.Equal(t, events.ContainerEventType, message.Type)
	assert.Equal(t, "abc123", message.Actor.ID)
	assert.Equal(t, map[string]string{"name": "web", "image": "nginx"}, message.Actor.Attributes)
	assert.Equal(t, int64(1700000000), message.Time)
	assert.Equal(t, int64(1700000000000000000), message.TimeNano)
}

func TestRewriteEvents(t *testing.T) {
	stream := strings.Join([]string{
		`{"Type":"container","Action":"start","Actor":{"ID":"abc"}}`,
		`{"Type":"container","Action":"health_status","Actor":{"ID":"abc"},"HealthStatus":"healthy"}`,
		`{"Type":"container","Action":"die","Actor":{"ID":"abc"}}`,
	}, "\n")

	body := rewriteEvents(io.NopCloser(strings.NewReader(stream)))
	defer body.Close()

	var actions []string
	decoder := json.NewDecoder(body)
	for {
		var message events.Message
		if err := decoder.Decode(&message); err != nil {
			break
		}
		actions = append(actions, string(message.Action))
	}

	assert.Equal(t, []string{"start", "health_status: healthy", "die"}, actions)
}

func TestHealthEventCompatLeavesOtherResponsesAlone(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		status      int
		contentType string
	}{
		{name: "another endpoint", path: "/v1.51/containers/json", status: http.StatusOK, contentType: "application/json"},
		{name: "an error", path: "/v1.51/events", status: http.StatusInternalServerError, contentType: "application/json"},
		{name: "a json sequence", path: "/v1.51/events", status: http.StatusOK, contentType: "application/json-seq"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := &http.Response{
				Request:    httptest.NewRequest(http.MethodGet, tt.path, nil),
				StatusCode: tt.status,
				Header:     http.Header{"Content-Type": []string{tt.contentType}},
				Body:       io.NopCloser(strings.NewReader("{}")),
			}

			healthEventCompat(response)
			_, rewritten := response.Body.(*rewrittenBody)
			assert.False(t, rewritten, "body should have been left alone")
		})
	}
}

// What the fix is actually for: a real client reading a real Podman events
// stream should see the health transition Docker would have reported.
func TestClientEventsReadsPodmanHealthStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/events") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// byte for byte what Podman 5.x writes for a passing health check
		io.WriteString(w, `{"status":"health_status","id":"abc123","Type":"container","Action":"health_status","Actor":{"ID":"abc123","Attributes":{"image":"nginx","name":"web","podId":""}},"scope":"local","time":1700000000,"timeNano":1700000000000000000,"HealthStatus":"healthy"}`+"\n")
		w.(http.Flusher).Flush()
	}))
	defer server.Close()

	cli, err := client.New(
		client.WithHost(strings.Replace(server.URL, "http://", "tcp://", 1)),
		client.WithResponseHook(healthEventCompat),
	)
	require.NoError(t, err)
	defer cli.Close()

	result := cli.Events(t.Context(), client.EventsListOptions{})

	select {
	case message := <-result.Messages:
		assert.Equal(t, "health_status: healthy", string(message.Action))
		assert.Equal(t, "abc123", message.Actor.ID)
	case err := <-result.Err:
		t.Fatalf("events stream failed: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for the event")
	}
}
