package hostservice

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/analytics"
	"github.com/amir20/dozzle/internal/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type closeCounter struct{ closed int }

func (c *closeCounter) Close() error {
	c.closed++
	return nil
}

// stubDialer answers each endpoint with the host or error listed for it and
// records every closer it hands out.
func stubDialer(hosts map[string]container.Host, errs map[string]error, closers map[string]*closeCounter) agentDialer {
	return func(endpoint string, _ tls.Certificate) (container.ClientService, io.Closer, error) {
		c := &closeCounter{}
		closers[endpoint] = c
		return &stubService{host: hosts[endpoint], err: errs[endpoint]}, c, nil
	}
}

func TestRetriableClientManager_AddAgentServesAndNotifies(t *testing.T) {
	closers := map[string]*closeCounter{}
	m := newRetriableClientManager(nil, nil, time.Second, tls.Certificate{},
		stubDialer(map[string]container.Host{"nas:7007|nas": {ID: "nas-id", Name: "nas"}}, nil, closers))

	hosts := make(chan container.Host, 1)
	m.Subscribe(t.Context(), hosts)

	host, err := m.AddAgent(t.Context(), "nas:7007|nas", nil)
	require.NoError(t, err)
	assert.Equal(t, "nas-id", host.ID)

	_, ok := m.Find("nas-id")
	assert.True(t, ok)
	assert.Equal(t, "nas-id", m.AgentHostID("nas:7007|nas"))

	select {
	case published := <-hosts:
		assert.Equal(t, "nas-id", published.ID)
		assert.True(t, published.Available)
	case <-time.After(time.Second):
		t.Fatal("subscriber never heard about the new agent")
	}

	_, err = m.AddAgent(t.Context(), "nas:7007|nas", nil)
	assert.ErrorIs(t, err, ErrAgentExists)
}

// A failed add must leave nothing behind: no unavailable host, no open client.
func TestRetriableClientManager_AddAgentFailureKeepsNothing(t *testing.T) {
	closers := map[string]*closeCounter{}
	m := newRetriableClientManager(nil, nil, time.Second, tls.Certificate{},
		stubDialer(nil, map[string]error{"down:7007": errors.New("connection refused")}, closers))

	_, err := m.AddAgent(t.Context(), "down:7007", nil)
	require.Error(t, err)

	assert.Empty(t, m.Hosts(context.Background()))
	assert.Equal(t, 1, closers["down:7007"].closed)
}

// Two addresses reaching the same agent would otherwise show one machine twice.
func TestRetriableClientManager_AddAgentRefusesDuplicateHost(t *testing.T) {
	closers := map[string]*closeCounter{}
	same := container.Host{ID: "same-id"}
	m := newRetriableClientManager(nil, nil, time.Second, tls.Certificate{},
		stubDialer(map[string]container.Host{"a:7007": same, "b:7007": same}, nil, closers))

	_, err := m.AddAgent(t.Context(), "a:7007", nil)
	require.NoError(t, err)
	_, err = m.AddAgent(t.Context(), "b:7007", nil)
	assert.ErrorIs(t, err, ErrDuplicateHost)
	assert.Equal(t, 1, closers["b:7007"].closed)
}

func TestRetriableClientManager_RemoveAgent(t *testing.T) {
	closers := map[string]*closeCounter{}
	m := newRetriableClientManager(nil, nil, time.Second, tls.Certificate{},
		stubDialer(map[string]container.Host{"nas:7007": {ID: "nas-id"}}, nil, closers))

	_, err := m.AddAgent(t.Context(), "nas:7007", nil)
	require.NoError(t, err)

	require.NoError(t, m.RemoveAgent("nas:7007"))
	_, ok := m.Find("nas-id")
	assert.False(t, ok)
	assert.Equal(t, 1, closers["nas:7007"].closed)
	assert.ErrorIs(t, m.RemoveAgent("nas:7007"), ErrAgentNotFound)
}

// An agent that never connected at startup is still removable.
func TestRetriableClientManager_RemoveFailedAgent(t *testing.T) {
	closers := map[string]*closeCounter{}
	m := newRetriableClientManager([]string{"down:7007"}, nil, time.Second, tls.Certificate{},
		stubDialer(nil, map[string]error{"down:7007": errors.New("refused")}, closers))
	require.Len(t, m.Hosts(context.Background()), 1)

	require.NoError(t, m.RemoveAgent("down:7007"))
	assert.Empty(t, m.Hosts(context.Background()))
}

// The endpoint keeps pointing at the host after the agent restarts under a new id.
func TestRetriableClientManager_RekeyKeepsAgentEndpoint(t *testing.T) {
	closers := map[string]*closeCounter{}
	m := newRetriableClientManager(nil, nil, time.Second, tls.Certificate{},
		stubDialer(map[string]container.Host{"nas:7007": {ID: "old-id"}}, nil, closers))
	_, err := m.AddAgent(t.Context(), "nas:7007", nil)
	require.NoError(t, err)

	service, _ := m.Find("old-id")
	service.(*stubService).host = container.Host{ID: "new-id"}
	m.Hosts(context.Background())

	assert.Equal(t, "new-id", m.AgentHostID("nas:7007"))
	require.NoError(t, m.RemoveAgent("nas:7007"))
	_, ok := m.Find("new-id")
	assert.False(t, ok)
}

// An agent given the private pair is dialed with it, on add and on every retry.
func TestRetriableClientManager_AgentCertOverride(t *testing.T) {
	private := tls.Certificate{Certificate: [][]byte{[]byte("private")}}
	var dialed []string
	dial := func(endpoint string, certs tls.Certificate) (container.ClientService, io.Closer, error) {
		dialed = append(dialed, endpoint+"="+string(append([]byte{}, firstOr(certs)...)))
		return &stubService{host: container.Host{ID: endpoint}}, &closeCounter{}, nil
	}
	m := newRetriableClientManager(nil, nil, time.Second, tls.Certificate{Certificate: [][]byte{[]byte("hub")}}, dial)

	_, err := m.AddAgent(t.Context(), "private:7007", &private)
	require.NoError(t, err)
	_, err = m.AddAgent(t.Context(), "shared:7007", nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"private:7007=private", "shared:7007=hub"}, dialed)
	assert.Equal(t, "private", string(m.certFor("private:7007").Certificate[0]))

	require.NoError(t, m.RemoveAgent("private:7007"))
	assert.Equal(t, "hub", string(m.certFor("private:7007").Certificate[0]))
}

func firstOr(c tls.Certificate) []byte {
	if len(c.Certificate) == 0 {
		return nil
	}
	return c.Certificate[0]
}

// A disconnect counts once when the agent stops answering, not on every look.
func TestRetriableClientManager_CountsAgentDisconnectOnce(t *testing.T) {
	analytics.Default.Take()
	closers := map[string]*closeCounter{}
	m := newRetriableClientManager(nil, nil, time.Second, tls.Certificate{},
		stubDialer(map[string]container.Host{"nas:7007": {ID: "nas-id", Type: "agent"}}, nil, closers))
	_, err := m.AddAgent(t.Context(), "nas:7007", nil)
	require.NoError(t, err)

	m.Hosts(context.Background())
	service, _ := m.Find("nas-id")
	service.(*stubService).err = errors.New("gone")
	m.Hosts(context.Background())
	m.Hosts(context.Background())

	assert.Equal(t, 1, analytics.Default.Take().Counts["agent.disconnect"])
}
