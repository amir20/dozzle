package cloud

import (
	"context"
	"sync"
	"time"

	"github.com/amir20/dozzle/internal/container"
	container_support "github.com/amir20/dozzle/internal/support/container"
	pb "github.com/amir20/dozzle/proto/cloud"
	"github.com/rs/zerolog/log"
)

// LogStreamHostService is the subset of the host service needed by the log
// streamer. MultiHostService and K8sClusterService both satisfy it.
type LogStreamHostService interface {
	ToolHostService
	SubscribeContainersStarted(ctx context.Context, containers chan<- container.Container, filter container_support.ContainerFilter)
}

const (
	// batch flush thresholds
	logBatchMaxEntries  = 500
	logBatchMaxBytes    = 256 * 1024
	logBatchFlushPeriod = 1 * time.Second

	// global back-pressure limits
	logBatchQueueSize    = 32    // number of batches queued to the sender
	logMaxPendingEntries = 10000 // cap across all in-flight container readers

	// per-container channel buffer
	logReaderChanBuffer = 128

	// how often to emit a rate-limited "dropped N lines" log per container
	dropLogInterval = 30 * time.Second
)

// logStreamer streams raw container log lines to Dozzle Cloud as unsolicited
// LogBatch ToolResponses. It is created per cloud connection and torn down
// when the connection drops; a new one is started fresh on reconnect.
type logStreamer struct {
	hostService LogStreamHostService
	labels      container.ContainerLabels
	send        func(resp *pb.ToolResponse) error

	// outgoing batches; a single goroutine drains this and calls send().
	outbound chan *pb.LogBatch

	// running readers, keyed by host|containerID. Used to avoid launching
	// duplicate readers for a container that appears twice (start event +
	// initial snapshot race).
	mu      sync.Mutex
	readers map[string]context.CancelFunc

	// total pending entries across all readers (atomic would be fine but we
	// already hold the mu when mutating readers; use separate mu for counter).
	pendingMu sync.Mutex
	pending   int

	wg sync.WaitGroup
}

func newLogStreamer(hostService LogStreamHostService, labels container.ContainerLabels, send func(resp *pb.ToolResponse) error) *logStreamer {
	return &logStreamer{
		hostService: hostService,
		labels:      labels,
		send:        send,
		outbound:    make(chan *pb.LogBatch, logBatchQueueSize),
		readers:     make(map[string]context.CancelFunc),
	}
}

// run blocks until ctx is cancelled. It launches readers for all currently
// running containers and subscribes to new-container events to launch readers
// for containers started after connect.
func (ls *logStreamer) run(ctx context.Context) {
	// Sender goroutine: drains outbound and pushes each batch on the wire.
	ls.wg.Add(1)
	go ls.runSender(ctx)

	// Subscribe to new-container events BEFORE snapshotting so we don't miss
	// a container that starts between snapshot and subscribe.
	started := make(chan container.Container, 64)
	ls.hostService.SubscribeContainersStarted(ctx, started, func(_ *container.Container) bool { return true })

	// Initial snapshot of all currently-running containers.
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
		ls.runReader(readerCtx, cs)
	}()
}

// runReader follows logs from a single container and pushes entries into
// batches. Returns when the log stream ends (EOF, container stopped) or ctx
// is cancelled.
func (ls *logStreamer) runReader(ctx context.Context, cs *container_support.ContainerService) {
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
	var droppedSinceLastLog int
	dropLogTicker := time.NewTicker(dropLogInterval)
	defer dropLogTicker.Stop()
	flushTicker := time.NewTicker(logBatchFlushPeriod)
	defer flushTicker.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}
		// Non-blocking send to outbound. If the outbound queue is full, drop
		// this batch — we never block the reader.
		lb := &pb.LogBatch{Entries: batch}
		select {
		case ls.outbound <- lb:
		default:
			droppedSinceLastLog += len(batch)
			ls.addPending(-len(batch))
		}
		batch = nil
		batchBytes = 0
	}

	defer func() {
		flush()
		log.Debug().Str("container", containerName).Str("host", hostID).Msg("log streamer: reader stopped")
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-dropLogTicker.C:
			if droppedSinceLastLog > 0 {
				log.Warn().Int("dropped", droppedSinceLastLog).Str("container", containerName).Msg("log streamer: dropped lines due to backpressure")
				droppedSinceLastLog = 0
			}
		case <-flushTicker.C:
			flush()
		case ev, ok := <-events:
			if !ok {
				// Stream ended (container stopped / EOF). Final flush happens
				// in the defer.
				if err := <-streamErr; err != nil && ctx.Err() == nil {
					log.Debug().Err(err).Str("container", containerName).Msg("log streamer: StreamLogs ended with error")
				}
				return
			}

			// Enforce global pending cap — if over, drop this entry.
			if !ls.tryAddPending(1) {
				droppedSinceLastLog++
				continue
			}

			msg := ev.RawMessage
			if msg == "" {
				if s, ok := ev.Message.(string); ok {
					msg = s
				}
			}

			tsNs := ev.Timestamp * int64(time.Millisecond) // LogEvent.Timestamp is UnixMilli
			if tsNs == 0 {
				tsNs = time.Now().UnixNano()
			}

			level := ev.Level
			if level == "unknown" {
				level = ""
			}

			entry := &pb.LogBatchEntry{
				HostId:        hostID,
				ContainerId:   containerID,
				ContainerName: containerName,
				TimestampNs:   tsNs,
				Message:       msg,
				Stream:        ev.Stream,
				Level:         level,
			}
			batch = append(batch, entry)
			batchBytes += len(msg) + len(hostID) + len(containerID) + len(containerName) + len(ev.Stream) + len(level) + 24

			if len(batch) >= logBatchMaxEntries || batchBytes >= logBatchMaxBytes {
				flush()
			}
		}
	}
}

// runSender drains the outbound channel and pushes each batch onto the cloud
// stream via ls.send, which serialises with the rest of the tool-response
// traffic on the same bidi stream.
func (ls *logStreamer) runSender(ctx context.Context) {
	defer ls.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case lb, ok := <-ls.outbound:
			if !ok {
				return
			}
			ls.addPending(-len(lb.Entries))
			resp := &pb.ToolResponse{
				Type: &pb.ToolResponse_LogBatch{LogBatch: lb},
			}
			if err := ls.send(resp); err != nil {
				// If the send fails, the surrounding connection is going
				// away — the stream.Recv loop in client.go will return an
				// error shortly and tear us down. Stop trying.
				log.Debug().Err(err).Msg("log streamer: send failed; aborting sender")
				return
			}
		}
	}
}

// tryAddPending atomically adds n to the pending counter if it stays within
// logMaxPendingEntries. Returns false if the addition would exceed the cap.
func (ls *logStreamer) tryAddPending(n int) bool {
	ls.pendingMu.Lock()
	defer ls.pendingMu.Unlock()
	if ls.pending+n > logMaxPendingEntries {
		return false
	}
	ls.pending += n
	return true
}

func (ls *logStreamer) addPending(n int) {
	ls.pendingMu.Lock()
	ls.pending += n
	if ls.pending < 0 {
		ls.pending = 0
	}
	ls.pendingMu.Unlock()
}
