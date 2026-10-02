// Package histogram counts a container's log lines per time bucket, for the
// volume ribbon above a time range.
//
// There is no index to ask. Docker's json-file and local drivers store a log as
// one file per rotation with nothing that maps a time to an offset, so a read
// with `since` decodes every line from the oldest file onwards no matter how
// small the window is: on a 4M-line log, asking for the last second cost 60% of
// reading the whole thing. A read with `tail` seeks from the end instead, and
// its cost follows the number of lines it returns.
//
// So the counter reads backwards. It asks for the newest lines, and when the
// oldest of them is still newer than the window, asks again for more, sized from
// how much time the last read covered. Its cost tracks the lines between the
// start of the window and now, never the size of the file, and a budget caps it
// so one request against a container that logs millions of lines an hour cannot
// pin the daemon: past the budget the answer says how far back it got.
//
// Nothing is persisted. Buckets that have closed never change, so each
// container keeps a short run of them in memory per bucket width, and a repeat
// request (the same range reopened, shifted, or the picker opened again) only
// reads the lines written since.
package histogram

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/container/logparse"
	"golang.org/x/sync/semaphore"
	"golang.org/x/sync/singleflight"
)

// LineReader yields one log line per call, oldest first, each starting with the
// RFC 3339 timestamp Docker and the kubelet prefix it with. It returns io.EOF
// after the last line.
type LineReader interface {
	Read() (string, container.StdType, error)
}

// TailFunc opens the newest `lines` lines of one container's log, both streams.
type TailFunc func(ctx context.Context, lines int) (LineReader, io.Closer, error)

const (
	// The first read is small: on most containers it already reaches back past
	// the window, and the next one is sized from what it saw.
	initialLines = 4096
	// defaultMaxLines bounds one request. Reading and counting runs at a few
	// million lines a second, so this is a second or two of daemon time.
	defaultMaxLines = 4_000_000
	// Lines are stamped by the daemon as they arrive, so the newest few seconds
	// can still grow after a read. Only buckets older than this are cached.
	settle = 5 * time.Second
	// maxSeriesBuckets caps one cached run; at 8 bytes a bucket this is 32KB.
	maxSeriesBuckets = 4096
	maxCachedSeries  = 128
	// Two scans at once is enough for several viewers and keeps a burst of them
	// from stacking reads on the daemon.
	maxConcurrentScans = 2
)

type seriesKey struct {
	id    string
	width time.Duration
}

// series is a contiguous run of closed buckets, starting at start.
type series struct {
	start  time.Time
	total  []uint32
	errors []uint32
	used   time.Time
}

func (s *series) end(width time.Duration) time.Time {
	return s.start.Add(time.Duration(len(s.total)) * width)
}

type Counter struct {
	mu     sync.Mutex
	cache  map[seriesKey]*series
	scans  *semaphore.Weighted
	flight singleflight.Group

	now      func() time.Time
	maxLines int
}

func NewCounter() *Counter {
	return &Counter{
		cache:    make(map[seriesKey]*series),
		scans:    semaphore.NewWeighted(maxConcurrentScans),
		now:      time.Now,
		maxLines: defaultMaxLines,
	}
}

// Count returns the histogram of [from, to) in buckets of width. from and to
// must be multiples of width. Identical requests that arrive together share
// one read.
func (c *Counter) Count(ctx context.Context, id string, from, to time.Time, width time.Duration, tail TailFunc) (container.LogHistogram, error) {
	if width < time.Second || width%time.Second != 0 {
		return container.LogHistogram{}, fmt.Errorf("width must be whole seconds, got %s", width)
	}
	if !from.Before(to) || from.Truncate(width) != from || to.Truncate(width) != to {
		return container.LogHistogram{}, fmt.Errorf("from and to must be ordered multiples of %s", width)
	}

	key := fmt.Sprintf("%s|%d|%d|%d", id, width, from.Unix(), to.Unix())
	v, err, _ := c.flight.Do(key, func() (any, error) {
		h, err := c.count(ctx, seriesKey{id, width}, from, to, tail)
		if err != nil || !h.ScannedFrom.IsZero() || !empty(h) || !from.Before(c.now()) {
			return h, err
		}
		// An empty window says nothing on its own; where the nearest lines are
		// is what lets the view offer a way to them. Rare, so never cached.
		h.Before, h.After, err = c.nearest(ctx, from, to, tail)
		return h, err
	})
	if err != nil {
		return container.LogHistogram{}, err
	}
	// Callers that shared the read must not share the slices.
	h := v.(container.LogHistogram)
	h.Total = append([]uint32(nil), h.Total...)
	h.Errors = append([]uint32(nil), h.Errors...)
	return h, nil
}

