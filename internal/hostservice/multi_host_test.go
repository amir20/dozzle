package hostservice

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type listManager struct {
	ClientManager
	clients []container.ClientService
}

func (m *listManager) List() []container.ClientService { return m.clients }

// Subscribe is a no-op: no host joins later in these tests.
func (m *listManager) Subscribe(context.Context, chan<- container.Host) {}

type containersService struct {
	stubService
	containers []container.Container
}

func (s *containersService) ListContainers(context.Context, container.ContainerLabels) ([]container.Container, error) {
	return s.containers, s.err
}

// Every host is listed, and one that fails shows up as an unavailable host
// error rather than hiding the rest.
func TestMultiHostService_ListAllContainersAsksEveryHost(t *testing.T) {
	service := NewMultiHostService(managerWith(map[string]container.ClientService{
		"a": &containersService{host: container.Host{ID: "a"}, containers: []container.Container{{ID: "a1"}, {ID: "a2"}}},
		"b": &containersService{host: container.Host{ID: "b"}, containers: []container.Container{{ID: "b1"}}},
		"c": &containersService{host: container.Host{ID: "c"}, err: errors.New("down")},
	}), time.Second)

	containers, errs := service.ListAllContainers(nil)

	ids := make([]string, 0, len(containers))
	for _, c := range containers {
		ids = append(ids, c.ID)
	}
	assert.ElementsMatch(t, []string{"a1", "a2", "b1"}, ids)
	require.Len(t, errs, 1)
	var unavailable *HostUnavailableError
	require.ErrorAs(t, errs[0], &unavailable)
	assert.Equal(t, "c", unavailable.Host.ID)
}

// startedService hands the subscribed channel back to the test so it can play
// the store's producer.
type startedService struct {
	container.ClientService
	subscribed chan chan<- container.Container
}

func (s *startedService) SubscribeContainersStarted(_ context.Context, ch chan<- container.Container) {
	s.subscribed <- ch
}

// A store only unsubscribes after ctx ends, so it can still send once the
// forwarder is gone. That send used to race a close of the channel and panic.
func TestSubscribeContainersStarted_SendAfterCancelDoesNotPanic(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		svc := &startedService{subscribed: make(chan chan<- container.Container, 1)}
		m := NewMultiHostService(&listManager{clients: []container.ClientService{svc}}, 0)

		ctx, cancel := context.WithCancel(t.Context())
		out := make(chan container.Container, 1)
		m.SubscribeContainersStarted(ctx, out, func(c *container.Container) bool { return c.ID != "skip" })
		ch := <-svc.subscribed

		ch <- container.Container{ID: "skip"}
		ch <- container.Container{ID: "a"}
		assert.Equal(t, "a", (<-out).ID)

		cancel()
		synctest.Wait() // the forwarder has returned

		// Same shape as sendBounded. Each round would pick the send on a closed
		// channel half the time, so 64 clean rounds rule the old close out.
		for range 64 {
			select {
			case ch <- container.Container{ID: "late"}:
				t.Fatal("forwarder received after cancel")
			case <-ctx.Done():
			}
		}
		assert.Empty(t, out)
	})
}
