package cloud

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/amir20/dozzle/internal/container"
	container_support "github.com/amir20/dozzle/internal/support/container"
	pb "github.com/amir20/dozzle/proto/cloud"
	"github.com/rs/zerolog/log"
)

const cloudMinLevelLabel = "dev.dozzle.cloud.min_level"

// cloudLevelRank ranks canonical log levels by severity. Levels not present
// (e.g. "unknown") rank 0 and always pass through filtering.
var cloudLevelRank = map[string]int{
	"trace": 1,
	"debug": 2,
	"info":  3,
	"warn":  4,
	"error": 5,
	"fatal": 6,
}

// parseMinLevel reads the cloudMinLevelLabel value. rank>0 means filter events
// below that severity. disabled=true means skip the container entirely. valid
// is false when a non-empty value is not a recognised level or "disabled";
// callers should log and ignore the label in that case. An empty value is
// valid and applies no filter.
func parseMinLevel(v string) (rank int, disabled bool, valid bool) {
	v = strings.ToLower(strings.TrimSpace(v))
	if v == "" {
		return 0, false, true
	}
	if v == "disabled" {
		return 0, true, true
	}
	if rank, ok := cloudLevelRank[v]; ok {
		return rank, false, true
	}
	return 0, false, false
}

// LogStreamHostService is the subset of the host service needed by the log
// streamer. MultiHostService and K8sClusterService both satisfy it.
type LogStreamHostService interface {
	ToolHostService
	SubscribeContainersStarted(ctx context.Context, containers chan<- container.Container, filter container_support.ContainerFilter)
}

const (
	logBatchMaxEntries  = 500
	logBatchMaxBytes    = 256 * 1024
	logBatchFlushPeriod = 1 * time.Second
	logReaderChanBuffer = 128
)

// logStreamer streams raw container log lines to Dozzle Cloud as unsolicited
// LogBatch ToolResponses. It is created per cloud connection and torn down
// when the connection drops; a new one is started fresh on reconnect.
type logStreamer struct {
	hostService LogStreamHostService
	labels      container.ContainerLabels
	send        func(resp *pb.ToolResponse) error

	mu      sync.Mutex
	readers map[string]context.CancelFunc

	wg sync.WaitGroup
}

func newLogStreamer(hostService LogStreamHostService, labels container.ContainerLabels, send func(resp *pb.ToolResponse) error) *logStreamer {
	return &logStreamer{
		hostService: hostService,
		labels:      labels,
		send:        send,
		readers:     make(map[string]context.CancelFunc),
	}
}

// run blocks until ctx is cancelled. It launches readers for all currently
// running containers and subscribes to new-container events to launch readers
// for containers started after connect.
func (ls *logStreamer) run(ctx context.Context) {
	// Subscribe BEFORE snapshotting so we don't miss a container that starts
	// between snapshot and subscribe.
	started := make(chan container.Container, 64)
	ls.hostService.SubscribeContainersStarted(ctx, started, func(_ *container.Container) bool { return true })

	existing, errs := ls.hostService.ListAllContainers(ls.labels)
	for _, err := range errs {
		if err != nil {
			log.Debug().Err(err).Msg("log streamer: error listing containers from host")
		}
	}
	for _, c := range existing {
		if c.State != "running" {
			continue
		}
		ls.startReader(ctx, c)
	}

	for {
		select {
		case <-ctx.Done():
			ls.wg.Wait()
			return
		case c, ok := <-started:
			if !ok {
				ls.wg.Wait()
				return
			}
			if c.State != "running" {
				continue
			}
			ls.startReader(ctx, c)
		}
	}
}

func readerKey(hostID, containerID string) string {
	return hostID + "|" + containerID
}