func (c *Counter) count(ctx context.Context, key seriesKey, from, to time.Time, tail TailFunc) (container.LogHistogram, error) {
	width := key.width
	n := int(to.Sub(from) / width)
	out := container.LogHistogram{Start: from, Width: width, Total: make([]uint32, n), Errors: make([]uint32, n)}
	now := c.now()

	// Whatever the cache holds from `from` onwards is reused, and only the lines
	// after its end are read. A cache that starts after `from` cannot help: the
	// read has to go back past it anyway.
	stopAt := from
	c.mu.Lock()
	if s := c.cache[key]; s != nil && !s.start.After(from) {
		s.used = now
		end := s.end(width)
		copyBuckets(&out, s.start, s.total, s.errors)
		if !end.Before(to) {
			c.mu.Unlock()
			return out, nil
		}
		if end.After(from) {
			stopAt = end
		}
	}
	c.mu.Unlock()

	// A window that ends in the future is empty past now; nothing to read there.
	if !stopAt.Before(now) {
		return out, nil
	}

	scanned, err := c.scan(ctx, stopAt, now, width, tail)
	if err != nil {
		return container.LogHistogram{}, err
	}
	copyBuckets(&out, scanned.start, scanned.total, scanned.errors)
	if !scanned.scannedFrom.IsZero() {
		out.ScannedFrom = scanned.scannedFrom
		// The gap between what was cached and where the read stopped is unknown,
		// so nothing here is stored: a later request will try again.
		return out, nil
	}

	c.store(key, scanned, now)
	return out, nil
}

func empty(h container.LogHistogram) bool {
	for _, n := range h.Total {
		if n > 0 {
			return false
		}
	}
	return true
}

// copyBuckets adds the run starting at start into the overlapping part of h.
func copyBuckets(h *container.LogHistogram, start time.Time, total, errs []uint32) {
	offset := int(start.Sub(h.Start) / h.Width)
	for i := range total {
		j := offset + i
		if j < 0 {
			continue
		}
		if j >= len(h.Total) {
			break
		}
		h.Total[j] = total[i]
		h.Errors[j] = errs[i]
	}
}

func (c *Counter) store(key seriesKey, scanned scanResult, now time.Time) {
	width := key.width
	closed := now.Add(-settle).Truncate(width)
	keep := int(closed.Sub(scanned.start) / width)
	if keep <= 0 {
		return
	}
	keep = min(keep, len(scanned.total))
	total, errs := scanned.total[:keep], scanned.errors[:keep]

	c.mu.Lock()
	defer c.mu.Unlock()

	next := &series{start: scanned.start, used: now}
	// Extend the run already cached when the read picked up where it ended.
	if s := c.cache[key]; s != nil && !s.start.After(scanned.start) && !s.end(width).Before(scanned.start) {
		head := int(scanned.start.Sub(s.start) / width)
		next.start = s.start
		next.total = append(append(make([]uint32, 0, head+keep), s.total[:head]...), total...)
		next.errors = append(append(make([]uint32, 0, head+keep), s.errors[:head]...), errs...)
	} else {
		next.total = append([]uint32(nil), total...)
		next.errors = append([]uint32(nil), errs...)
	}
	if extra := len(next.total) - maxSeriesBuckets; extra > 0 {
		next.start = next.start.Add(time.Duration(extra) * width)
		next.total = next.total[extra:]
		next.errors = next.errors[extra:]
	}
	c.cache[key] = next

	if len(c.cache) > maxCachedSeries {
		var oldest seriesKey
		var oldestUsed time.Time
		for k, s := range c.cache {
			if oldestUsed.IsZero() || s.used.Before(oldestUsed) {
				oldest, oldestUsed = k, s.used
			}
		}
		delete(c.cache, oldest)
	}
}

type scanResult struct {
	start  time.Time
	total  []uint32
	errors []uint32
	// scannedFrom is set when the budget ran out before the read reached start.
	scannedFrom time.Time
}

