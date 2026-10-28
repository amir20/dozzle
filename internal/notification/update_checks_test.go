package notification

import (
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/imagecheck"
	"github.com/amir20/dozzle/types"
	"github.com/expr-lang/expr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNextUpdateCheck(t *testing.T) {
	now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	assert.Equal(t, time.Date(2026, 10, 28, 15, 30, 0, 0, time.UTC), nextUpdateCheck(now, 15*time.Hour+30*time.Minute))
	assert.Equal(t, time.Date(2026, 10, 29, 3, 0, 0, 0, time.UTC), nextUpdateCheck(now, 3*time.Hour))
	// Exactly now has already run.
	assert.Equal(t, time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC), nextUpdateCheck(now, 12*time.Hour))
}

func availableUpdate(name, remote string) container.ImageUpdate {
	return container.ImageUpdate{
		Container: container.Container{ID: name + "-id", Name: name, Host: "agent-1", Image: "nginx:latest"},
		Result: imagecheck.Result{
			Status:         imagecheck.StatusUpdateAvailable,
			LocalDigest:    "sha256:1111111111111111",
			RemoteDigest:   remote,
			CurrentVersion: "1.27.0",
			LatestVersion:  "1.27.1",
		},
	}
}

func drain(l *ContainerEventListener) []*ContainerEventEntry {
	var entries []*ContainerEventEntry
	for {
		select {
		case e := <-l.Channel():
			entries = append(entries, e)
		default:
			return entries
		}
	}
}

func TestAvailableUpdateRaisedOncePerDigest(t *testing.T) {
	agent := &updateHostClient{host: "agent-1"}
	l := NewContainerEventListener(t.Context(), nil)
	l.ListenToUpdates(container.NewUpdateRecords(), func() []container.ClientService { return []container.ClientService{agent} })

	upToDate := availableUpdate("db", "")
	upToDate.Result.Status = imagecheck.StatusUpToDate
	l.raiseAvailableUpdates(t.Context(), []container.ImageUpdate{availableUpdate("web", "sha256:2222222222222222"), upToDate})

	entries := drain(l)
	require.Len(t, entries, 1)
	assert.Equal(t, UpdateAvailableEventName, entries[0].Event.Name)
	assert.Equal(t, "web-id", entries[0].Event.ActorID)
	assert.Equal(t, "agent-1", entries[0].Host.ID)
	assert.Equal(t, map[string]string{
		"image":          "nginx:latest",
		"currentVersion": "1.27.0",
		"latestVersion":  "1.27.1",
		"currentDigest":  "sha256:1111111111111111",
		"latestDigest":   "sha256:2222222222222222",
	}, entries[0].Event.ActorAttributes)

	// The same release the next day is not raised again, even on a new container id.
	again := availableUpdate("web", "sha256:2222222222222222")
	again.Container.ID = "recreated"
	l.raiseAvailableUpdates(t.Context(), []container.ImageUpdate{again})
	assert.Empty(t, drain(l))

	// A newer release is.
	l.raiseAvailableUpdates(t.Context(), []container.ImageUpdate{availableUpdate("web", "sha256:3333333333333333")})
	assert.Len(t, drain(l), 1)
}

func TestAvailableUpdateOnUnknownHostIsDropped(t *testing.T) {
	local := &updateHostClient{host: "local"}
	l := NewContainerEventListener(t.Context(), nil)
	l.ListenToUpdates(container.NewUpdateRecords(), func() []container.ClientService { return []container.ClientService{local} })

	l.raiseAvailableUpdates(t.Context(), []container.ImageUpdate{availableUpdate("web", "sha256:2222222222222222")})
	assert.Empty(t, drain(l))
}

func TestDescribeUpdateAvailable(t *testing.T) {
	assert.Equal(t, "Update available: nginx:latest (1.27.0 → 1.27.1)", describeUpdateAvailable(map[string]string{
		"image":          "nginx:latest",
		"currentVersion": "1.27.0",
		"latestVersion":  "1.27.1",
		"currentDigest":  "sha256:1111111111111111",
		"latestDigest":   "sha256:2222222222222222",
	}))
	assert.Equal(t, "Update available: app:dev (111111111111 → 222222222222)", describeUpdateAvailable(map[string]string{
		"image":         "app:dev",
		"latestVersion": "2.0",
		"currentDigest": "sha256:1111111111111111",
		"latestDigest":  "sha256:2222222222222222",
	}))
}

func TestUpdateAvailableEventMatchesExample(t *testing.T) {
	program, err := expr.Compile(`name == "update_available"`, expr.Env(types.NotificationEvent{}))
	require.NoError(t, err)
	sub := &Subscription{EventProgram: program}

	event := updateAvailableEvent(availableUpdate("web", "sha256:2222222222222222"))
	assert.True(t, sub.MatchesEvent(types.NotificationEvent{Name: event.Name, Attributes: event.ActorAttributes}))
}
