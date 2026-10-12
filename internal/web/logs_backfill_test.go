package web

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeLogs serves LogsBetweenDates from a fixed set of lines per container and
// counts the reads, which is the cost the search is trying to keep down.
type fakeLogs struct {
	container.ClientService
	mu    sync.Mutex
	lines map[string][]*container.LogEvent
	reads int
	// ignoreEnd reads to the end of the log whatever `to` says, as k8s does.
	ignoreEnd bool
	// failing names containers whose read errors, as an unreachable agent's does.
	failing map[string]bool
}

func (f *fakeLogs) LogsBetweenDates(_ context.Context, c container.Container, from time.Time, to time.Time, _ container.StdType) (<-chan *container.LogEvent, error) {
	f.mu.Lock()
	f.reads++
	f.mu.Unlock()
	if f.failing[c.ID] {
		return nil, errors.New("agent unreachable")
	}
	ch := make(chan *container.LogEvent, len(f.lines[c.ID]))
	for _, e := range f.lines[c.ID] {
		ts := time.UnixMilli(e.Timestamp)
		if !ts.Before(from) && (f.ignoreEnd || ts.Before(to)) {
			ch <- e
		}
	}
	close(ch)
	return ch, nil
}

// lines returns n lines spread evenly over [from, to), with the ones whose
// index is in needles at level error and the rest at info.
func lines(from, to time.Time, n int, needle func(i int) bool) []*container.LogEvent {
	step := to.Sub(from) / time.Duration(n)
	events := make([]*container.LogEvent, n)
	for i := range events {
		level := "info"
		if needle(i) {
			level = "error"
		}
		events[i] = &container.LogEvent{Id: uint32(i), Timestamp: from.Add(step * time.Duration(i)).UnixMilli(), Level: level}
	}
	return events
}

type backfillResult struct {
	batches  [][]*container.LogEvent
	progress []searchStatus
	final    searchStatus
}

func (r backfillResult) matches() []*container.LogEvent {
	var all []*container.LogEvent
	for _, b := range r.batches {
		all = append(all, b...)
	}
	return all
}

func runBackfill(t *testing.T, fake *fakeLogs, containers []container.Container, to time.Time) backfillResult {
	t.Helper()
	services := make([]*container.ContainerService, len(containers))
	for i, c := range containers {
		services[i] = container.NewContainerService(fake, c)
	}
	filter := logFilter{levels: map[string]struct{}{"error": {}}}
	backfill := make(chan []*container.LogEvent)
	status := make(chan searchStatus)
	go searchBackfill(context.Background(), services, to, time.Time{}, container.STDALL, filter, backfill, status)

	var r backfillResult
	timeout := time.After(5 * time.Second)
	for {
		select {
		case b := <-backfill:
			r.batches = append(r.batches, b)
		case s := <-status:
			if s.Done {
				r.final = s
				return r
			}
			r.progress = append(r.progress, s)
		case <-timeout:
			require.FailNow(t, "search never finished")
		}
	}
}

func Test_searchBackfill_sparse_reads_twice(t *testing.T) {
	to := time.Now()
	created := to.Add(-30 * 24 * time.Hour)
	fake := &fakeLogs{lines: map[string][]*container.LogEvent{
		"a": lines(created, to, 10000, func(i int) bool { return i%3000 == 7 }),
	}}

	r := runBackfill(t, fake, []container.Container{{ID: "a", Created: created}}, to)

	// a walk of doubling windows over 30 days took 18 reads
	assert.Equal(t, 2, fake.reads)
	assert.Len(t, r.matches(), 4)
	assert.Equal(t, "exhausted", r.final.Reason)
	assert.Equal(t, 4, r.final.Matches)
	assert.True(t, r.final.ScannedTo.Equal(created))
}

func Test_searchBackfill_dense_stops_after_first_window(t *testing.T) {
	to := time.Now()
	created := to.Add(-time.Hour)
	fake := &fakeLogs{lines: map[string][]*container.LogEvent{
		"a": lines(to.Add(-9*time.Second), to, 60, func(int) bool { return true }),
	}}

	r := runBackfill(t, fake, []container.Container{{ID: "a", Created: created}}, to)

	assert.Equal(t, 1, fake.reads)
	assert.Equal(t, "capped", r.final.Reason)
	// 10 older matches were dropped, so the results only reach the oldest kept
	got := r.matches()
	require.Len(t, got, backfillMatches)
	assert.True(t, r.final.ScannedTo.Equal(time.UnixMilli(got[0].Timestamp)))
}