// scan counts every line from stopAt to now, reading from the end of the log
// until it reaches stopAt or the budget.
func (c *Counter) scan(ctx context.Context, stopAt, now time.Time, width time.Duration, tail TailFunc) (scanResult, error) {
	if err := c.scans.Acquire(ctx, 1); err != nil {
		return scanResult{}, err
	}
	defer c.scans.Release(1)

	limit := now.Add(width - 1).Truncate(width)
	if furthest := stopAt.Add(maxSeriesBuckets * width); limit.After(furthest) {
		limit = furthest
	}

	lines := initialLines
	for {
		res, read, oldest, err := countTail(ctx, tail, lines, stopAt, limit, width)
		if err != nil {
			return scanResult{}, err
		}
		// A short read means the whole log was returned: nothing older exists.
		if read < lines || oldest.IsZero() || oldest.Before(stopAt) {
			return res, nil
		}
		if lines >= c.maxLines {
			res.scannedFrom = oldest.Add(width - 1).Truncate(width)
			return res, nil
		}

		// Size the next read from the line rate the last one saw, with headroom so
		// one more read is usually enough. Never less than double, so a burst
		// right at the end of the log cannot make the estimate crawl.
		next := lines * 2
		if covered := now.Sub(oldest); covered > 0 {
			estimate := float64(lines) * float64(now.Sub(stopAt)) / float64(covered) * 1.25
			if estimate > float64(next) {
				next = int(min(estimate, float64(c.maxLines)))
			}
		}
		lines = min(next, c.maxLines)
	}
}

// countTail reads the newest `lines` lines once and counts those in
// [stopAt, limit). It returns how many lines it read and the oldest timestamp.
func countTail(ctx context.Context, tail TailFunc, lines int, stopAt, limit time.Time, width time.Duration) (scanResult, int, time.Time, error) {
	n := int(limit.Sub(stopAt) / width)
	res := scanResult{start: stopAt, total: make([]uint32, n), errors: make([]uint32, n)}
	from, until, step := stopAt.Unix(), limit.Unix(), int64(width/time.Second)

	read, oldest, err := readTail(ctx, tail, lines, func(sec int64, line string, std container.StdType) {
		if sec >= from && sec < until {
			i := (sec - from) / step
			res.total[i]++
			if logparse.IsErrorLine(line, std) {
				res.errors[i]++
			}
		}
	})
	return res, read, oldest, err
}

// readTail reads the newest `lines` lines once, handing each timestamped one to
// visit. It returns how many lines it read and the oldest timestamp, zero when
// no line had one.
func readTail(ctx context.Context, tail TailFunc, lines int, visit func(sec int64, line string, std container.StdType)) (int, time.Time, error) {
	reader, closer, err := tail(ctx, lines)
	if err != nil {
		return 0, time.Time{}, err
	}
	defer closer.Close()

	read := 0
	var oldest int64
	haveOldest := false
	for {
		line, std, err := reader.Read()
		if line != "" {
			read++
			if sec, ok := lineSeconds(line); ok {
				if !haveOldest {
					oldest, haveOldest = sec, true
				}
				visit(sec, line, std)
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return read, time.Time{}, err
		}
	}
	if err := ctx.Err(); err != nil {
		return read, time.Time{}, err
	}
	if !haveOldest {
		return read, time.Time{}, nil
	}
	return read, time.Unix(oldest, 0).UTC(), nil
}

// nearest finds the newest line before from and the oldest at or after to, for
// a window that holds none. It reads back the same way a count does, growing
// the read until it passes from, and gives up on whichever side the budget
// leaves unanswered.
func (c *Counter) nearest(ctx context.Context, from, to time.Time, tail TailFunc) (before, after time.Time, err error) {
	if err := c.scans.Acquire(ctx, 1); err != nil {
		return time.Time{}, time.Time{}, err
	}
	defer c.scans.Release(1)

	start, end := from.Unix(), to.Unix()
	lines := initialLines
	for {
		var newestBefore, oldestAfter int64
		haveBefore, haveAfter := false, false
		read, oldest, err := readTail(ctx, tail, lines, func(sec int64, _ string, _ container.StdType) {
			if sec < start && (!haveBefore || sec > newestBefore) {
				newestBefore, haveBefore = sec, true
			}
			if sec >= end && (!haveAfter || sec < oldestAfter) {
				oldestAfter, haveAfter = sec, true
			}
		})
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		whole := read < lines || oldest.IsZero()
		// The oldest line after the window is only known once the read reaches
		// back into the window; before that, older ones may still be unread.
		if haveAfter && (whole || oldest.Before(to)) {
			after = time.Unix(oldestAfter, 0).UTC()
		}
		if haveBefore {
			before = time.Unix(newestBefore, 0).UTC()
		}
		if whole || haveBefore || lines >= c.maxLines {
			return before, after, nil
		}
		lines = min(lines*4, c.maxLines)
	}
}
