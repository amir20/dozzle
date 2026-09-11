package docker_support

import (
	"context"
	"crypto/tls"
	"errors"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	container_support "github.com/amir20/dozzle/internal/support/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubService answers only Host. RetriableClientManager should not be calling
// anything else while it decides where a client belongs.
type stubService struct {
	container_support.ClientService
	host container.Host
	err  error
}

func (s *stubService) Host(context.Context) (container.Host, error) {
	return s.host, s.err
}

func managerWith(clients map[string]container_support.ClientService) *RetriableClientManager {
	m := NewRetriableClientManager(nil, time.Second, tls.Certificate{})
	m.clients = clients
	return m
}

// An agent mints its host id when its own process starts, so a restarted agent
// answers under a different id than the one the hub filed it under. Podman makes
// that the normal case: its /info hands out a fresh uuid on every call, so every
// agent restart is a new identity. Nothing else repairs this — RetryAndList only
// revisits endpoints that never connected — so the id handed to the UI would be
// one Find has never heard of, and every lookup would fail until the hub itself
// was restarted.
func TestRetriableClientManager_RekeysRestartedAgent(t *testing.T) {
	service := &stubService{host: container.Host{ID: "new", Name: "node-1"}}
	m := managerWith(map[string]container_support.ClientService{"old": service})

	hosts := m.Hosts(t.Context())

	_, found := m.Find("new")
	assert.True(t, found, "the id the agent now reports should resolve")

	_, stale := m.Find("old")
	assert.False(t, stale, "the id it used to report should be gone")

	require.Len(t, hosts, 1)
	assert.Equal(t, "new", hosts[0].ID)
	assert.Equal(t, "old", hosts[0].ReplacesID, "an open tab needs this to drop the stale entry")
}

func TestRetriableClientManager_RekeyNotifiesSubscribers(t *testing.T) {
	service := &stubService{host: container.Host{ID: "new", Name: "node-1"}}
	m := managerWith(map[string]container_support.ClientService{"old": service})

	updates := make(chan container.Host, 1)
	m.Subscribe(t.Context(), updates)

	m.Hosts(t.Context())

	select {
	case host := <-updates:
		assert.Equal(t, "new", host.ID)
		assert.Equal(t, "old", host.ReplacesID)
	case <-time.After(time.Second):
		t.Fatal("expected the re-keyed host to be published")
	}
}

// An unreachable agent reports its last known host with Available false. That
// stale id is not evidence of anything, so the map is left exactly as it is.
func TestRetriableClientManager_DoesNotRekeyUnavailableHost(t *testing.T) {
	service := &stubService{host: container.Host{ID: "cached", Name: "node-1"}, err: errors.New("unreachable")}
	m := managerWith(map[string]container_support.ClientService{"old": service})

	m.Hosts(t.Context())

	_, found := m.Find("old")
	assert.True(t, found, "an unreachable agent should keep the key it had")
	assert.Len(t, m.clients, 1)
}

// Two clients claiming one id is the duplicate-host case, not a restart.
// Re-keying here would evict a host that is working fine, so both stay put.
func TestRetriableClientManager_DoesNotClobberAnotherHost(t *testing.T) {
	drifted := &stubService{host: container.Host{ID: "shared", Name: "node-1"}}
	incumbent := &stubService{host: container.Host{ID: "shared", Name: "node-2"}}
	m := managerWith(map[string]container_support.ClientService{
		"old":    drifted,
		"shared": incumbent,
	})

	m.Hosts(t.Context())

	still, found := m.Find("shared")
	assert.True(t, found)
	assert.Same(t, incumbent, still, "the host already holding the id keeps it")

	_, kept := m.Find("old")
	assert.True(t, kept, "the drifted host stays reachable under its old key")
}

func TestRetriableClientManager_LeavesStableHostAlone(t *testing.T) {
	service := &stubService{host: container.Host{ID: "same", Name: "node-1"}}
	m := managerWith(map[string]container_support.ClientService{"same": service})

	updates := make(chan container.Host, 1)
	m.Subscribe(t.Context(), updates)

	hosts := m.Hosts(t.Context())

	require.Len(t, hosts, 1)
	assert.Empty(t, hosts[0].ReplacesID)

	select {
	case <-updates:
		t.Fatal("a host that did not move should not be published")
	case <-time.After(50 * time.Millisecond):
	}
}
