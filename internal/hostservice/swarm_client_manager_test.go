package hostservice

import (
	"context"
	"errors"
	"net"
	"testing"
	"testing/synctest"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/puzpuzpuz/xsync/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func swarmManager() *SwarmClientManager {
	return &SwarmClientManager{
		clients: map[string]container.ClientService{
			"node-1": &stubService{host: container.Host{ID: "node-1"}},
			"node-2": &stubService{host: container.Host{ID: "node-2"}},
			"node-3": &stubService{host: container.Host{ID: "node-3"}, err: errors.New("unreachable")},
		},
		subscribers: xsync.NewMap[*hostSubscriber, struct{}](),
		agentManager: managerWith(map[string]container.ClientService{
			"agent": &stubService{host: container.Host{ID: "agent"}},
		}),
	}
}

func TestSwarmClientManager_ListIncludesAgentsAndNodes(t *testing.T) {
	assert.Len(t, swarmManager().List(), 4)
}

// Every node is asked for its host, and one that fails is reported as
// unavailable instead of dropped.
func TestSwarmClientManager_HostsAsksEveryNode(t *testing.T) {
	hosts := swarmManager().Hosts(t.Context())

	assert.ElementsMatch(t, []string{"agent", "node-1", "node-2", "node-3"}, hostIDs(hosts))
	for _, h := range hosts {
		if h.ID == "agent" {
			continue
		}
		assert.Equal(t, "swarm", h.Type)
		assert.Equalf(t, h.ID != "node-3", h.Available, "availability of %s", h.ID)
	}
}

// A node that leaves the swarm drops out of tasks.<service>. Its agent has to go
// with it, or it is listed as unavailable until the hub restarts.
func TestSwarmClientManager_RetryAndListDropsPeerThatLeft(t *testing.T) {
	staying, gone := &closeCounter{}, &closeCounter{}
	m := swarmManager()
	m.peers = map[string]swarmPeer{
		"node-1": {endpoint: "10.0.0.1:7007", closer: staying},
		"node-3": {endpoint: "10.0.0.3:7007", closer: gone},
	}
	m.lookup = func(string) ([]net.IP, error) { return []net.IP{net.ParseIP("10.0.0.1")}, nil }
	updates := make(chan container.Host, 1)
	m.Subscribe(t.Context(), updates)

	m.RetryAndList()

	_, ok := m.Find("node-3")
	assert.False(t, ok, "the peer that left is no longer served")
	_, ok = m.Find("node-1")
	assert.True(t, ok)
	assert.Equal(t, 1, gone.closed)
	assert.Equal(t, 0, staying.closed)

	// An open tab keeps the host's visible set across a removal, so a node that
	// came back later would show up empty there.
	select {
	case h := <-updates:
		assert.Failf(t, "unexpected update", "%+v", h)
	case <-time.After(100 * time.Millisecond):
	}
}

type localStub struct {
	container.Client
	host container.Host
}

func (l *localStub) Host() container.Host { return l.host }

// A peer's id is its node's, so a Dozzle task restarted on the same node leaves
// with its old IP and comes back under a new one with the same id. The node
// stays served through the new client and the old one is closed.
func TestSwarmClientManager_RetryAndListKeepsNodeWhoseTaskRestarted(t *testing.T) {
	old := &closeCounter{}
	closers := map[string]*closeCounter{}
	m := swarmManager()
	m.localClient = &localStub{host: container.Host{ID: "self"}}
	m.peers = map[string]swarmPeer{"node-1": {endpoint: "10.0.0.1:7007", closer: old}}
	m.lookup = func(string) ([]net.IP, error) { return []net.IP{net.ParseIP("10.0.0.9")}, nil }
	m.dial = stubDialer(map[string]container.Host{"10.0.0.9:7007": {ID: "node-1"}}, nil, closers)
	updates := make(chan container.Host, 4)
	m.Subscribe(t.Context(), updates)

	m.RetryAndList()

	_, ok := m.Find("node-1")
	assert.True(t, ok, "the node is still served")
	assert.Equal(t, 1, old.closed, "the client to the old IP is closed")
	assert.Equal(t, "10.0.0.9:7007", m.peers["node-1"].endpoint)

	select {
	case h := <-updates:
		assert.Equal(t, "node-1", h.ID)
		assert.False(t, h.Removed)
		assert.True(t, h.Available)
	case <-time.After(time.Second):
		require.FailNow(t, "the node coming back was not published")
	}
	select {
	case h := <-updates:
		assert.Failf(t, "unexpected update", "%+v", h)
	case <-time.After(100 * time.Millisecond):
	}
}

// ctxService answers Host unless the deadline it was given already passed.
type ctxService struct {
	container.ClientService
	host container.Host
}

func (s *ctxService) Host(ctx context.Context) (container.Host, error) {
	if err := ctx.Err(); err != nil {
		return container.Host{ID: s.host.ID}, err
	}
	return s.host, nil
}

type hangingService struct {
	container.ClientService
	id string
}

func (s *hangingService) Host(ctx context.Context) (container.Host, error) {
	<-ctx.Done()
	return container.Host{ID: s.id}, ctx.Err()
}

// A node that never answers holds its Host call until the deadline. The agents
// share that deadline, so they must not be asked only once it has passed.
func TestSwarmClientManager_HostsAgentsNotStarvedByHangingNode(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		m := &SwarmClientManager{
			clients:     map[string]container.ClientService{"dead": &hangingService{id: "dead"}},
			subscribers: xsync.NewMap[*hostSubscriber, struct{}](),
			agentManager: managerWith(map[string]container.ClientService{
				"agent": &ctxService{host: container.Host{ID: "agent"}},
			}),
		}
		ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
		defer cancel()

		hosts := m.Hosts(ctx)

		available := map[string]bool{}
		for _, h := range hosts {
			available[h.ID] = h.Available
		}
		assert.True(t, available["agent"], "the agent answered within the deadline")
		assert.False(t, available["dead"])
	})
}

// Two live tasks can answer for one node: a start-first update runs the new
// task beside the old one, and a replicated service can put two on a node. The
// peer already serving the node stays, and the duplicate is closed instead of
// replacing it, which would leak the old client and republish the node.
func TestSwarmClientManager_RetryAndListKeepsServingPeerForDuplicateNode(t *testing.T) {
	serving := &closeCounter{}
	closers := map[string]*closeCounter{}
	m := swarmManager()
	m.localClient = &localStub{host: container.Host{ID: "self"}}
	m.peers = map[string]swarmPeer{"node-1": {endpoint: "10.0.0.1:7007", closer: serving}}
	before := m.clients["node-1"]
	m.lookup = func(string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("10.0.0.1"), net.ParseIP("10.0.0.9")}, nil
	}
	m.dial = stubDialer(map[string]container.Host{"10.0.0.9:7007": {ID: "node-1"}}, nil, closers)
	updates := make(chan container.Host, 4)
	m.Subscribe(t.Context(), updates)

	m.RetryAndList()

	assert.Same(t, before, m.clients["node-1"], "the serving client is kept")
	assert.Equal(t, "10.0.0.1:7007", m.peers["node-1"].endpoint)
	assert.Equal(t, 0, serving.closed)
	require.Contains(t, closers, "10.0.0.9:7007")
	assert.Equal(t, 1, closers["10.0.0.9:7007"].closed, "the duplicate is closed")

	select {
	case h := <-updates:
		assert.Failf(t, "unexpected update", "%+v", h)
	case <-time.After(100 * time.Millisecond):
	}
}
