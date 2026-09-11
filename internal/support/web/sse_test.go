package support_web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// deadlineRecorder is an httptest.ResponseRecorder that can set write deadlines, the way
// a real *http.response over a net.Conn can.
type deadlineRecorder struct {
	*httptest.ResponseRecorder
	deadlines []time.Time
	err       error
}

func (d *deadlineRecorder) SetWriteDeadline(t time.Time) error {
	if d.err != nil {
		return d.err
	}
	d.deadlines = append(d.deadlines, t)
	return nil
}

func newSSE(t *testing.T, w http.ResponseWriter) *SSEWriter {
	t.Helper()
	r := httptest.NewRequest("GET", "/api/events/stream", nil)
	sse, err := NewSSEWriter(t.Context(), w, r)
	require.NoError(t, err)
	return sse
}

// A client that goes away without closing its socket leaves the write blocking until the
// kernel gives up retransmitting, which is minutes. For that whole window the handler
// stops reading its event channel and the container store drops every event aimed at it.
func TestSSEWriter_setsWriteDeadlinePerWrite(t *testing.T) {
	w := &deadlineRecorder{ResponseRecorder: httptest.NewRecorder()}
	sse := newSSE(t, w)

	// one cleared deadline from the probe in NewSSEWriter
	require.Len(t, w.deadlines, 1)
	assert.True(t, w.deadlines[0].IsZero(), "the probe should not leave a deadline armed")

	before := time.Now()
	require.NoError(t, sse.Event("container-event", map[string]string{"id": "1234"}))
	require.NoError(t, sse.Ping())

	require.Len(t, w.deadlines, 3, "every write should arm its own deadline")
	for _, d := range w.deadlines[1:] {
		assert.WithinRange(t, d, before.Add(writeTimeout), time.Now().Add(writeTimeout))
	}
	assert.Contains(t, w.Body.String(), "container-event")
}

// A ResponseWriter wrapped by middleware may not support deadlines. That is worth a debug
// line, not a broken stream.
func TestSSEWriter_writesWithoutDeadlineSupport(t *testing.T) {
	w := &deadlineRecorder{ResponseRecorder: httptest.NewRecorder(), err: http.ErrNotSupported}
	sse := newSSE(t, w)

	assert.Nil(t, sse.rc, "deadline support should be probed once, not attempted per write")
	require.NoError(t, sse.Event("container-event", map[string]string{"id": "1234"}))
	assert.Contains(t, w.Body.String(), "container-event")
}
