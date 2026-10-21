package notification

import (
	"context"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/types"
	"github.com/expr-lang/expr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeEvent(t *testing.T) {
	tests := []struct {
		name           string
		event          container.ContainerEvent
		wantName       string
		wantAttributes map[string]string
	}{
		{
			name:           "unhealthy health event is normalized",
			event:          container.ContainerEvent{Name: "health_status: unhealthy"},
			wantName:       "health_status",
			wantAttributes: map[string]string{"healthStatus": "unhealthy"},
		},
		{
			name:           "healthy health event is normalized",
			event:          container.ContainerEvent{Name: "health_status: healthy"},
			wantName:       "health_status",
			wantAttributes: map[string]string{"healthStatus": "healthy"},
		},
		{
			name: "existing attributes are preserved",
			event: container.ContainerEvent{
				Name:            "health_status: unhealthy",
				ActorAttributes: map[string]string{"name": "postgres"},
			},
			wantName:       "health_status",
			wantAttributes: map[string]string{"name": "postgres", "healthStatus": "unhealthy"},
		},
		{
			name:           "non-health events are left untouched",
			event:          container.ContainerEvent{Name: "die", ActorAttributes: map[string]string{"exitCode": "1"}},
			wantName:       "die",
			wantAttributes: map[string]string{"exitCode": "1"},
		},
		{
			name:           "bare health_status name is left untouched",
			event:          container.ContainerEvent{Name: "health_status"},
			wantName:       "health_status",
			wantAttributes: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := tt.event
			normalizeEvent(&event)
			assert.Equal(t, tt.wantName, event.Name)
			assert.Equal(t, tt.wantAttributes, event.ActorAttributes)
		})
	}
}

// Ensures the documented health alert expression actually matches a normalized event.
func TestNormalizedHealthEventMatchesSubscription(t *testing.T) {
	event := container.ContainerEvent{Name: "health_status: unhealthy"}
	normalizeEvent(&event)

	require.True(t, allowedEventNames[event.Name], "normalized event must pass the allowlist")

	sub := &Subscription{
		EventExpression: `name == "health_status" && attributes["healthStatus"] == "unhealthy"`,
	}
	program, err := expr.Compile(sub.EventExpression, expr.Env(types.NotificationEvent{}))
	require.NoError(t, err)
	sub.EventProgram = program

	notificationEvent := types.NotificationEvent{
		Name:       event.Name,
		Attributes: event.ActorAttributes,
	}
	assert.True(t, sub.MatchesEvent(notificationEvent))

	healthy := container.ContainerEvent{Name: "health_status: healthy"}
	normalizeEvent(&healthy)
	assert.False(t, sub.MatchesEvent(types.NotificationEvent{
		Name:       healthy.Name,
		Attributes: healthy.ActorAttributes,
	}))
}

// updateHostClient is a host an update ran on, possibly an agent the
// listener's own clients do not include.
type updateHostClient struct {
	container.ClientService
	host string
	c    container.Container
}

func (u *updateHostClient) Host(context.Context) (container.Host, error) {
	return container.Host{ID: u.host}, nil
}

func (u *updateHostClient) FindContainer(_ context.Context, id string, _ container.ContainerLabels) (container.Container, error) {
	if id != u.c.ID {
		return container.Container{}, container.ErrContainerNotFound
	}
	return u.c, nil
}

func TestUpdateRecordRaisesUpdateEvent(t *testing.T) {
	records := container.NewUpdateRecords()
	other := &updateHostClient{host: "local", c: container.Container{ID: "aaaaaaaaaaaa"}}
	agent := &updateHostClient{host: "agent-1", c: container.Container{ID: "bbbbbbbbbbbb", Name: "web"}}

	l := NewContainerEventListener(t.Context(), nil)
	l.ListenToUpdates(records, func() []container.ClientService { return []container.ClientService{other, agent} })
	l.Start()
	defer l.Stop()

	records.Add(container.UpdateRecord{
		Host:       "agent-1",
		Name:       "web",
		OldID:      "cccccccccccc",
		NewID:      "bbbbbbbbbbbb",
		ImageRef:   "nginx:latest",
		FromDigest: "nginx@sha256:1111111111111111",
		ToDigest:   "nginx@sha256:2222222222222222",
		Source:     container.UpdateSourceSchedule,
		At:         time.Now(),
	})

	select {
	case entry := <-l.Channel():
		assert.Equal(t, UpdateEventName, entry.Event.Name)
		assert.Equal(t, "bbbbbbbbbbbb", entry.Event.ActorID)
		assert.Equal(t, "web", entry.Container.Name)
		assert.Equal(t, "agent-1", entry.Host.ID)
		assert.Equal(t, map[string]string{
			"source":     "schedule",
			"rolledBack": "false",
			"image":      "nginx:latest",
			"fromDigest": "nginx@sha256:1111111111111111",
			"toDigest":   "nginx@sha256:2222222222222222",
		}, entry.Event.ActorAttributes)
	case <-time.After(time.Second):
		t.Fatal("no update event")
	}
}

func TestUpdateRecordOnUnknownHostIsDropped(t *testing.T) {
	records := container.NewUpdateRecords()
	local := &updateHostClient{host: "local", c: container.Container{ID: "bbbbbbbbbbbb"}}

	l := NewContainerEventListener(t.Context(), nil)
	l.ListenToUpdates(records, func() []container.ClientService { return []container.ClientService{local} })
	l.Start()
	defer l.Stop()

	records.Add(container.UpdateRecord{Host: "gone", NewID: "bbbbbbbbbbbb", At: time.Now()})

	select {
	case entry := <-l.Channel():
		t.Fatalf("unexpected event %+v", entry.Event)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestDescribeUpdate(t *testing.T) {
	assert.Equal(t, "Container updated: nginx:latest (111111111111 → 222222222222)", describeUpdate(map[string]string{
		"image":      "nginx:latest",
		"fromDigest": "nginx@sha256:1111111111111111",
		"toDigest":   "sha256:2222222222222222",
		"rolledBack": "false",
	}))
	assert.Equal(t, "Container update rolled back: app:dev", describeUpdate(map[string]string{
		"image":      "app:dev",
		"rolledBack": "true",
	}))
}

// The starter rule the Cloud welcome modal offers matches an update event.
func TestUpdateEventMatchesStarterRule(t *testing.T) {
	program, err := expr.Compile(`name == "update"`, expr.Env(types.NotificationEvent{}))
	require.NoError(t, err)
	sub := &Subscription{EventExpression: `name == "update"`, EventProgram: program}

	event := updateEvent(container.UpdateRecord{Source: container.UpdateSourceDozzle})
	assert.True(t, sub.MatchesEvent(types.NotificationEvent{Name: event.Name, Attributes: event.ActorAttributes}))
}
