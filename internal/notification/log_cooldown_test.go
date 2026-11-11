package notification

import (
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/notification/dispatcher"
	"github.com/puzpuzpuz/xsync/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/semaphore"
)

func TestProcessLogEvent_cooldownPerRuleAndContainer(t *testing.T) {
	m := &Manager{
		ctx:           t.Context(),
		listener:      NewContainerLogListener(t.Context(), nil),
		subscriptions: xsync.NewMap[int, *Subscription](),
		dispatchers:   xsync.NewMap[int, dispatcher.Dispatcher](),
		sendSem:       semaphore.NewWeighted(8),
	}
	d := &recordingDispatcher{}
	m.dispatchers.Store(1, d)
	for _, id := range []string{"a", "b"} {
		m.listener.cache.Store(id, containerInfo{container: container.Container{ID: id, Name: id}})
	}
	addRule := func(id int) *Subscription {
		sub := &Subscription{
			ID: id, Enabled: true, DispatcherID: 1, Cooldown: 60,
			LogExpression: `message contains "429"`, ContainerExpression: "true",
			EventCooldowns:        xsync.NewMap[string, time.Time](),
			TriggeredContainerIDs: xsync.NewMap[string, struct{}](),
		}
		require.NoError(t, sub.CompileExpressions())
		m.subscriptions.Store(id, sub)
		return sub
	}
	sub := addRule(1)
	send := func(id, message string) {
		m.processLogEvent(&container.LogEvent{ContainerID: id, Message: message})
	}

	send("a", "200 OK")
	assert.Zero(t, sub.EventCooldowns.Size(), "non-matches must not start a cooldown")
	send("a", "429 from client A")
	first, ok := sub.EventCooldowns.Load("a")
	require.True(t, ok)
	send("a", "429 from client B")
	last, _ := sub.EventCooldowns.Load("a")
	assert.Equal(t, first, last, "suppressed matches must not extend the cooldown")
	assert.EqualValues(t, 1, sub.TriggerCount.Load(), "different content is still suppressed")

	send("b", "429 from client A")
	assert.EqualValues(t, 2, sub.TriggerCount.Load(), "container A must not silence container B")
	assert.Equal(t, 2, sub.TriggeredContainersCount())

	other := addRule(2)
	send("a", "429 from client C")
	assert.EqualValues(t, 1, other.TriggerCount.Load(), "rule 1 must not silence rule 2")
	assert.EqualValues(t, 2, sub.TriggerCount.Load())

	sub.EventCooldowns.Store("a", time.Now().Add(-61*time.Second))
	send("a", "429 from client D")
	assert.EqualValues(t, 3, sub.TriggerCount.Load(), "the next match after expiry must fire")
	assert.EqualValues(t, 1, other.TriggerCount.Load())
	require.Eventually(t, func() bool { return len(d.snapshot()) == 4 }, time.Second, time.Millisecond)
}
