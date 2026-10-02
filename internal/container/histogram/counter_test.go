package histogram

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeLog is a log the way `docker logs --tail N --timestamps` serves it.
type fakeLog struct {
	lines []string
	// reads records the tail size of every read, which is what a request costs.
	reads []int
}

type sliceReader struct {
	lines []string
	i     int
}

func (r *sliceReader) Read() (string, container.StdType, error) {
	if r.i >= len(r.lines) {
		return "", container.STDOUT, io.EOF
	}
	r.i++
	return r.lines[r.i-1], container.STDOUT, nil
}

func (f *fakeLog) tail(_ context.Context, n int) (LineReader, io.Closer, error) {
	f.reads = append(f.reads, n)
	start := max(0, len(f.lines)-n)
	return &sliceReader{lines: f.lines[start:]}, io.NopCloser(nil), nil
}

func (f *fakeLog) linesRead() int {
	total := 0
	for _, n := range f.reads {
		total += min(n, len(f.lines))
	}
	return total
}

// every writes one line per step from start to end, every tenth one an error.
func (f *fakeLog) every(start, end time.Time, step time.Duration) {
	i := 0
	for t := start; t.Before(end); t = t.Add(step) {
		msg := "INFO request handled"
		if i%10 == 0 {
			msg = "ERROR request failed"
		}
		f.lines = append(f.lines, fmt.Sprintf("%s %s %d", t.UTC().Format(time.RFC3339Nano), msg, i))
		i++
	}
}

var base = time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)

func newTestCounter(now time.Time) *Counter {
	c := NewCounter()
	c.now = func() time.Time { return now }
	return c
}

func TestCountMatchesBruteForce(t *testing.T) {
	log := &fakeLog{}
	log.every(base, base.Add(2*time.Hour), 700*time.Millisecond)
	c := newTestCounter(base.Add(2 * time.Hour))

	from, to, width := base.Add(30*time.Minute), base.Add(90*time.Minute), time.Minute
	h, err := c.Count(context.Background(), "a", from, to, width, log.tail)
	require.NoError(t, err)
	require.Equal(t, 60, h.Len())
	assert.True(t, h.ScannedFrom.IsZero())

	want := make([]uint32, 60)
	wantErr := make([]uint32, 60)
	for i, line := range log.lines {
		sec, ok := lineSeconds(line)
		require.True(t, ok)
		ts := time.Unix(sec, 0)
		if ts.Before(from) || !ts.Before(to) {
			continue
		}
		b := int(ts.Sub(from) / width)
		want[b]++
		if i%10 == 0 {
			wantErr[b]++
		}
	}
	assert.Equal(t, want, h.Total)
	assert.Equal(t, wantErr, h.Errors)
}

func TestCountReadsOnlyBackToTheWindow(t *testing.T) {
	log := &fakeLog{}
	// Ten hours at one line a second: 36,000 lines.
	log.every(base, base.Add(10*time.Hour), time.Second)
	now := base.Add(10 * time.Hour)
	c := newTestCounter(now)

	_, err := c.Count(context.Background(), "a", now.Add(-10*time.Minute), now, 10*time.Second, log.tail)
	require.NoError(t, err)
	// The window holds 600 lines; one small read covers it.
	assert.Equal(t, []int{initialLines}, log.reads)
	assert.Less(t, log.linesRead(), len(log.lines)/5)
}

func TestCountSizesTheNextReadFromTheRate(t *testing.T) {
	log := &fakeLog{}
	log.every(base, base.Add(10*time.Hour), 100*time.Millisecond) // 360,000 lines
	now := base.Add(10 * time.Hour)
	c := newTestCounter(now)

	h, err := c.Count(context.Background(), "a", now.Add(-2*time.Hour), now, time.Minute, log.tail)
	require.NoError(t, err)
	assert.True(t, h.ScannedFrom.IsZero())
	// 72,000 lines in the window. The first read sees the rate and the second is
	// sized to cover it, rather than doubling its way there.
	assert.Len(t, log.reads, 2)
	assert.Less(t, log.linesRead(), 72_000*2)
}

func TestCountStopsAtTheBudget(t *testing.T) {
	log := &fakeLog{}
	log.every(base, base.Add(10*time.Hour), 100*time.Millisecond)
	now := base.Add(10 * time.Hour)
	c := newTestCounter(now)
	c.maxLines = 20_000

	h, err := c.Count(context.Background(), "a", now.Add(-5*time.Hour), now, time.Minute, log.tail)
	require.NoError(t, err)
	require.False(t, h.ScannedFrom.IsZero())
	// 20,000 lines at ten a second is 2,000 seconds back from now.
	assert.Equal(t, now.Add(-2000*time.Second).Truncate(time.Minute).Add(time.Minute), h.ScannedFrom)
	for _, n := range log.reads {
		assert.LessOrEqual(t, n, 20_000)
	}
}

