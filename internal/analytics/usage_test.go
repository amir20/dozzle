package analytics

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/amir20/dozzle/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuckets(t *testing.T) {
	for n, want := range map[int]string{0: "0", -3: "0", 1: "1-10", 10: "1-10", 11: "11-60", 60: "11-60", 61: "61-240", 240: "61-240", 241: "241+"} {
		assert.Equal(t, want, BucketMinutes(n), n)
	}
	for n, want := range map[int]string{0: "", 1: "1", 2: "2-5", 5: "2-5", 6: "6-20", 20: "6-20", 21: "21+"} {
		assert.Equal(t, want, BucketUsers(n), n)
	}
}

func TestUsageIgnoresUnknownKeysAndBadLocales(t *testing.T) {
	u := NewUsage()
	u.Add("view.container", 2)
	u.Add("container.my-secret-app", 1)
	u.Add("view.host", -4)
	u.AddLocale("en")
	u.AddLocale("zh-TW")
	u.AddLocale("../etc/passwd")
	u.AddLocale("")

	s := u.Take()
	assert.Equal(t, map[string]int{"view.container": 2}, s.Counts)
	assert.Equal(t, map[string]int{"en": 1, "zh-TW": 1}, s.Locales)
}

// A browser key missing from UsageKeys is accepted by POST /api/usage and then
// dropped by Add, so it would never reach the beacon and nothing would say so.
func TestBrowserUsageKeysAreCounted(t *testing.T) {
	u := NewUsage()
	want := map[string]int{}
	for key := range BrowserUsageKeys {
		u.Add(key, 1)
		want[key] = 1
	}
	assert.Equal(t, want, u.Take().Counts)
}

func TestUsageLocalesAreBounded(t *testing.T) {
	u := NewUsage()
	for i := range 100 {
		u.AddLocale(fmt.Sprintf("x%c", 'a'+rune(i%26)) + "-" + fmt.Sprintf("%c%c", 'A'+rune(i%26), 'A'+rune(i/26)))
	}
	assert.LessOrEqual(t, len(u.Take().Locales), maxLocales)
}

func TestTakeResets(t *testing.T) {
	u := NewUsage()
	u.Add("logs.sql", 3)
	u.AddMinutes(12)
	first := u.Take()
	assert.Equal(t, 3, first.Counts["logs.sql"])
	assert.Equal(t, 12, first.Minutes)
	assert.True(t, u.Take().Empty())
}

func TestDisabledCountsNothing(t *testing.T) {
	u := NewUsage()
	u.Disable()
	u.Add("view.container", 1)
	u.AddLocale("en")
	u.AddMinutes(5)
	sent := false
	assert.False(t, SendUsage(u, func() types.BeaconEvent { return types.BeaconEvent{} }, func(types.BeaconEvent) error { sent = true; return nil }))
	assert.False(t, sent)
}

func TestSendUsage(t *testing.T) {
	u := NewUsage()
	base := func() types.BeaconEvent { return types.BeaconEvent{ServerID: "abc", Version: "v1"} }

	var got []types.BeaconEvent
	send := func(b types.BeaconEvent) error { got = append(got, b); return nil }

	// Nothing counted, nothing sent.
	assert.False(t, SendUsage(u, base, send))
	assert.Empty(t, got)

	u.Add("view.merged", 4)
	u.AddLocale("de")
	u.AddMinutes(30)
	require.True(t, SendUsage(u, base, send))
	require.Len(t, got, 1)
	assert.Equal(t, "usage", got[0].Name)
	assert.Equal(t, "abc", got[0].ServerID)
	assert.Equal(t, map[string]int{"view.merged": 4}, got[0].Usage)
	assert.Equal(t, map[string]int{"de": 1}, got[0].Locales)
	assert.Equal(t, "11-60", got[0].ActiveMinutes)

	// Sent means reset.
	assert.False(t, SendUsage(u, base, send))
}

func TestSendUsageRestoresOnFailure(t *testing.T) {
	u := NewUsage()
	u.Add("action.restart", 2)
	u.AddLocale("fr")
	assert.False(t, SendUsage(u, func() types.BeaconEvent { return types.BeaconEvent{} }, func(types.BeaconEvent) error { return errors.New("offline") }))

	s := u.Take()
	assert.Equal(t, 2, s.Counts["action.restart"])
	assert.Equal(t, 1, s.Locales["fr"])
}

func TestRunUsageBeaconStopsWithContext(t *testing.T) {
	u := NewUsage()
	u.Add("view.host", 1)
	ctx, cancel := context.WithCancel(context.Background())
	sent := make(chan types.BeaconEvent, 4)
	done := make(chan struct{})
	go func() {
		RunUsageBeacon(ctx, u, 10*time.Millisecond, func() types.BeaconEvent { return types.BeaconEvent{} }, func(b types.BeaconEvent) error { sent <- b; return nil })
		close(done)
	}()
	select {
	case b := <-sent:
		assert.Equal(t, 1, b.Usage["view.host"])
	case <-time.After(time.Second):
		t.Fatal("no usage beacon sent")
	}
	cancel()
	<-done
}
