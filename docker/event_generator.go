package docker

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"regexp"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

type eventGenerator struct {
	reader *bufio.Reader
	events chan *LogEvent
	errors chan error
	next   *LogEvent

	tty bool
}

func NewEventGenerator(reader *io.Reader, tty bool) *eventGenerator {
	generator := &eventGenerator{
		reader:  bufio.NewReader(*reader),
		channel: make(chan *LogEvent), errors: make(chan error), tty: tty}
	go generator.consume()
	return generator
}

func (g *eventGenerator) Next() (*LogEvent, error) {
	var currentEvent *LogEvent
	var nextEvent *LogEvent
	if g.next != nil {
		currentEvent = g.next
		g.next = nil
		nextEvent = g.Peek()
	} else {
		event, ok := <-g.channel
		if !ok {
			return nil, g.lastError
		}

		currentEvent = event
		nextEvent = g.Peek()
	}

	currentLevel := guessLogLevel(currentEvent)

	if nextEvent != nil {
		if currentEvent.IsCloseToTime(nextEvent) && currentLevel != "" && !nextEvent.HasLevel() {
			currentEvent.Position = START
			nextEvent.Position = MIDDLE
		}

		// If next item is not close to current item or has level, set current item position to end
		if currentEvent.Position == MIDDLE && (nextEvent.HasLevel() || !currentEvent.IsCloseToTime(nextEvent)) {
			currentEvent.Position = END
		}

		// If next item is close to current item and has no level, set next item position to middle
		if currentEvent.Position == MIDDLE && !nextEvent.HasLevel() && currentEvent.IsCloseToTime(nextEvent) {
			nextEvent.Position = MIDDLE
		}
		// Set next item level to current item level
		if currentEvent.Position == START || currentEvent.Position == MIDDLE {
			nextEvent.Level = currentEvent.Level
		}
	} else if currentEvent.Position == MIDDLE {
		currentEvent.Position = END
	}

	return currentEvent, nil
}

func (g *eventGenerator) Peek() *LogEvent {
	if g.next != nil {
		return g.next
	}
	select {
	case event := <-g.channel:
		g.next = event
		return g.next
	case <-time.After(50 * time.Millisecond):
		return nil
	}
}

func (g *eventGenerator) consume() {
	for {
		message, streamType, readerError := readEvent(g.reader, g.tty)
		logEvent := createEvent(message, streamType)
		logEvent.Level = guessLogLevel(logEvent)
		g.events <- logEvent

		if readerError != nil {
			g.lastError = readerError
			close(g.channel)
			return
		}
	}
}

func readEvent(reader *bufio.Reader, tty bool) (string, StdType, error) {
	header := []byte{0, 0, 0, 0, 0, 0, 0, 0}
	buffer := bytes.Buffer{} // todo: use a pool
	var streamType StdType = STDOUT
	if tty {
		message, err := reader.ReadString('\n')
		if err != nil {
			return "", streamType, err
		}
		return message, streamType, nil
	} else {
		n, err := reader.Read(header)
		if err != nil {
			return "", streamType, err
		}
		if n != 8 {
			return "", streamType, fmt.Errorf("unable to read header")
		}

		switch header[0] {
		case 1:
			streamType = STDOUT
		case 2:
			streamType = STDERR
		default:
			log.Warnf("unknown stream type %d", header[0])
		}

		count := binary.BigEndian.Uint32(header[4:])
		if count == 0 {
			return "", streamType, nil
		}
		buffer.Reset()
		_, err = io.CopyN(&buffer, reader, int64(count))
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
			if strings.HasPrefix(message, "{") && strings.HasSuffix(message, "}") {
				var data map[string]interface{}
				if err := json.Unmarshal([]byte(message), &data); err != nil {
					log.Warnf("unable to parse json logs - error was \"%v\" while trying unmarshal \"%v\"", err.Error(), message)
				} else {
					logEvent.Message = data
				}
			}
		}
	}
	return logEvent
}

var KEY_VALUE_REGEX = regexp.MustCompile(`level=(\w+)`)
var ANSI_COLOR_REGEX = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func guessLogLevel(logEvent *LogEvent) string {
	switch value := logEvent.Message.(type) {
	case string:
		levels := []string{"error", "warn", "warning", "info", "debug", "trace", "fatal"}
		stripped := ANSI_COLOR_REGEX.ReplaceAllString(value, "") // remove ansi color codes
		for _, level := range levels {
			if match, _ := regexp.MatchString("(?i)^"+level+"[^a-z]", stripped); match {
				return level
			}

			if strings.Contains(value, "["+strings.ToUpper(level)+"]") {
				return level
			}

			if strings.Contains(value, " "+strings.ToUpper(level)+" ") {
				return level
			}
		}

		if matches := KEY_VALUE_REGEX.FindStringSubmatch(value); matches != nil {
			return matches[1]
		}

	case map[string]interface{}:
		if level, ok := value["level"].(string); ok {
			return level
		}
	}

	return ""
}
