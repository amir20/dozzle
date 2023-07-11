package docker

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventGenerator_Events_tty(t *testing.T) {
	input := "example input"
	reader := bufio.NewReader(strings.NewReader(input))

	g := NewEventGenerator(reader, true)
	event := <-g.Events

	require.NotNil(t, event, "Expected event to not be nil, but got nil")
	assert.Equal(t, input, event.Message)
}

func TestEventGenerator_Events_non_tty(t *testing.T) {
	input := "example input"
	reader := bytes.NewReader(makeMessage(input, STDOUT))

	g := NewEventGenerator(reader, false)
	event := <-g.Events

	require.NotNil(t, event, "Expected event to not be nil, but got nil")
	assert.Equal(t, input, event.Message)
}

func TestEventGenerator_Events_non_tty_close_channel(t *testing.T) {
	input := "example input"
	reader := bytes.NewReader(makeMessage(input, STDOUT))

	g := NewEventGenerator(reader, false)
	<-g.Events
	_, ok := <-g.Events

	assert.False(t, ok, "Expected channel to be closed")
}

func TestEventGenerator_Events_routines_done(t *testing.T) {
	input := "example input"
	reader := bytes.NewReader(makeMessage(input, STDOUT))

	g := NewEventGenerator(reader, false)
	<-g.Events
	assert.False(t, waitTimeout(&g.wg, 1*time.Second), "Expected routines to be done")
}

func makeMessage(message string, stream StdType) []byte {
	data := make([]byte, 8)
	binary.BigEndian.PutUint32(data[4:], uint32(len(message)))
	data[0] = byte(stream / 2)
	data = append(data, []byte(message)...)

	return data
}

func waitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return false // completed normally
	case <-time.After(timeout):
		return true // timed out
	}
}
