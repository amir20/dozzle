package notification

import (
	"testing"
	"time"

	"github.com/amir20/dozzle/types"
	"github.com/puzpuzpuz/xsync/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func logCooldownSub(cooldown int) *Subscription {
	return &Subscription{Cooldown: cooldown, LogCooldowns: xsync.NewMap[string, *logWindow]()}
}

func containerFixture(id string) types.NotificationContainer {
	return types.NotificationContainer{ID: id, Name: "app-" + id}
}

func logFixture(message string) types.NotificationLog {
	return types.NotificationLog{Message: message, Level: "error"}
}

func TestRecordLogMatch_FirstSightingDispatches(t *testing.T) {
	sub := logCooldownSub(300)
	c := containerFixture("c1")

	assert.True(t, sub.RecordLogMatch(c, logFixture("connection refused")))
}

func TestRecordLogMatch_RepeatIsSuppressed(t *testing.T) {
	sub := logCooldownSub(300)
	c := containerFixture("c1")

	require.True(t, sub.RecordLogMatch(c, logFixture("connection refused")))
	assert.False(t, sub.RecordLogMatch(c, logFixture("connection refused")))
	assert.False(t, sub.RecordLogMatch(c, logFixture("connection refused")))
}

func TestRecordLogMatch_VaryingTokensCollapseToOnePattern(t *testing.T) {
	// Same failure, different IP each time. This is the case the cooldown exists
	// for: it is one incident, not three.
	sub := logCooldownSub(300)
	c := containerFixture("c1")

	require.True(t, sub.RecordLogMatch(c, logFixture("Connection refused to database at 10.0.0.5:5432")))
	assert.False(t, sub.RecordLogMatch(c, logFixture("Connection refused to database at 192.168.1.1:5432")))
}

func TestRecordLogMatch_DifferentErrorSameContainerStillDispatches(t *testing.T) {
	// The property that makes a pattern-keyed cooldown safe. A per-container
	// cooldown would swallow the second line and Cloud would never learn the
	// app fell over.
	sub := logCooldownSub(300)
	c := containerFixture("c1")

	require.True(t, sub.RecordLogMatch(c, logFixture("FATAL: could not extend file: No space left on device")))
	assert.True(t, sub.RecordLogMatch(c, logFixture("FATAL: connection refused")))
}

func TestRecordLogMatch_SamePatternDifferentContainersBothDispatch(t *testing.T) {
	sub := logCooldownSub(300)

	require.True(t, sub.RecordLogMatch(containerFixture("c1"), logFixture("connection refused")))
	assert.True(t, sub.RecordLogMatch(containerFixture("c2"), logFixture("connection refused")))
}

func TestRecordLogMatch_NoCooldownAlwaysDispatches(t *testing.T) {
	// Existing rules all carry cooldown 0 and must keep behaving exactly as before.
	sub := logCooldownSub(0)
	c := containerFixture("c1")

	assert.True(t, sub.RecordLogMatch(c, logFixture("connection refused")))
	assert.True(t, sub.RecordLogMatch(c, logFixture("connection refused")))
}

func TestRecordLogMatch_EmptyMessageDispatches(t *testing.T) {
	sub := logCooldownSub(300)
	c := containerFixture("c1")

	assert.True(t, sub.RecordLogMatch(c, logFixture("")))
	assert.True(t, sub.RecordLogMatch(c, logFixture("")))
}

func TestRecordLogMatch_FailsOpenPastPatternLimit(t *testing.T) {
	sub := logCooldownSub(300)
	c := containerFixture("c1")

	for i := range maxTrackedLogPatterns {
		require.True(t, sub.RecordLogMatch(c, logFixture("unique failure "+lettersFor(i))))
	}

	// Map is full: a new pattern is dispatched rather than tracked, and repeats
	// of it are dispatched too, because we would not be able to account for them.
	assert.True(t, sub.RecordLogMatch(c, logFixture("brand new failure mode")))
	assert.True(t, sub.RecordLogMatch(c, logFixture("brand new failure mode")))
	assert.Equal(t, maxTrackedLogPatterns, sub.LogCooldowns.Size())
}

