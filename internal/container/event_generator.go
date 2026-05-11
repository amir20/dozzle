package container

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"math"
	"strings"
	"sync"
	"time"

	"encoding/json"

	orderedmap "github.com/wk8/go-ordered-map/v2"

	"github.com/rs/zerolog/log"
)

type EventGenerator struct {
	Events      chan *LogEvent
	Errors      chan error
	reader      LogReader
	next        *LogEvent
	buffer      chan *LogEvent
	wg          sync.WaitGroup
	containerID string
	startedAt   time.Time
	ctx         context.Context
}

var ErrBadHeader = fmt.Errorf("dozzle/docker: unable to read header")

type LogReader interface {
	Read() (string, StdType, error)
}

func NewEventGenerator(ctx context.Context, reader LogReader, container Container) *EventGenerator {
	generator := &EventGenerator{
		reader:      reader,
		buffer:      make(chan *LogEvent, 100),
		Errors:      make(chan error, 1),
		Events:      make(chan *LogEvent),
		containerID: container.ID,
		startedAt:   container.StartedAt,
		ctx:         ctx,
	}
	generator.wg.Add(2)
	go generator.consumeReader()
	go generator.processBuffer()
	return generator
}

func (g *EventGenerator) emit(event *LogEvent) bool {
	select {
	case g.Events <- event:
		return true
	case <-g.ctx.Done():
		return false
	}
}

func (g *EventGenerator) flushGroup(pendingGroup []*LogEvent) bool {
	if len(pendingGroup) == 0 {
		return true
	}

	if len(pendingGroup) == 1 {
		pendingGroup[0].Type = LogTypeSingle
		return g.emit(pendingGroup[0])
	}

	first := pendingGroup[0]
	fragments := make([]LogFragment, len(pendingGroup))
	for i, e := range pendingGroup {
		fragments[i] = LogFragment{Message: e.Message.(string)}
	}

	return g.emit(&LogEvent{
		Type:        LogTypeGroup,
		Message:     fragments,
		Timestamp:   first.Timestamp,
		Id:          first.Id,
		Level:       first.Level,
		Stream:      first.Stream,
		ContainerID: first.ContainerID,
	})
}

// emitAsSingles emits each event individually as LogTypeSingle.
func (g *EventGenerator) emitAsSingles(events []*LogEvent) bool {
	for _, e := range events {
		e.Type = LogTypeSingle
		if !g.emit(e) {
			return false
		}
	}
	return true
}

// skipOrphanedLines drains leading simple events without a level that look
// like orphaned continuation lines from a group already emitted in a prior
// fetch. Returns the first non-orphan event (or nil if the stream ends).
// If no non-orphan event arrives (stream ends or times out waiting), the
// buffered events are emitted as singles — they weren't really orphans.
// Lines near the container start time are never skipped since nothing can
// precede them.
func (g *EventGenerator) skipOrphanedLines() *LogEvent {
	var orphanBuffer []*LogEvent
	var lastTimestamp int64

	// First event must block — we need at least one event to start.
	current := g.nextEvent()
	if current == nil {
		return nil
	}

	// If the first event is near the container start, there can't be prior
	// logs so nothing is orphaned — return immediately.
	if !g.startedAt.IsZero() && current.Timestamp > 0 &&
		math.Abs(float64(g.startedAt.UnixMilli()-current.Timestamp)) < 5000 {
		return current
	}

	for {
		isOrphan := current.IsSimple() && !current.HasLevel() && current.Timestamp > 0 &&
			(lastTimestamp == 0 || math.Abs(float64(lastTimestamp-current.Timestamp)) < maxGroupTimeDelta)

		if !isOrphan {
			if len(orphanBuffer) > 0 {
				// If the chain broke because `current` is far in time from the
				// last buffered line, the buffered lines weren't continuations
				// of anything — they're real isolated entries. Emit them as
				// singles so first-of-window lines aren't silently dropped
				// (e.g. postgres "checkpoint starting: time" — only entry in
				// a 5-min window followed by a 0.4s-later "complete" line).
				timeGap := lastTimestamp != 0 && current.Timestamp > 0 &&
					math.Abs(float64(lastTimestamp-current.Timestamp)) >= maxGroupTimeDelta
				if timeGap {
					g.emitAsSingles(orphanBuffer)
				} else {
					log.Debug().Int("count", len(orphanBuffer)).Str("container", g.containerID).Msg("skipped orphaned continuation lines")
				}
			}
			return current
		}

		lastTimestamp = current.Timestamp
		orphanBuffer = append(orphanBuffer, current)

		// Use peek (with timeout) so we don't block forever on a live stream.
		if next := g.peek(); next == nil {
			// No more events within the timeout — these aren't orphans.
			// Emit them as singles, then block for the next event so the
			// stream continues processing.
			g.emitAsSingles(orphanBuffer)
			return g.nextEvent()
		}
		if current = g.nextEvent(); current == nil {
			g.emitAsSingles(orphanBuffer)
			return nil
		}
	}
}

