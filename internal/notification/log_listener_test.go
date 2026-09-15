package notification

import (
	"context"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/stretchr/testify/require"
)

// shortLivedClient emulates a k8s CronJob pod: StreamLogs emits its output and
// returns immediately because the container has already exited.
type shortLivedClient struct {
	container.ClientService
	c container.Container
}

func (s *shortLivedClient) ListContainers(context.Context, container.ContainerLabels) ([]container.Container, error) {
	return nil, nil
}

func (s *shortLivedClient) SubscribeContainersStarted(context.Context, chan<- container.Container) {}

func (s *shortLivedClient) FindContainer(context.Context, string, container.ContainerLabels) (container.Container, error) {
	return s.c, nil
}

func (s *shortLivedClient) Host(context.Context) (container.Host, error) {
	return container.Host{ID: "host"}, nil
}

func (s *shortLivedClient) StreamLogs(_ context.Context, c container.Container, _ time.Time, _ container.StdType, ch chan<- *container.LogEvent) error {
	ch <- &container.LogEvent{ContainerID: c.ID, Message: "hello"}
	return nil
}

type matchAll struct{}

func (matchAll) ShouldListenToContainer(container.Container) bool { return true }

func TestFindContainerAfterShortLivedStreamEnds(t *testing.T) {
	ctx := t.Context()

	c := container.Container{ID: "default:hello-29824580-tznrr:hello", Name: "hello"}
	client := &shortLivedClient{c: c}
	l := NewContainerLogListener(ctx, []container.ClientService{client})
	require.NoError(t, l.Start(matchAll{}))

	l.startListening(c, client, time.Now())

	event := <-l.LogChannel()
	// Give the stream goroutine time to return and clean up, as it does in
	// production before the processor gets to the buffered event.
	require.Eventually(t, func() bool { return !l.isListening(c.ID) }, time.Second, time.Millisecond)

	_, _, err := l.FindContainerWithHost(ctx, event.ContainerID, nil)
	require.NoError(t, err)
}
