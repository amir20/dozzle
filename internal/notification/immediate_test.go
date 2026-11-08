package notification

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/notification/dispatcher"
	"github.com/amir20/dozzle/types"
	"github.com/puzpuzpuz/xsync/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/semaphore"
)

type recordingDispatcher struct {
	mu   sync.Mutex
	sent []types.Notification
}

func (d *recordingDispatcher) Send(_ context.Context, n types.Notification) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.sent = append(d.sent, n)
	return nil
}

func (d *recordingDispatcher) snapshot() []types.Notification {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]types.Notification(nil), d.sent...)
}

// Log rules only get a cooldown when they are immediate: Dozzle Cloud has to see
// every line of any other rule to judge it.
func TestProcessLogEvent_cooldownAppliesOnlyToImmediateRules(t *testing.T) {
	for _, tc := range []struct {
		name      string
		immediate bool
		cooldown  int
		want      int
	}{
		{"immediate with a cooldown sends once", true, 60, 1},
		{"immediate with no cooldown sends every match", true, 0, 3},
		{"normal rule ignores a cooldown", false, 60, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			c := container.Container{ID: "abc", Name: "app"}
			d := &recordingDispatcher{}
			m := &Manager{
				ctx:           ctx,
				listener:      NewContainerLogListener(ctx, []container.ClientService{&shortLivedClient{c: c}}),
				subscriptions: xsync.NewMap[int, *Subscription](),
				dispatchers:   xsync.NewMap[int, dispatcher.Dispatcher](),
				sendSem:       semaphore.NewWeighted(8),
			}
			m.dispatchers.Store(1, d)

			sub := &Subscription{
				ID: 1, Enabled: true, DispatcherID: 1,
				LogExpression: "true", ContainerExpression: "true",
				Immediate: tc.immediate, Cooldown: tc.cooldown,
				EventCooldowns:        xsync.NewMap[string, time.Time](),
				TriggeredContainerIDs: xsync.NewMap[string, struct{}](),
			}
			require.NoError(t, sub.CompileExpressions())
			m.subscriptions.Store(1, sub)

			for range 3 {
				m.processLogEvent(&container.LogEvent{ContainerID: c.ID, Message: "hello"})
			}

			require.Eventually(t, func() bool { return len(d.snapshot()) == tc.want }, time.Second, time.Millisecond)
			time.Sleep(50 * time.Millisecond) // no late extra sends
			sent := d.snapshot()
			require.Len(t, sent, tc.want)
			assert.Equal(t, tc.immediate, sent[0].Subscription.Immediate, "Cloud reads this off the event")
		})
	}
}
