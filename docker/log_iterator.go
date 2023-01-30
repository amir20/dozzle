package docker

import (
	"bufio"
	"encoding/json"
	"hash/fnv"
	"regexp"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

type eventGenerator struct {
	reader    *bufio.Reader
	channel   chan *LogEvent
	next      *LogEvent
	lastError error
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
		currentEvent = <-g.channel

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

	return currentEvent, g.lastError
}

func (g *eventGenerator) LastError() error {
	return g.lastError
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
			g.lastError = readerError
			break
		}
	}
}

var NON_ASCII_REGEX = regexp.MustCompile("^[^a-z ]+[^ewidtf]?")
var KEY_VALUE_REGEX = regexp.MustCompile("level=([^ ]+)")

func guessLogLevel(logEvent *LogEvent) string {
	switch value := logEvent.Message.(type) {
	case string:
		value = NON_ASCII_REGEX.ReplaceAllString(strings.ToLower(value), "")

		levels := []string{"error", "warn", "info", "debug", "trace", "fatal"}
		for _, level := range levels {
			prefix := regexp.MustCompile("^" + level + "[^a-z]")
			if prefix.MatchString(value) {
				return level
			}
		}

		if matches := KEY_VALUE_REGEX.FindStringSubmatch(value); matches != nil {
			return matches[1]
		}

	case map[string]interface{}:
		if value["level"] != nil {
			return strings.ToLower(value["level"].(string))
		}
	}

	return ""
}
