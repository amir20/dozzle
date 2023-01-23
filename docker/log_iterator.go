package docker

import (
	"bufio"
	"encoding/json"
	"hash/fnv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

type eventGenerator struct {
	reader  *bufio.Reader
	channel chan *LogEvent
	next    *LogEvent
	error   error
}

func NewEventIterator(reader *bufio.Reader) *eventGenerator {
	generator := &eventGenerator{reader: reader, channel: make(chan *LogEvent, 100)}
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
			return nil, g.error
		}
		currentEvent = event
		nextEvent = g.Peek()
	}

	currentLevel := guessLogLevel(currentEvent)

	if nextEvent != nil {
		if currentEvent.Timestamp == nextEvent.Timestamp && currentLevel != "" && nextEvent.Level == "" {
			currentEvent.Position = START
			nextEvent.Position = MIDDLE
		}

		if currentEvent.Position == MIDDLE && (nextEvent.Level != "" || currentEvent.Timestamp != nextEvent.Timestamp) {
			currentEvent.Position = END
		}

		if currentEvent.Position == MIDDLE && nextEvent.Level == "" && currentEvent.Timestamp == nextEvent.Timestamp {
			nextEvent.Position = MIDDLE
		}
		if currentEvent.Position == START || currentEvent.Position == MIDDLE {
			nextEvent.Level = currentEvent.Level
		}
	} else if currentEvent.Position == MIDDLE {
		currentEvent.Position = END
	}

	return currentEvent, nil
}

func (g *eventGenerator) LastError() error {
	return g.error
}

func (g *eventGenerator) Peek() *LogEvent {
	if g.next != nil {
		return g.next
	}
	select {
	case event := <-g.channel:
		g.next = event
		return g.next
	default:
		return nil
	}
}

func (g *eventGenerator) consume() {
	for {
		message, readerError := g.reader.ReadString('\n')

		h := fnv.New32a()
		h.Write([]byte(message))

		logEvent := &LogEvent{Id: h.Sum32(), Message: message}

		if index := strings.IndexAny(message, " "); index != -1 {
			logId := message[:index]
			if timestamp, err := time.Parse(time.RFC3339Nano, logId); err == nil {
				logEvent.Timestamp = timestamp.UnixMilli()
				message = strings.TrimSuffix(message[index+1:], "\n")
				logEvent.Message = message
				if strings.HasPrefix(message, "{") && strings.HasSuffix(message, "}") {
					var data map[string]interface{}
					if err := json.Unmarshal([]byte(message), &data); err != nil {
						log.Errorf("json unmarshal error while streaming %v", err.Error())
					} else {
						logEvent.Message = data
					}
				}
			}
		}

		logEvent.Level = guessLogLevel(logEvent)

		g.channel <- logEvent

		if readerError != nil {
			close(g.channel)
			g.error = readerError
			break
		}
	}
}

func guessLogLevel(logEvent *LogEvent) string {
	if logEvent.Message == nil {
		return "info"
	}

	switch logEvent.Message.(type) {
	case string:
		message := logEvent.Message.(string)
		if strings.HasPrefix(message, "ERROR") {
			return "error"
		}
		if strings.HasPrefix(message, "WARN") {
			return "warn"
		}
		if strings.HasPrefix(message, "INFO") {
			return "info"
		}
		if strings.HasPrefix(message, "DEBUG") {
			return "debug"
		}
		if strings.HasPrefix(message, "TRACE") {
			return "trace"
		}
		if strings.HasPrefix(message, "FATAL") {
			return "fatal"
		}

	case map[string]interface{}:
		message := logEvent.Message.(map[string]interface{})
		if message["level"] != nil {
			return message["level"].(string)
		}
	}

	return ""
}
