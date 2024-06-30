package docker

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/goccy/go-json"

	orderedmap "github.com/wk8/go-ordered-map/v2"

	log "github.com/sirupsen/logrus"
)

type EventGenerator struct {
	Events      chan *LogEvent
	Errors      chan error
	reader      *bufio.Reader
	next        *LogEvent
	buffer      chan *LogEvent
	tty         bool
	wg          sync.WaitGroup
	containerID string
}

var bufPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

var ErrBadHeader = fmt.Errorf("dozzle/docker: unable to read header")

func NewEventGenerator(reader io.Reader, container Container) *EventGenerator {
	generator := &EventGenerator{
		reader:      bufio.NewReader(reader),
		buffer:      make(chan *LogEvent, 100),
		Errors:      make(chan error, 1),
		Events:      make(chan *LogEvent),
		tty:         container.Tty,
		containerID: container.ID,
	}
	generator.wg.Add(2)
	go generator.consumeReader()
	go generator.processBuffer()
	return generator
}

func (g *EventGenerator) processBuffer() {
	var current, next *LogEvent

	for {
		if g.next != nil {
			current = g.next
			g.next = nil
			next = g.peek()
		} else {
			event, ok := <-g.buffer
			if !ok {
				close(g.Events)
				break
			}

			current = event
			next = g.peek()
		}

		checkPosition(current, next)

		g.Events <- current
	}
	g.wg.Done()
}

func (g *EventGenerator) consumeReader() {
	for {
		message, streamType, readerError := readEvent(g.reader, g.tty)
		if message != "" {
			logEvent := createEvent(message, streamType)
			logEvent.ContainerID = g.containerID
			logEvent.Level = guessLogLevel(logEvent)
			g.buffer <- logEvent
		}

		if readerError != nil {
			if readerError != ErrBadHeader {
				log.Tracef("reader error: %v", readerError)
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

func readEvent(reader *bufio.Reader, tty bool) (string, StdType, error) {
	header := []byte{0, 0, 0, 0, 0, 0, 0, 0}
	buffer := bufPool.Get().(*bytes.Buffer)
	buffer.Reset()
	defer bufPool.Put(buffer)
	var streamType StdType = STDOUT
	if tty {
		message, err := reader.ReadString('\n')
		if err != nil {
			return message, streamType, err
		}
		return message, streamType, nil
	} else {
		n, err := io.ReadFull(reader, header)
		if err != nil {
			return "", streamType, err
		}
		if n != 8 {
			log.Warnf("unable to read header: %v", header)
			message, _ := reader.ReadString('\n')
			return message, streamType, ErrBadHeader
		}

		switch header[0] {
		case 1:
			streamType = STDOUT
		case 2:
			streamType = STDERR
		default:
			log.Warnf("unknown stream type: %v", header[0])
		}

		count := binary.BigEndian.Uint32(header[4:])
		if count == 0 {
			return "", streamType, nil
		}
		_, err = io.CopyN(buffer, reader, int64(count))
		if err != nil {
			return "", streamType, err
		}
		return buffer.String(), streamType, nil
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
			if json.Valid([]byte(message)) {
				data := orderedmap.New[string, any]()
				if err := json.Unmarshal([]byte(message), &data); err != nil {
					var jsonErr *json.UnmarshalTypeError
					if errors.As(err, &jsonErr) {
						if jsonErr.Value == "string" {
							log.Warnf("unable to parse json logs - error was \"%v\" while trying unmarshal \"%v\"", err.Error(), message)
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
			}
		}
	}
	return logEvent
}

func checkPosition(currentEvent *LogEvent, nextEvent *LogEvent) {
	currentLevel := guessLogLevel(currentEvent)
	if nextEvent != nil {
		if currentEvent.IsCloseToTime(nextEvent) && currentLevel != "" && !nextEvent.HasLevel() {
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