func (ls *logStreamer) startReader(parent context.Context, c container.Container) {
	minRank, disabled, valid := parseMinLevel(c.Labels[cloudMinLevelLabel])
	if disabled {
		log.Debug().Str("container", c.Name).Str("host", c.Host).Msg("log streamer: container disabled via label")
		return
	}
	if !valid {
		log.Error().
			Str("container", c.Name).
			Str("host", c.Host).
			Str("label", cloudMinLevelLabel).
			Str("value", c.Labels[cloudMinLevelLabel]).
			Msg("log streamer: invalid min_level label value, ignoring (expected one of: trace, debug, info, warn, error, fatal, disabled)")
	}

	key := readerKey(c.Host, c.ID)

	ls.mu.Lock()
	if _, exists := ls.readers[key]; exists {
		ls.mu.Unlock()
		return
	}
	readerCtx, cancel := context.WithCancel(parent)
	ls.readers[key] = cancel
	ls.mu.Unlock()

	cs, err := ls.hostService.FindContainer(c.Host, c.ID, ls.labels)
	if err != nil {
		ls.mu.Lock()
		delete(ls.readers, key)
		ls.mu.Unlock()
		cancel()
		log.Debug().Err(err).Str("container", c.ID).Str("host", c.Host).Msg("log streamer: could not find container, skipping")
		return
	}

	ls.wg.Add(1)
	go func() {
		defer ls.wg.Done()
		defer func() {
			ls.mu.Lock()
			delete(ls.readers, key)
			ls.mu.Unlock()
			cancel()
		}()
		ls.runReader(readerCtx, cs, minRank)
	}()
}

// runReader follows logs from a single container and pushes batches directly
// to the cloud via ls.send. send() is serialised by the caller, so a slow
// cloud connection backpressures all readers — this is intentional.
func (ls *logStreamer) runReader(ctx context.Context, cs *container_support.ContainerService, minRank int) {
	events := make(chan *container.LogEvent, logReaderChanBuffer)

	streamErr := make(chan error, 1)
	go func() {
		defer close(events)
		// Start from "now" to avoid replaying historical logs on every reconnect.
		streamErr <- cs.StreamLogs(ctx, time.Now(), container.STDOUT|container.STDERR, events)
	}()

	hostID := cs.Container.Host
	containerID := cs.Container.ID
	containerName := cs.Container.Name

	log.Debug().Str("container", containerName).Str("host", hostID).Msg("log streamer: reader started")

	var batch []*pb.LogBatchEntry
	var batchBytes int
	flushTicker := time.NewTicker(logBatchFlushPeriod)
	defer flushTicker.Stop()

	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		err := ls.send(&pb.ToolResponse{Type: &pb.ToolResponse_LogBatch{LogBatch: &pb.LogBatch{Entries: batch}}})
		batch = nil
		batchBytes = 0
		return err
	}

	defer func() {
		_ = flush()
		log.Debug().Str("container", containerName).Str("host", hostID).Msg("log streamer: reader stopped")
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-flushTicker.C:
			if err := flush(); err != nil {
				log.Debug().Err(err).Msg("log streamer: send failed")
				return
			}
		case ev, ok := <-events:
			if !ok {
				if err := <-streamErr; err != nil && ctx.Err() == nil {
					log.Debug().Err(err).Str("container", containerName).Msg("log streamer: StreamLogs ended with error")
				}
				return
			}

			if minRank > 0 {
				if r := cloudLevelRank[ev.Level]; r > 0 && r < minRank {
					continue
				}
			}

			msg := ev.RawMessage
			if msg == "" {
				msg = messageToString(ev.Message)
			}

			tsNs := ev.Timestamp * int64(time.Millisecond) // LogEvent.Timestamp is UnixMilli
			if tsNs == 0 {
				tsNs = time.Now().UnixNano()
			}

			level := ev.Level
			if level == "unknown" {
				level = ""
			}

			batch = append(batch, &pb.LogBatchEntry{
				HostId:        hostID,
				ContainerId:   containerID,
				ContainerName: containerName,
				TimestampNs:   tsNs,
				Message:       msg,
				Stream:        ev.Stream,
				Level:         level,
				LogId:         ev.Id,
			})
			batchBytes += len(msg)

			if len(batch) >= logBatchMaxEntries || batchBytes >= logBatchMaxBytes {
				if err := flush(); err != nil {
					log.Debug().Err(err).Msg("log streamer: send failed")
					return
				}
			}
		}
	}
}

// messageToString renders a LogEvent.Message of any concrete type into a
// string suitable for transport. Grouped multi-line events don't set
// RawMessage, so JSON-encode their fragment slice as a fallback.
func messageToString(m any) string {
	switch v := m.(type) {
	case nil:
		return ""
	case string:
		return v
	default:
		b, err := json.Marshal(v)
		if err != nil {
			log.Debug().Err(err).Msg("log streamer: failed to marshal message")
			return ""
		}
		return string(b)
	}
}