func TestCountServesClosedBucketsFromCache(t *testing.T) {
	log := &fakeLog{}
	log.every(base, base.Add(time.Hour), time.Second)
	now := base.Add(time.Hour)
	c := newTestCounter(now)
	from, to := now.Add(-30*time.Minute), now.Add(-10*time.Minute)

	first, err := c.Count(context.Background(), "a", from, to, time.Minute, log.tail)
	require.NoError(t, err)
	reads := len(log.reads)

	again, err := c.Count(context.Background(), "a", from, to, time.Minute, log.tail)
	require.NoError(t, err)
	assert.Equal(t, first, again)
	assert.Len(t, log.reads, reads, "a window that is entirely cached reads nothing")
}

func TestCountReadsOnlyWhatIsNewSinceTheCache(t *testing.T) {
	log := &fakeLog{}
	log.every(base, base.Add(10*time.Hour), 100*time.Millisecond)
	now := base.Add(10 * time.Hour)
	c := newTestCounter(now)

	_, err := c.Count(context.Background(), "a", now.Add(-2*time.Hour), now, time.Minute, log.tail)
	require.NoError(t, err)

	// Five minutes pass. The same "last two hours" now only needs those minutes.
	later := now.Add(5 * time.Minute)
	log.every(now, later, 100*time.Millisecond)
	c.now = func() time.Time { return later }
	log.reads = nil

	h, err := c.Count(context.Background(), "a", later.Add(-2*time.Hour), later, time.Minute, log.tail)
	require.NoError(t, err)
	assert.Equal(t, []int{initialLines}, log.reads)
	assert.EqualValues(t, 600, h.Total[len(h.Total)-2], "a full minute at ten lines a second")
	assert.EqualValues(t, 600, h.Total[0])
}

func TestCountWindowInTheFutureIsEmpty(t *testing.T) {
	log := &fakeLog{}
	log.every(base, base.Add(time.Hour), time.Second)
	now := base.Add(time.Hour)
	c := newTestCounter(now)

	h, err := c.Count(context.Background(), "a", now.Add(time.Hour), now.Add(2*time.Hour), time.Minute, log.tail)
	require.NoError(t, err)
	assert.Empty(t, log.reads)
	assert.Equal(t, make([]uint32, 60), h.Total)
}

func TestCountFindsTheNearestLinesAroundAnEmptyWindow(t *testing.T) {
	log := &fakeLog{}
	// A burst, twelve quiet hours, another burst: a container like flipt.
	log.every(base, base.Add(time.Minute), time.Second)
	log.every(base.Add(13*time.Hour), base.Add(13*time.Hour+time.Minute), time.Second)
	now := base.Add(14 * time.Hour)
	c := newTestCounter(now)

	h, err := c.Count(context.Background(), "a", base.Add(6*time.Hour), base.Add(7*time.Hour), time.Minute, log.tail)
	require.NoError(t, err)
	assert.Equal(t, make([]uint32, 60), h.Total)
	assert.Equal(t, base.Add(59*time.Second), h.Before)
	assert.Equal(t, base.Add(13*time.Hour), h.After)
}

func TestCountLeavesNearestUnsetWhenTheWindowHasLines(t *testing.T) {
	log := &fakeLog{}
	log.every(base, base.Add(time.Hour), time.Second)
	c := newTestCounter(base.Add(time.Hour))

	h, err := c.Count(context.Background(), "a", base, base.Add(time.Hour), time.Minute, log.tail)
	require.NoError(t, err)
	assert.True(t, h.Before.IsZero())
	assert.True(t, h.After.IsZero())
}

func TestCountWindowBeforeTheFirstLineHasOnlyAfter(t *testing.T) {
	log := &fakeLog{}
	log.every(base.Add(time.Hour), base.Add(2*time.Hour), time.Second)
	c := newTestCounter(base.Add(2 * time.Hour))

	h, err := c.Count(context.Background(), "a", base, base.Add(30*time.Minute), time.Minute, log.tail)
	require.NoError(t, err)
	assert.True(t, h.Before.IsZero())
	assert.Equal(t, base.Add(time.Hour), h.After)
}

func TestCountRejectsUnalignedWindows(t *testing.T) {
	c := NewCounter()
	log := &fakeLog{}
	_, err := c.Count(context.Background(), "a", base.Add(30*time.Second), base.Add(time.Hour), time.Minute, log.tail)
	assert.Error(t, err)
	_, err = c.Count(context.Background(), "a", base, base.Add(time.Hour), 1500*time.Millisecond, log.tail)
	assert.Error(t, err)
}

func TestLineSeconds(t *testing.T) {
	got, ok := lineSeconds("2026-10-02T15:04:12.502114+02:00 hello")
	assert.True(t, ok)
	assert.Equal(t, time.Date(2026, 10, 2, 13, 4, 12, 0, time.UTC).Unix(), got)
	_, ok = lineSeconds("not a timestamp")
	assert.False(t, ok)
}
