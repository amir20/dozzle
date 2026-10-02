package cloud

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/amir20/dozzle/internal/container"
	pb "github.com/amir20/dozzle/proto/cloud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// scriptedLogs serves LogsBetweenDates from a per-container list, honouring
// the window and closing the channel when ctx ends, like the real clients.
type scriptedLogs struct {
	*fakeClientService
	logs  map[string][]*container.LogEvent
	block map[string]bool // never deliver, to exercise the deadline
}

func (s *scriptedLogs) LogsBetweenDates(ctx context.Context, c container.Container, from, to time.Time, _ container.StdType) (<-chan *container.LogEvent, error) {
	if c.ID == "broken" {
		return nil, fmt.Errorf("configured logging driver does not support reading")
	}
	ch := make(chan *container.LogEvent)
	go func() {
		defer close(ch)
		if s.block[c.ID] {
			<-ctx.Done()
			return
		}
		for _, ev := range s.logs[c.ID] {
			t := time.UnixMilli(ev.Timestamp)
			if t.Before(from) || t.After(to) {
				continue
			}
			select {
			case ch <- ev:
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch, nil
}

var scanNow = time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)

func lines(n int, level, msg string, every time.Duration) []*container.LogEvent {
	out := make([]*container.LogEvent, n)
	for i := range n {
		out[i] = &container.LogEvent{Timestamp: scanNow.Add(-time.Duration(n-i) * every).UnixMilli(), Level: level, RawMessage: fmt.Sprintf(msg, i)}
	}
	return out
}

func scan(t *testing.T, ctx context.Context, svc *scriptedLogs, list []container.Container) *pb.RetroScanResult {
	t.Helper()
	return scanContainers(ctx, list, scanNow.Add(-24*time.Hour), scanNow, retroDefaultLevels, func(c container.Container) (*container.ContainerService, error) {
		return container.NewContainerService(svc, c), nil
	})
}

func TestRetroScanCountsEveryLevelInOnePass(t *testing.T) {
	logs := append(append(append(
		lines(500, "error", "upstream reset %d", time.Minute),
		lines(20, "warn", "slow query %d", time.Hour)...),
		lines(3, "fatal", "panic %d", time.Hour)...),
		lines(1000, "info", "GET / %d", time.Second)...)
	svc := &scriptedLogs{fakeClientService: newFakeClientService("h"), logs: map[string][]*container.LogEvent{"api": logs}}
	res := scan(t, context.Background(), svc, []container.Container{{ID: "api", Name: "api", Host: "h", State: "running", RestartCount: 31, OOMKilled: true, ExitCode: 137}})

	require.Len(t, res.Containers, 1)
	c := res.Containers[0]
	assert.True(t, c.Scanned)
	assert.Equal(t, map[string]int64{"error": 500, "warn": 20, "fatal": 3}, c.LevelCounts, "true counts, not the 100-line floor")
	assert.EqualValues(t, 31, c.RestartCount)
	assert.True(t, c.OomKilled)
	assert.EqualValues(t, 137, c.ExitCode)
	for _, l := range c.Lines {
		assert.NotEqual(t, "info", l.Level)
	}
}

func TestRetroScanKeepsTheNewestLinesUnderTheCap(t *testing.T) {
	big := strings.Repeat("x", 1000) + " %d"
	svc := &scriptedLogs{fakeClientService: newFakeClientService("h"), logs: map[string][]*container.LogEvent{"api": lines(200, "error", big, time.Minute)}}
	c := scan(t, context.Background(), svc, []container.Container{{ID: "api", State: "running"}}).Containers[0]

	assert.EqualValues(t, 200, c.LevelCounts["error"])
	assert.True(t, c.LinesTruncated)
	assert.Less(t, len(c.Lines), 200)
	assert.True(t, strings.HasSuffix(c.Lines[len(c.Lines)-1].Message, " 199"), "the newest line is kept")
}

func TestRetroScanReportsUnreadableAndStoppedContainers(t *testing.T) {
	stopped := scanNow.Add(-2 * time.Hour)
	svc := &scriptedLogs{fakeClientService: newFakeClientService("h"), logs: map[string][]*container.LogEvent{
		"old": append(lines(5, "error", "before stop %d", time.Minute+stoppedOffset), lines(1, "error", "after %d", time.Second)...),
	}}
	res := scan(t, context.Background(), svc, []container.Container{
		{ID: "broken", Name: "broken", State: "running"},
		{ID: "old", Name: "old", State: "exited", FinishedAt: stopped},
	})
	byID := map[string]*pb.RetroScanContainer{}
	for _, c := range res.Containers {
		byID[c.Id] = c
	}
	assert.NotEmpty(t, byID["broken"].Error)
	assert.Equal(t, stopped.Format(time.RFC3339), byID["old"].To, "a stopped container is read up to when it stopped")
	assert.EqualValues(t, 5, byID["old"].LevelCounts["error"])
}

const stoppedOffset = 2 * time.Hour

func TestRetroScanDeadlineMarksTheRestUnscanned(t *testing.T) {
	svc := &scriptedLogs{
		fakeClientService: newFakeClientService("h"),
		logs:              map[string][]*container.LogEvent{"a": lines(1, "error", "x %d", time.Minute)},
		block:             map[string]bool{"slow1": true, "slow2": true},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	res := scan(t, ctx, svc, []container.Container{
		{ID: "a", State: "running"}, {ID: "slow1", State: "running"}, {ID: "slow2", State: "running"},
		{ID: "never", State: "exited", FinishedAt: scanNow.Add(-time.Hour)},
	})
	assert.True(t, res.DeadlineExceeded)
	require.Len(t, res.Containers, 4, "every container is reported, scanned or not")
	for _, c := range res.Containers {
		if c.Id == "a" {
			assert.True(t, c.Scanned)
		} else {
			assert.False(t, c.Scanned, c.Id)
		}
	}
}

func TestCapTotalDropsLinesFromTheQuietestFirst(t *testing.T) {
	mk := func(id string, errs int64, n int) *pb.RetroScanContainer {
		c := &pb.RetroScanContainer{Id: id, LevelCounts: map[string]int64{"error": errs}}
		for range n {
			c.Lines = append(c.Lines, &pb.LogEntry{Message: strings.Repeat("y", 100)})
		}
		return c
	}
	loud, quiet := mk("loud", 900, 10), mk("quiet", 1, 10)
	assert.True(t, capTotal([]*pb.RetroScanContainer{loud, quiet}, 1500))
	assert.Empty(t, quiet.Lines)
	assert.True(t, quiet.LinesTruncated)
	assert.Len(t, loud.Lines, 10)
	assert.EqualValues(t, 1, quiet.LevelCounts["error"], "counts survive the cap")
}

func TestRetroScanIsNeverOfferedToAModel(t *testing.T) {
	for _, d := range AvailableTools(false, Principal{Kind: PrincipalInstance}) {
		if d.Name == toolRetroScan {
			assert.True(t, d.Internal)
			assert.True(t, d.ReadOnly)
			return
		}
	}
	t.Fatal("retro_scan must be listed")
}

// A crash-looping container's earlier runs are in the same log file; reading
// only the latest run would report near-zero counts next to 40 restarts.
func TestRetroScanReadsEveryRunInTheWindow(t *testing.T) {
	svc := &scriptedLogs{fakeClientService: newFakeClientService("h"), logs: map[string][]*container.LogEvent{
		"loop": lines(40, "error", "crash %d", 30*time.Minute),
	}}
	c := scan(t, context.Background(), svc, []container.Container{{
		ID: "loop", State: "running", RestartCount: 40,
		Created: scanNow.Add(-48 * time.Hour), StartedAt: scanNow.Add(-3 * time.Minute),
	}}).Containers[0]
	assert.EqualValues(t, 40, c.LevelCounts["error"], "every run since the window opened, not just the latest")
	assert.Equal(t, scanNow.Add(-24*time.Hour).Format(time.RFC3339), c.From)
}

func TestClipUTF8NeverSplitsARune(t *testing.T) {
	msg := strings.Repeat("a", 2047) + "é" + "tail"
	got := clipUTF8(msg, 2048)
	assert.True(t, utf8.ValidString(got))
	assert.Equal(t, 2047, len(got), "the partial rune is dropped, not kept")
	assert.True(t, utf8.ValidString(clipUTF8("bad \xff byte", 100)))
}

func TestUpdateContainerKeepsALongBudget(t *testing.T) {
	assert.Greater(t, toolCallTimeout(toolUpdateContainer), 10*time.Minute, "image pulls run on the call's context")
	assert.Greater(t, toolCallTimeout(toolRetroScan), retroMaxDeadline)
}
