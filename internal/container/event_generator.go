package container

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
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

func (g *EventGenerator) processBuffer() {
	var pendingGroup []*LogEvent

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

		if next == nil || !next.IsSimple() || !canContinueGroup(pendingGroup[0], next) {
			if !g.flushGroup(pendingGroup) {
				break loop
			}
			pendingGroup = nil
		} else {
			next.Level = pendingGroup[0].Level
		}
	}

	close(g.Events)
	g.wg.Done()
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
	// Current must have a known level
	if !current.HasLevel() {
		return false
	}
	// Next must not have its own level (continuation)
	if next.HasLevel() {
		return false
	}
	// Must be close in time
	if !current.IsCloseToTime(next) {
		return false
	}
	return true
}

// canContinueGroup checks if next can be added to a group started by first
func canContinueGroup(first, next *LogEvent) bool {
	// Next must not have its own level (continuation)
	if next.HasLevel() {
		return false
	}
	// Must be close in time to the group leader
	if !first.IsCloseToTime(next) {
		return false
	}
	return true
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
	if index := strings.IndexAny(message, " "); index != -1 {
		logId := message[:index]
		if timestamp, err := time.Parse(time.RFC3339Nano, logId); err == nil {
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
