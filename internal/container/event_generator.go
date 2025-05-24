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

func (g *EventGenerator) processBuffer() {
	var current, next *LogEvent

loop:
	for {
		if g.next != nil {
			current = g.next
			g.next = nil
			next = g.peek()
		} else {
			event, ok := <-g.buffer
			if !ok {
				break loop
			}
			current = event
			next = g.peek()
		}

		checkPosition(current, next)

		select {
		case g.Events <- current:
		case <-g.ctx.Done():
			break loop
		}
	}

	close(g.Events)

	g.wg.Done()
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
	logEvent := &LogEvent{Id: h.Sum32(), Message: message, Stream: streamType.String()}
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
					}
				}
			} else if data, err := ParseLogFmt(message); err == nil {
				logEvent.Message = data
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

func checkPosition(currentEvent *LogEvent, nextEvent *LogEvent) {
	currentLevel := guessLogLevel(currentEvent)
	if nextEvent != nil {
		if currentEvent.IsCloseToTime(nextEvent) && currentLevel != "unknown" && !nextEvent.HasLevel() {
			currentEvent.Position = Beginning
			nextEvent.Position = Middle
		}

		// If next item is not close to current item or has level, set current item position to end
		if currentEvent.Position == Middle && (nextEvent.HasLevel() || !currentEvent.IsCloseToTime(nextEvent)) {
			currentEvent.Position = End
		}

		// If next item is close to current item and has no level, set next item position to middle
		if currentEvent.Position == Middle && !nextEvent.HasLevel() && currentEvent.IsCloseToTime(nextEvent) {
			nextEvent.Position = Middle
		}
		// Set next item level to current item level
		if currentEvent.Position == Beginning || currentEvent.Position == Middle {
			nextEvent.Level = currentEvent.Level
		}
	} else if currentEvent.Position == Middle {
		currentEvent.Position = End
	}
}