func TestCollectLogRollups_NothingWhileWindowOpen(t *testing.T) {
	sub := logCooldownSub(300)
	c := containerFixture("c1")

	require.True(t, sub.RecordLogMatch(c, logFixture("connection refused")))
	sub.RecordLogMatch(c, logFixture("connection refused"))

	assert.Empty(t, sub.CollectLogRollups())
}

func TestCollectLogRollups_EmitsCountWhenWindowCloses(t *testing.T) {
	sub := logCooldownSub(1)
	c := containerFixture("c1")

	require.True(t, sub.RecordLogMatch(c, logFixture("connection refused")))
	for range 4 {
		sub.RecordLogMatch(c, logFixture("connection refused"))
	}

	expireLogWindows(sub)

	rollups := sub.CollectLogRollups()
	require.Len(t, rollups, 1)
	assert.Equal(t, int64(4), rollups[0].Count)
	assert.Equal(t, "c1", rollups[0].Container.ID)
	assert.Equal(t, "connection refused", rollups[0].Log.Message)
}

func TestCollectLogRollups_QuietWindowIsDroppedAndPatternIsNewsAgain(t *testing.T) {
	sub := logCooldownSub(1)
	c := containerFixture("c1")

	require.True(t, sub.RecordLogMatch(c, logFixture("connection refused")))
	expireLogWindows(sub)

	assert.Empty(t, sub.CollectLogRollups())
	assert.Zero(t, sub.LogCooldowns.Size())
	assert.True(t, sub.RecordLogMatch(c, logFixture("connection refused")))
}

func TestCollectLogRollups_OngoingPatternCostsOneNotificationPerWindow(t *testing.T) {
	sub := logCooldownSub(1)
	c := containerFixture("c1")

	require.True(t, sub.RecordLogMatch(c, logFixture("connection refused")))
	sub.RecordLogMatch(c, logFixture("connection refused"))
	expireLogWindows(sub)
	require.Len(t, sub.CollectLogRollups(), 1)

	// Window was reset, not deleted: still suppressing, and the next close
	// reports only the repeats seen since.
	assert.False(t, sub.RecordLogMatch(c, logFixture("connection refused")))
	expireLogWindows(sub)
	rollups := sub.CollectLogRollups()
	require.Len(t, rollups, 1)
	assert.Equal(t, int64(1), rollups[0].Count)
}

func TestCollectLogRollups_CapFlushesBeforeWindowCloses(t *testing.T) {
	sub := logCooldownSub(3600)
	c := containerFixture("c1")

	require.True(t, sub.RecordLogMatch(c, logFixture("connection refused")))
	for range maxSuppressedPerWindow {
		sub.RecordLogMatch(c, logFixture("connection refused"))
	}

	rollups := sub.CollectLogRollups()
	require.Len(t, rollups, 1)
	assert.Equal(t, int64(maxSuppressedPerWindow), rollups[0].Count)
}

// lettersFor builds a distinct all-letter token, so pattern extraction keeps it
// verbatim instead of shaping digits into a placeholder and collapsing them.
func lettersFor(n int) string {
	var b []byte
	for {
		b = append([]byte{byte('a' + n%26)}, b...)
		n /= 26
		if n == 0 {
			return string(b)
		}
		n--
	}
}

// expireLogWindows backdates every open window so the next collection treats it
// as closed, without making the test wait out a real cooldown.
func expireLogWindows(sub *Subscription) {
	sub.LogCooldowns.Range(func(_ string, w *logWindow) bool {
		w.mu.Lock()
		w.openedAt = w.openedAt.Add(-time.Hour)
		w.mu.Unlock()
		return true
	})
}
