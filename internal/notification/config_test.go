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