func Test_searchBackfill_long_read_keeps_newest_matches(t *testing.T) {
	to := time.Now()
	created := to.Add(-24 * time.Hour)
	all := lines(created, to.Add(-time.Minute), 1000, func(i int) bool { return i%5 == 0 })
	fake := &fakeLogs{lines: map[string][]*container.LogEvent{"a": all}}

	r := runBackfill(t, fake, []container.Container{{ID: "a", Created: created}}, to)

	got := r.matches()
	require.Len(t, got, backfillMatches)
	assert.Equal(t, "capped", r.final.Reason)
	assert.Equal(t, backfillMatches, r.final.Matches)
	// the newest 50 of the 200 matches, oldest first
	assert.Equal(t, uint32(750), got[0].Id)
	assert.Equal(t, uint32(995), got[len(got)-1].Id)
	assert.True(t, r.final.ScannedTo.Equal(time.UnixMilli(got[0].Timestamp)))
}

func Test_searchBackfill_merges_newest_across_containers(t *testing.T) {
	to := time.Now()
	created := to.Add(-24 * time.Hour)
	fake := &fakeLogs{lines: map[string][]*container.LogEvent{
		// a's matches are all older than b's
		"a": lines(created, to.Add(-12*time.Hour), 100, func(int) bool { return true }),
		"b": lines(to.Add(-12*time.Hour), to.Add(-time.Minute), 30, func(int) bool { return true }),
	}}

	r := runBackfill(t, fake, []container.Container{{ID: "a", Created: created}, {ID: "b", Created: created}}, to)

	got := r.matches()
	require.Len(t, got, backfillMatches)
	assert.Equal(t, "capped", r.final.Reason)
	// all 30 of b's and the newest 20 of a's
	assert.Equal(t, uint32(80), got[0].Id)
	for i := 1; i < len(got); i++ {
		assert.LessOrEqual(t, got[i-1].Timestamp, got[i].Timestamp)
	}
}

func Test_searchBackfill_reports_progress(t *testing.T) {
	old := searchProgressEvery
	searchProgressEvery = 0
	t.Cleanup(func() { searchProgressEvery = old })

	to := time.Now()
	created := to.Add(-24 * time.Hour)
	fake := &fakeLogs{lines: map[string][]*container.LogEvent{
		"a": lines(created, to.Add(-time.Minute), 100, func(i int) bool { return i == 50 }),
	}}

	r := runBackfill(t, fake, []container.Container{{ID: "a", Created: created}}, to)

	require.NotEmpty(t, r.progress)
	last := 0.0
	for _, s := range r.progress {
		assert.GreaterOrEqual(t, s.Progress, last)
		assert.LessOrEqual(t, s.Progress, 1.0)
		last = s.Progress
	}
	assert.Greater(t, last, 0.9)
	// the match count climbs while the read is still going
	assert.Equal(t, 1, r.progress[len(r.progress)-1].Matches)
}

func Test_searchBackfill_ignores_lines_past_the_range(t *testing.T) {
	to := time.Now()
	created := to.Add(-24 * time.Hour)
	// 20 matches in the newest window, 200 older ones, and more logged after
	// the search started, all served by a backend that ignores `to`
	older := lines(created, to.Add(-time.Hour), 200, func(int) bool { return true })
	newest := lines(to.Add(-9*time.Second), to.Add(-time.Second), 20, func(int) bool { return true })
	after := lines(to.Add(time.Millisecond), to.Add(time.Minute), 50, func(int) bool { return true })
	for i, e := range newest {
		e.Id = uint32(1000 + i)
	}
	for i, e := range after {
		e.Id = uint32(2000 + i)
	}
	fake := &fakeLogs{ignoreEnd: true, lines: map[string][]*container.LogEvent{
		"a": append(append(older, newest...), after...),
	}}

	r := runBackfill(t, fake, []container.Container{{ID: "a", Created: created}}, to)

	got := r.matches()
	require.Len(t, got, backfillMatches)
	ids := map[uint32]int{}
	for _, e := range got {
		ids[e.Id]++
		assert.Less(t, e.Timestamp, to.UnixMilli(), "nothing logged after the search started")
	}
	for id, n := range ids {
		assert.Equal(t, 1, n, "match %d sent twice", id)
	}
	assert.Equal(t, "capped", r.final.Reason)
}

// One container that cannot be read does not throw away what the others matched.
func Test_searchBackfill_skips_a_failing_container(t *testing.T) {
	to := time.Now()
	created := to.Add(-time.Hour)
	fake := &fakeLogs{
		lines:   map[string][]*container.LogEvent{"b": lines(to.Add(-50*time.Minute), to.Add(-time.Minute), 10, func(int) bool { return true })},
		failing: map[string]bool{"a": true},
	}

	r := runBackfill(t, fake, []container.Container{{ID: "a", Created: created}, {ID: "b", Created: created}}, to)

	assert.Len(t, r.matches(), 10)
	assert.Equal(t, 10, r.final.Matches)
}
