package notification

import (
	"bytes"
	"testing"

	"github.com/amir20/dozzle/internal/notification/dispatcher"
	"github.com/puzpuzpuz/xsync/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteConfigKeepsLongExpressionsOnOneLine(t *testing.T) {
	m := &Manager{
		subscriptions: xsync.NewMap[int, *Subscription](),
		dispatchers:   xsync.NewMap[int, dispatcher.Dispatcher](),
	}
	expr := `level == "error" && message contains "connection refused" && message contains "upstream" && name startsWith "api"`
	m.subscriptions.Store(1, &Subscription{ID: 1, Name: "long", LogExpression: expr})

	var buf bytes.Buffer
	require.NoError(t, m.WriteConfig(&buf))

	assert.Contains(t, buf.String(), "logExpression: "+expr+"\n")
}

// A rule that no longer compiles must not cost the user their webhooks, or itself, on the next save.
func TestLoadConfigKeepsDispatchersAndInvalidRule(t *testing.T) {
	ctx := t.Context()
	m := &Manager{
		ctx:           ctx,
		listener:      NewContainerLogListener(ctx, nil),
		statsListener: NewContainerStatsListener(ctx, nil),
		eventListener: NewContainerEventListener(ctx, nil),
		subscriptions: xsync.NewMap[int, *Subscription](),
		dispatchers:   xsync.NewMap[int, dispatcher.Dispatcher](),
	}

	in := `subscriptions:
  - id: 1
    name: broken
    enabled: true
    dispatcherId: 1
    logExpression: 'level == == "error"'
    containerExpression: 'name == "api"'
dispatchers:
  - id: 1
    name: hook
    type: webhook
    url: https://example.com/hook
`
	require.NoError(t, m.LoadConfig(bytes.NewBufferString(in)))

	require.Len(t, m.Dispatchers(), 1)
	subs := m.Subscriptions()
	require.Len(t, subs, 1)
	assert.False(t, subs[0].IsLogAlert())
	assert.Nil(t, subs[0].ContainerProgram)

	var buf bytes.Buffer
	require.NoError(t, m.WriteConfig(&buf))
	assert.Contains(t, buf.String(), "https://example.com/hook")
	assert.Contains(t, buf.String(), `level == == "error"`)
}
