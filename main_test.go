package main

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	container_support "github.com/amir20/dozzle/internal/support/container"
	docker_support "github.com/amir20/dozzle/internal/support/docker"
	"github.com/stretchr/testify/assert"
)

// fakeClientService is a ClientService that only knows its own host. Every
// other method is left nil on the embedded interface — the cloud host service
// selection logic under test never calls them.
type fakeClientService struct {
	container_support.ClientService
	host container.Host

	subscribed atomic.Bool // set when SubscribeContainersStarted is called
}

func (f *fakeClientService) Host(context.Context) (container.Host, error) {
	return f.host, nil
}

func (f *fakeClientService) SubscribeContainersStarted(context.Context, chan<- container.Container) {
	f.subscribed.Store(true)
}

// fakeClientManager mimics a hub: one local docker daemon plus one
// --remote-agent endpoint. LocalClientServices filters agents out, exactly as
// the real managers do by concrete type.
type fakeClientManager struct {
	mu    sync.Mutex
	local container_support.ClientService
	agent container_support.ClientService
}

// addAgent makes an agent reachable, standing in for one that was down at boot
// and joined later.
func (m *fakeClientManager) addAgent(s container_support.ClientService) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.agent = s
}

func (m *fakeClientManager) Find(id string) (container_support.ClientService, bool) {
	for _, s := range m.all() {
		if h, _ := s.Host(context.Background()); h.ID == id {
			return s, true
		}
	}
	return nil, false
}

// all skips a nil agent so a hub can be built with its agent still unreachable.
func (m *fakeClientManager) all() []container_support.ClientService {
	m.mu.Lock()
	defer m.mu.Unlock()

	services := []container_support.ClientService{m.local}
	if m.agent != nil {
		services = append(services, m.agent)
	}
	return services
}

func (m *fakeClientManager) List() []container_support.ClientService { return m.all() }

func (m *fakeClientManager) RetryAndList() ([]container_support.ClientService, []error) {
	return m.all(), nil
}

func (m *fakeClientManager) Subscribe(ctx context.Context, channel chan<- container.Host) {}

func (m *fakeClientManager) Hosts(ctx context.Context) []container.Host {
	hosts := make([]container.Host, 0, 2)
	for _, s := range m.all() {
		h, _ := s.Host(ctx)
		hosts = append(hosts, h)
	}
	return hosts
}

func (m *fakeClientManager) LocalClients() []container.Client { return nil }

func (m *fakeClientManager) LocalClientServices() []container_support.ClientService {
	return []container_support.ClientService{m.local}
}

func newFakeHub() *docker_support.MultiHostService {
	return docker_support.NewMultiHostService(&fakeClientManager{
		local: &fakeClientService{host: container.Host{ID: "local-id", Name: "hub", Type: "local"}},
		agent: &fakeClientService{host: container.Host{ID: "agent-id", Name: "home-assistant", Type: "agent"}},
	}, time.Second)
}

func hostIDs(hosts []container.Host) []string {
	ids := make([]string, 0, len(hosts))
	for _, h := range hosts {
		ids = append(ids, h.ID)
	}
	return ids
}

// A hub with a --remote-agent is the only process that knows about that agent,
// so the cloud must see it. Scoping the cloud view to local docker hid every
// agent from Dozzle Cloud entirely — no host, no containers, no logs, no metrics.
func TestNewCloudHostService_ServerModeIncludesAgents(t *testing.T) {
	svc := newCloudHostService("server", newFakeHub())

	assert.ElementsMatch(t, []string{"local-id", "agent-id"}, hostIDs(svc.Hosts()))
}

// Swarm is the exception: every replica discovers every peer, so a replica
// reporting the whole fleet would multiply hosts — and duplicate log and stat
// ingestion — by replica count on the cloud side.
func TestNewCloudHostService_SwarmModeStaysLocal(t *testing.T) {
	svc := newCloudHostService("swarm", newFakeHub())

	assert.Equal(t, []string{"local-id"}, hostIDs(svc.Hosts()))
}

// An agent that is unreachable at boot joins later through RetryAndList. The
// cloud view reads the fleet per call rather than snapshotting it at startup,
// so such an agent appears without a restart.
func TestCloudHostService_PicksUpLateJoiningAgents(t *testing.T) {
	mgr := &fakeClientManager{
		local: &fakeClientService{host: container.Host{ID: "local-id", Name: "hub", Type: "local"}},
	}
	svc := newCloudHostService("server", docker_support.NewMultiHostService(mgr, time.Second))
	assert.Equal(t, []string{"local-id"}, hostIDs(svc.Hosts()))

	mgr.addAgent(&fakeClientService{host: container.Host{ID: "agent-id", Name: "home-assistant", Type: "agent"}})

	assert.ElementsMatch(t, []string{"local-id", "agent-id"}, hostIDs(svc.Hosts()))
}

// Appearing in Hosts() is not enough — a late agent has to be picked up by the
// log and stat subscriptions too, or the cloud sees the host and never a line
// from it. SubscribeAvailableHosts only fires on the unreachable-to-reachable
// edge, so the re-attach timer is what has to carry this.
func TestCloudHostService_SubscribesLateJoiningAgents(t *testing.T) {
	reattachInterval = 10 * time.Millisecond
	t.Cleanup(func() { reattachInterval = time.Minute })

	local := &fakeClientService{host: container.Host{ID: "local-id", Name: "hub", Type: "local"}}
	mgr := &fakeClientManager{local: local}
	svc := newCloudHostService("server", docker_support.NewMultiHostService(mgr, time.Second))

	ctx := t.Context()
	svc.SubscribeContainersStarted(ctx, make(chan container.Container, 1), func(*container.Container) bool { return true })
	assert.True(t, local.subscribed.Load())

	agent := &fakeClientService{host: container.Host{ID: "agent-id", Name: "home-assistant", Type: "agent"}}
	mgr.addAgent(agent)

	assert.Eventually(t, agent.subscribed.Load, time.Second, 5*time.Millisecond)
}
