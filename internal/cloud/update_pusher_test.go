package cloud

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	pb "github.com/amir20/dozzle/proto/cloud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sentUpdates collects what a pusher sends, and can fail sends.
type sentUpdates struct {
	mu   sync.Mutex
	got  []*pb.ContainerUpdate
	fail bool
}

func (s *sentUpdates) send(resp *pb.ToolResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fail {
		return errors.New("connection gone")
	}
	s.got = append(s.got, resp.GetContainerUpdate())
	return nil
}

func (s *sentUpdates) names() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []string
	for _, u := range s.got {
		out = append(out, u.GetName())
	}
	return out
}

func record(name, source string, at time.Time) container.UpdateRecord {
	return container.UpdateRecord{
		Host: "nas", Name: name, OldID: "old-" + name, NewID: "new-" + name,
		ImageRef:   "nginx:latest",
		FromDigest: "nginx@sha256:aaa", ToDigest: "nginx@sha256:bbb",
		FromImageID: "sha256:1", ToImageID: "sha256:2",
		At: at, Source: source,
	}
}

func runPusher(t *testing.T, records *container.UpdateRecords, sent *sentUpdates) (stop func()) {
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		pushUpdates(ctx, records, sent.send)
		close(done)
	}()
	return func() {
		cancel()
		<-done
	}
}

// Only what the schedule made, and rollbacks, reach Dozzle Cloud.
func TestPushUpdates_OnlyScheduleAndRollbacks(t *testing.T) {
	at := time.Date(2026, 10, 1, 3, 0, 0, 0, time.UTC)
	records := container.NewUpdateRecords()
	records.Add(record("scheduled", container.UpdateSourceSchedule, at))
	records.Add(record("clicked", container.UpdateSourceDozzle, at.Add(time.Second)))
	records.Add(record("by-cloud", container.UpdateSourceCloud, at.Add(2*time.Second)))
	undone := record("undone", container.UpdateSourceSchedule, at.Add(3*time.Second))
	undone.RolledBack = true
	records.Add(undone)

	sent := &sentUpdates{}
	stop := runPusher(t, records, sent)
	require.Eventually(t, func() bool { return len(sent.names()) == 2 }, time.Second, 5*time.Millisecond)

	// One that lands while connected goes straight out.
	records.Add(record("rolled-back", container.UpdateSourceRollback, at.Add(4*time.Second)))
	require.Eventually(t, func() bool { return len(sent.names()) == 3 }, time.Second, 5*time.Millisecond)
	stop()

	assert.Equal(t, []string{"scheduled", "undone", "rolled-back"}, sent.names())
	first := sent.got[0]
	assert.Equal(t, "sha256:aaa", first.GetFromDigest(), "bare digests, the repository is in the refs")
	assert.Equal(t, "sha256:bbb", first.GetToDigest())
	assert.Equal(t, "new-scheduled", first.GetNewContainerId())
	assert.Equal(t, at.UnixNano(), first.GetAt())
	assert.Equal(t, "nginx:latest", first.GetImageRef())
	assert.True(t, sent.got[1].GetRolledBack())
	assert.Equal(t, container.UpdateSourceRollback, sent.got[2].GetSource())
}

// Each connection replays every kept update: Cloud dedupes what it already has.
func TestPushUpdates_ReplaysOnReconnect(t *testing.T) {
	at := time.Date(2026, 10, 1, 3, 0, 0, 0, time.UTC)
	records := container.NewUpdateRecords()
	records.Add(record("first", container.UpdateSourceSchedule, at))

	sent := &sentUpdates{}
	stop := runPusher(t, records, sent)
	require.Eventually(t, func() bool { return len(sent.names()) == 1 }, time.Second, 5*time.Millisecond)
	stop()

	// Made while the link was down.
	records.Add(record("offline", container.UpdateSourceSchedule, at.Add(time.Minute)))

	again := &sentUpdates{}
	stop = runPusher(t, records, again)
	require.Eventually(t, func() bool { return len(again.names()) == 2 }, time.Second, 5*time.Millisecond)
	stop()
	assert.Equal(t, []string{"first", "offline"}, again.names())
}
