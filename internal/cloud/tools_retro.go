package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/amir20/dozzle/internal/container"
	pb "github.com/amir20/dozzle/proto/cloud"
)

const (
	// retroDefaultWindow is how far back a scan reads when the caller names no
	// start: the cloud's first look covers one day.
	retroDefaultWindow = 24 * time.Hour
	// retroMaxWindow bounds a scan. A week of every container is a lot of disk.
	retroMaxWindow = 7 * 24 * time.Hour
	// retroDefaultDeadline and retroMaxDeadline bound how long one scan runs.
	// Containers not reached in time are reported unscanned, never dropped.
	retroDefaultDeadline = 60 * time.Second
	retroMaxDeadline     = 5 * time.Minute
	// retroContainerBytes caps the lines kept per container. Counts are never
	// capped; only the evidence is.
	retroContainerBytes = 48 * 1024
	// retroTotalBytes keeps the whole response well under gRPC's 4 MiB frame.
	retroTotalBytes = 3 * 1024 * 1024
	// retroMaxLineBytes trims one runaway line (a JSON blob, a stack dump).
	retroMaxLineBytes = 2 * 1024
	// retroWorkers reads this many containers at once. The scan holds one of
	// the cloud's tool slots; going wider here would only move the load onto
	// the Docker daemon.
	retroWorkers = 2
)

var retroDefaultLevels = []string{"error", "fatal", "warn"}

type retroScanArgs struct {
	Since           string `json:"since"`
	Levels          string `json:"levels"`
	DeadlineSeconds int    `json:"deadline_seconds"`
}

func executeRetroScan(ctx context.Context, argsJSON string, deps ToolDeps) (*pb.CallToolResponse, error) {
	var args retroScanArgs
	if argsJSON != "" {
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return nil, fmt.Errorf("failed to parse arguments: %w", err)
		}
	}
	now := time.Now()
	since := now.Add(-retroDefaultWindow)
	if args.Since != "" {
		t, err := time.Parse(time.RFC3339, args.Since)
		if err != nil {
			return nil, fmt.Errorf("invalid since (expected RFC3339): %w", err)
		}
		since = t
	}
	if now.Sub(since) > retroMaxWindow {
		since = now.Add(-retroMaxWindow)
	}
	levels := retroDefaultLevels
	if args.Levels != "" {
		levels = nil
		for l := range strings.SplitSeq(args.Levels, ",") {
			if l = strings.ToLower(strings.TrimSpace(l)); l != "" {
				levels = append(levels, l)
			}
		}
	}
	deadline := retroDefaultDeadline
	if args.DeadlineSeconds > 0 {
		deadline = min(time.Duration(args.DeadlineSeconds)*time.Second, retroMaxDeadline)
	}
	ctx, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()

	scoped := deps.scoped()
	list, errs := scoped.ListAllContainers()
	logHostErrors(errs)
	res := scanContainers(ctx, list, since, now, levels, func(c container.Container) (*container.ContainerService, error) {
		return scoped.FindContainer(c.Host, c.ID)
	})
	return &pb.CallToolResponse{
		Success: true,
		Result:  &pb.CallToolResponse_RetroScan{RetroScan: res},
	}, nil
}

// scanContainers reads every container once, running ones first and then
// stopped ones by how recently they stopped, so a deadline cuts the least
// interesting ones. find resolves a listed container to its service.
func scanContainers(ctx context.Context, list []container.Container, since, now time.Time, levels []string, find func(container.Container) (*container.ContainerService, error)) *pb.RetroScanResult {
	slices.SortStableFunc(list, func(a, b container.Container) int {
		ar, br := a.State == "running", b.State == "running"
		if ar != br {
			if ar {
				return -1
			}
			return 1
		}
		return b.FinishedAt.Compare(a.FinishedAt)
	})

	out := make([]*pb.RetroScanContainer, len(list))
	jobs := make(chan int)
	var wg sync.WaitGroup
	for range retroWorkers {
		wg.Go(func() {
			for i := range jobs {
				out[i] = scanOne(ctx, list[i], since, now, levels, find)
			}
		})
	}
	reached := 0
feed:
	for i := range list {
		select {
		case jobs <- i:
			reached++
		case <-ctx.Done():
			break feed
		}
	}
	close(jobs)
	wg.Wait()

	res := &pb.RetroScanResult{DeadlineExceeded: ctx.Err() != nil}
	for i, c := range list {
		if out[i] == nil {
			out[i] = &pb.RetroScanContainer{Id: c.ID, Name: c.Name, HostId: c.Host, State: c.State}
		}
	}
	res.Containers = out
	res.Truncated = capTotal(out, retroTotalBytes)
	return res
}