func (g *EventGenerator) processBuffer() {
	defer func() {
		close(g.Events)
		g.wg.Done()
	}()

	var pendingGroup []*LogEvent

	// Skip leading orphaned continuation lines from a prior fetch.
	first := g.skipOrphanedLines()
	if first == nil {
		return
	}
	// Put the first real event back so the main loop picks it up.
	g.next = first

loop:
	for {
		current := g.nextEvent()
		if current == nil {
			g.flushGroup(pendingGroup)
			break loop
		}

		// Complex logs are emitted immediately
		if !current.IsSimple() {
			if !g.flushGroup(pendingGroup) {
				break loop
			}
			pendingGroup = nil
			if !g.emit(current) {
				break loop
			}
			continue
		}

		// Simple log - peek ahead to decide grouping
		next := g.peek()

		if len(pendingGroup) == 0 {
			if next != nil && next.IsSimple() && canStartGroup(current, next) {
				next.Level = current.Level
				pendingGroup = append(pendingGroup, current)
			} else {
				current.Type = LogTypeSingle
				if !g.emit(current) {
					break loop
				}
			}
			continue
		}

		pendingGroup = append(pendingGroup, current)

		if next == nil || !next.IsSimple() || !canContinueGroup(pendingGroup[len(pendingGroup)-1], next, pendingGroup[0].Level) {
			if !g.flushGroup(pendingGroup) {
				break loop
			}
			pendingGroup = nil
		} else {
			next.Level = pendingGroup[0].Level
		}
	}
}

func (g *EventGenerator) nextEvent() *LogEvent {
	if g.next != nil {
		event := g.next
		g.next = nil
		return event
	}
	event, ok := <-g.buffer
	if !ok {
		return nil
	}
	return event
}

// canStartGroup checks if current can start a group with next
func canStartGroup(current, next *LogEvent) bool {
	return current.HasLevel() && canContinueGroup(current, next, current.Level)
}

// canContinueGroup checks if next can be appended after prev in a group.
// Lines without a level always continue the group. Lines with the same level
// as the group also continue it (e.g. repeated error lines in a stack trace).
func canContinueGroup(prev, next *LogEvent, groupLevel string) bool {
	return (!next.HasLevel() || next.Level == groupLevel) && prev.IsCloseToTime(next)
}

func (g *EventGenerator) consumeReader() {
	for {
		message, streamType, readerError := g.reader.Read()
		if message != "" {
			logEvent := createEvent(message, streamType)
			logEvent.ContainerID = g.containerID
			logEvent.Level = guessLogLevel(logEvent)
			g.buffer <- logEvent
		}

		if readerError != nil {
			if readerError != ErrBadHeader {
				g.Errors <- readerError
				close(g.buffer)
				break
			}
		}
	}
	g.wg.Done()
}

func (g *EventGenerator) peek() *LogEvent {
	if g.next != nil {
		return g.next
	}
	select {
	case event := <-g.buffer:
		g.next = event
		return g.next
	case <-time.After(50 * time.Millisecond):
		return nil
	}
}

func createEvent(message string, streamType StdType) *LogEvent {
	h := fnv.New32a()
	h.Write([]byte(message))
	logEvent := &LogEvent{Id: h.Sum32(), Message: message, Stream: streamType.String(), Type: LogTypeSingle}
	if index := strings.IndexByte(message, ' '); index != -1 {
		if timestamp, err := time.Parse(time.RFC3339Nano, message[:index]); err == nil {
			logEvent.Timestamp = timestamp.UnixMilli()
			message = strings.TrimSuffix(message[index+1:], "\n")
			logEvent.Message = message
			logEvent.RawMessage = message
			if message == "" {
				logEvent.Message = "" // empty message so do nothing
			} else if json.Valid([]byte(message)) {
				data := orderedmap.New[string, any]()
				if err := json.Unmarshal([]byte(message), &data); err != nil {
					var jsonErr *json.UnmarshalTypeError
					if errors.As(err, &jsonErr) {
						if jsonErr.Value == "string" {
							log.Warn().Err(err).Str("value", jsonErr.Value).Msg("failed to unmarshal json")
						}
					}
				} else {
					if data == nil {
						logEvent.Message = ""
					} else {
						logEvent.Message = data
						logEvent.Type = LogTypeComplex
					}
				}
			} else if data, err := ParseLogFmt(message); err == nil {
				logEvent.Message = data
				logEvent.Type = LogTypeComplex
				data, err := json.Marshal(data)
				if err != nil {
					log.Error().Err(err).Msg("failed to marshal json")
				}
				logEvent.RawMessage = string(data)
			}
		}
	}
	return logEvent
}
