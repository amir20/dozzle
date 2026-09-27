package hostservice

import (
	"context"
	"crypto/tls"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// liveService records who subscribed to it and reports one running container.
type liveService struct {
	container.ClientService
	host container.Host

	mu      sync.Mutex
	events  int
	stats   int
	started int
}

func (s *liveService) Host(context.Context) (container.Host, error) { return s.host, nil }

func (s *liveService) SubscribeEvents(context.Context, chan<- container.ContainerEvent) {
	s.mu.Lock()
	s.events++
	s.mu.Unlock()
}

func (s *liveService) SubscribeStats(context.Context, chan<- container.ContainerStat) {
	s.mu.Lock()
	s.stats++
	s.mu.Unlock()
}

func (s *liveService) SubscribeContainersStarted(context.Context, chan<- container.Container) {
	s.mu.Lock()
	s.started++
	s.mu.Unlock()
}

func (s *liveService) ListContainers(context.Context, container.ContainerLabels) ([]container.Container, error) {
	return []container.Container{
		{ID: "web", Host: s.host.ID, State: "running"},
		{ID: "old", Host: s.host.ID, State: "exited"},
	}, nil
}

func (s *liveService) counts() (int, int, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.events, s.stats, s.started
}

func followFixture() (*MultiHostService, *RetriableClientManager, *liveService) {
	added := &liveService{host: container.Host{ID: "nas", Name: "nas"}}
	dial := func(string, tls.Certificate) (container.ClientService, io.Closer, error) {
		return added, &closeCounter{}, nil
	}
	local := &liveService{host: container.Host{ID: "local"}}
	manager := newRetriableClientManager(nil, nil, time.Second, tls.Certificate{}, dial, local)
	return NewMultiHostService(manager, time.Second), manager, added
}

// A host added after a subscriber subscribed used to be invisible to it until it
// reconnected: open tabs never streamed its stats, and the cloud never saw its logs.
func TestMultiHostService_SubscribersFollowAddedHosts(t *testing.T) {
	service, manager, added := followFixture()

	service.SubscribeEventsAndStats(t.Context(), make(chan container.ContainerEvent), make(chan container.ContainerStat))
	started := make(chan container.Container, 4)
	service.SubscribeContainersStarted(t.Context(), started, func(*container.Container) bool { return true })

	_, err := manager.AddAgent(t.Context(), "nas:7007", nil)
	require.NoError(t, err)

	assert.Eventually(t, func() bool {
		events, stats, startedSubs := added.counts()
		return events == 1 && stats == 1 && startedSubs == 1
	}, time.Second, 5*time.Millisecond)

	// What was already running on it arrives as started; stopped containers do not.
	select {
	case c := <-started:
		assert.Equal(t, "web", c.ID)
	case <-time.After(time.Second):
		t.Fatal("running container of the added host never reached the subscriber")
	}
	select {
	case c := <-started:
		t.Fatalf("unexpected container %s", c.ID)
	case <-time.After(50 * time.Millisecond):
	}
}

// Every tab has to hear about a removal, not only the one that made it.
func TestRetriableClientManager_RemoveAgentAnnouncesIt(t *testing.T) {
	_, manager, _ := followFixture()
	_, err := manager.AddAgent(t.Context(), "nas:7007", nil)
	require.NoError(t, err)

	hosts := make(chan container.Host, 4)
	manager.Subscribe(t.Context(), hosts)
	require.NoError(t, manager.RemoveAgent("nas:7007"))

	select {
	case h := <-hosts:
		assert.Equal(t, "nas", h.ID)
		assert.True(t, h.Removed)
		assert.False(t, h.Available)
	case <-time.After(time.Second):
		t.Fatal("removal was never published")
	}
}

// An agent that restarted comes back under a new id, and the restart ended every
// stream open to it. Subscribers have to open them again, or its events, stats and
// new containers stay dark until the page reloads.
func TestMultiHostService_SubscribersResubscribeAfterRekey(t *testing.T) {
	service, manager, added := followFixture()
	_, err := manager.AddAgent(t.Context(), "nas:7007", nil)
	require.NoError(t, err)

	service.SubscribeEventsAndStats(t.Context(), make(chan container.ContainerEvent), make(chan container.ContainerStat))
	started := make(chan container.Container, 4)
	service.SubscribeContainersStarted(t.Context(), started, func(*container.Container) bool { return true })

	added.host.ID = "nas-restarted"
	manager.Hosts(t.Context())

	assert.Eventually(t, func() bool {
		events, stats, startedSubs := added.counts()
		return events == 2 && stats == 2 && startedSubs == 2
	}, time.Second, 5*time.Millisecond)

	// Its containers were running before the restart too, so none is new.
	select {
	case c := <-started:
		t.Fatalf("unexpected container %s", c.ID)
	case <-time.After(50 * time.Millisecond):
	}
}