func scanOne(ctx context.Context, c container.Container, since, now time.Time, levels []string, find func(container.Container) (*container.ContainerService, error)) *pb.RetroScanContainer {
	// Clamp to when the container was created, not when its latest run
	// started: Docker's json-file and local drivers keep one log across
	// restarts, so a container that crash-looped forty times today and last
	// restarted three minutes ago still has the whole day to read — and is
	// the one a retro review most needs to see.
	from, to := since, now
	if c.Created.After(from) {
		from = c.Created
	}
	if c.State != "running" && !c.FinishedAt.IsZero() && c.FinishedAt.Before(to) {
		to = c.FinishedAt
	}
	r := &pb.RetroScanContainer{
		Id: c.ID, Name: c.Name, HostId: c.Host, State: c.State,
		From: from.UTC().Format(time.RFC3339), To: to.UTC().Format(time.RFC3339),
		LevelCounts: map[string]int64{},
	}
	cs, err := find(c)
	if err != nil {
		r.Error = err.Error()
		return r
	}
	// The resolved container is the inspected one: it knows the restart count
	// and how the last run ended, which a list entry does not.
	r.RestartCount = int32(cs.Container.RestartCount)
	r.OomKilled = cs.Container.OOMKilled
	r.ExitCode = int32(cs.Container.ExitCode)
	if !to.After(from) {
		r.Scanned = true
		return r
	}

	events, err := cs.LogsBetweenDates(ctx, from, to, container.STDOUT|container.STDERR)
	if err != nil {
		r.Error = err.Error()
		return r
	}
	var kept []*pb.LogEntry
	keptBytes := 0
	for ev := range events {
		level := strings.ToLower(ev.Level)
		if !slices.Contains(levels, level) {
			continue
		}
		r.LevelCounts[level]++
		msg := ev.RawMessage
		if msg == "" {
			msg = fmt.Sprintf("%v", ev.Message)
		}
		msg = clipUTF8(msg, retroMaxLineBytes)
		kept = append(kept, &pb.LogEntry{Timestamp: ev.Timestamp, Message: msg, Stream: ev.Stream, Level: level})
		keptBytes += len(msg)
		// Newest win: lines arrive oldest first, so drop from the front.
		for keptBytes > retroContainerBytes && len(kept) > 1 {
			keptBytes -= len(kept[0].Message)
			kept = kept[1:]
			r.LinesTruncated = true
		}
	}
	// A channel closed by the deadline is a partial read, not a scanned one.
	r.Scanned = ctx.Err() == nil
	r.Lines = kept
	return r
}

// capTotal trims lines until the result fits in limit bytes, taking them from
// the containers with the fewest error-class matches first: a container that
// only warned gives up its evidence before one that crashed. Counts stay.
func capTotal(cs []*pb.RetroScanContainer, limit int) bool {
	size := func(c *pb.RetroScanContainer) int {
		n := 0
		for _, l := range c.Lines {
			n += len(l.Message) + 32
		}
		return n
	}
	total := 0
	for _, c := range cs {
		total += size(c)
	}
	if total <= limit {
		return false
	}
	order := slices.Clone(cs)
	slices.SortStableFunc(order, func(a, b *pb.RetroScanContainer) int {
		ea := a.LevelCounts["error"] + a.LevelCounts["fatal"]
		eb := b.LevelCounts["error"] + b.LevelCounts["fatal"]
		return int(ea - eb)
	})
	for _, c := range order {
		if total <= limit {
			break
		}
		total -= size(c)
		c.Lines = nil
		c.LinesTruncated = true
	}
	return true
}

// clipUTF8 cuts msg to at most n bytes on a rune boundary and replaces any
// invalid bytes. LogEntry.message is a proto3 string: one invalid byte fails
// the marshal, and with every container in one message that loses the whole
// scan (grpc-go tears the stream down on a failed send).
func clipUTF8(msg string, n int) string {
	msg = strings.ToValidUTF8(msg, "\uFFFD")
	if len(msg) <= n {
		return msg
	}
	cut := n
	for cut > 0 && !utf8.RuneStart(msg[cut]) {
		cut--
	}
	return msg[:cut]
}
