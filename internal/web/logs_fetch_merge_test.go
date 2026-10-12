package web

import (
	"context"
	"errors"
	"io"
	"math"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/stretchr/testify/require"
)

// failingClient fails every log read, so a merged page has one container it cannot read.
type failingClient struct{ container.ClientService }

func (failingClient) LogsBetweenDates(context.Context, container.Container, time.Time, time.Time, container.StdType) (<-chan *container.LogEvent, error) {
	return nil, errors.New("engine unavailable")
}

func (failingClient) RawLogs(context.Context, container.Container, time.Time, time.Time, container.StdType) (io.ReadCloser, error) {
	return nil, errors.New("engine unavailable")
}

func Test_fetchWindowedLogsMerged_one_failure_is_a_gap(t *testing.T) {
	win := fetchWindow{from: time.Unix(0, 0), to: time.Unix(100, 0), size: defaultFetchSize, maxStart: math.MaxInt}
	bad := container.NewContainerService(failingClient{}, container.Container{ID: "bad"})

	_, err := fetchWindowedLogsMerged(context.Background(), []*container.ContainerService{bad}, win, logFilter{}, container.STDOUT)
	require.Error(t, err, "a page where nothing could be read is an error")
}

// oneLineClient serves a single line for any window.
type oneLineClient struct{ container.ClientService }

func (oneLineClient) LogsBetweenDates(_ context.Context, c container.Container, _ time.Time, _ time.Time, _ container.StdType) (<-chan *container.LogEvent, error) {
	ch := make(chan *container.LogEvent, 1)
	ch <- &container.LogEvent{Id: 1, Timestamp: 50_000, ContainerID: c.ID, Message: "hello", RawMessage: "hello", Level: "info"}
	close(ch)
	return ch, nil
}

func Test_fetchWindowedLogsMerged_skips_a_failing_container(t *testing.T) {
	win := fetchWindow{from: time.Unix(0, 0), to: time.Unix(100, 0), size: defaultFetchSize, maxStart: math.MaxInt}
	good := container.NewContainerService(oneLineClient{}, container.Container{ID: "good"})
	bad := container.NewContainerService(failingClient{}, container.Container{ID: "bad"})

	// The windowed fetch keeps only the levels asked for.
	filter := logFilter{levels: map[string]struct{}{"info": {}}}
	events, err := fetchWindowedLogsMerged(context.Background(), []*container.ContainerService{bad, good}, win, filter, container.STDOUT)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, "good", events[0].ContainerID)
}
