package agent

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/container/agent/pb"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/test/bufconn"
)

// historyService is an agent's client service whose host keeps update events,
// like the Docker service.
type historyService struct {
	*MockedClientService
	mu          sync.Mutex
	events      []container.ContainerUpdateEvent
	subscribers map[context.Context]chan<- container.ContainerUpdateEvent
}

func newHistoryService(events ...container.ContainerUpdateEvent) *historyService {
	return &historyService{
		MockedClientService: &MockedClientService{},
		events:              events,
		subscribers:         map[context.Context]chan<- container.ContainerUpdateEvent{},
	}
}

func (h *historyService) RecentUpdates() []container.ContainerUpdateEvent {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]container.ContainerUpdateEvent(nil), h.events...)
}

func (h *historyService) SubscribeUpdates(ctx context.Context, ch chan<- container.ContainerUpdateEvent) {
	h.mu.Lock()
	h.subscribers[ctx] = ch
	h.mu.Unlock()
	go func() {
		<-ctx.Done()
		h.mu.Lock()
		delete(h.subscribers, ctx)
		h.mu.Unlock()
	}()
}

func (h *historyService) subscriberCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.subscribers)
}

// record keeps e and, when live, sends it to every subscriber, as the store does.
func (h *historyService) record(e container.ContainerUpdateEvent, live bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.events = append(h.events, e)
	if !live {
		return
	}
	for _, ch := range h.subscribers {
		ch <- e
	}
}

// testAgent serves an agent over bufconn.
type testAgent struct {
	t        *testing.T
	register func(*grpc.Server)
	lis      atomic.Pointer[bufconn.Listener]
	server   *grpc.Server
}

func startTestAgent(t *testing.T, register func(*grpc.Server)) *testAgent {
	a := &testAgent{t: t, register: register}
	a.start()
	t.Cleanup(func() { a.server.Stop() })
	return a
}

func (a *testAgent) start() {
	pool := x509.NewCertPool()
	cert, err := x509.ParseCertificate(certs.Certificate[0])
	require.NoError(a.t, err)
	pool.AddCert(cert)
	creds := credentials.NewTLS(&tls.Config{Certificates: []tls.Certificate{certs}, ClientCAs: pool, ClientAuth: tls.RequireAndVerifyClientCert})
	lis := bufconn.Listen(bufSize)
	a.lis.Store(lis)
	a.server = grpc.NewServer(grpc.Creds(creds))
	a.register(a.server)
	go a.server.Serve(lis)
}

func (a *testAgent) client() *Client {
	c, err := NewClient("passthrough://bufnet", certs, grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
		return a.lis.Load().DialContext(ctx)
	}))
	require.NoError(a.t, err)
	a.t.Cleanup(func() { c.Close() })
	return c
}

func agentWithHistory(t *testing.T, h *historyService) *testAgent {
	return startTestAgent(t, func(s *grpc.Server) {
		pb.RegisterAgentServiceServer(s, newServer(h, "test", &mockNotificationHandler{}))
	})
}

// oldAgent is an agent from before the update RPCs: every method it does not
// know answers Unimplemented.
type oldAgent struct {
	pb.UnimplementedAgentServiceServer
}

func updateEvent(name, to string, at time.Time) container.ContainerUpdateEvent {
	return container.ContainerUpdateEvent{
		Host:         "agent-host",
		Name:         name + "-display",
		EngineName:   name,
		OldID:        "old-" + name,
		NewID:        "new-" + name,
		FromRef:      "nginx:latest",
		ToRef:        "nginx:latest",
		FromDigest:   "nginx@sha256:aaa",
		ToDigest:     "nginx@sha256:" + to,
		FromImageID:  "sha256:from",
		ToImageID:    "sha256:" + to,
		OldStartedAt: at.Add(-time.Hour),
		At:           at,
		Source:       container.UpdateSourceSchedule,
		RunID:        "run-1",
	}
}

func receive(t *testing.T, ch <-chan container.ContainerUpdateEvent) container.ContainerUpdateEvent {
	t.Helper()
	select {
	case e := <-ch:
		return e
	case <-time.After(10 * time.Second):
		t.Fatal("no update event")
		return container.ContainerUpdateEvent{}
	}
}

func assertNothing(t *testing.T, ch <-chan container.ContainerUpdateEvent) {
	t.Helper()
	select {
	case e := <-ch:
		t.Fatalf("unexpected update event %+v", e)
	case <-time.After(200 * time.Millisecond):
	}
}

// agentService is the server's service for an agent, retrying fast, and
// waiting at cleanup for its subscriptions to end.
func agentService(t *testing.T, c *Client) *service {
	svc := NewService(c).(*service)
	svc.updateRetry = 50 * time.Millisecond
	svc.noHistoryRetry = 50 * time.Millisecond
	t.Cleanup(svc.updateStreams.Wait)
	return svc
}

// The server reads an agent host's update events, every field intact, through
// the same UpdateHistory a local host has.
func TestRecentUpdatesFromAgent(t *testing.T) {
	at := time.Date(2026, 10, 3, 4, 0, 0, 0, time.UTC)
	first := updateEvent("web", "bbb", at)
	rolled := updateEvent("db", "ccc", at.Add(time.Minute))
	rolled.RolledBack = true
	rolled.OldStartedAt = time.Time{}
	rolled.RunID = ""
	h := newHistoryService(first, rolled)

	svc := NewService(agentWithHistory(t, h).client())
	history, ok := svc.(container.UpdateHistory)
	require.True(t, ok, "an agent host keeps update history like a local one")
	got := history.RecentUpdates()
	assert.Equal(t, []container.ContainerUpdateEvent{first, rolled}, got)
	assert.True(t, got[1].OldStartedAt.IsZero(), "an unset time stays zero, not the epoch")
}

// A subscriber gets what happens from now on, not what the agent already had.
func TestSubscribeUpdatesFromAgent(t *testing.T) {
	at := time.Date(2026, 10, 3, 4, 0, 0, 0, time.UTC)
	h := newHistoryService(updateEvent("old", "aaa", at))
	svc := agentService(t, agentWithHistory(t, h).client())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := make(chan container.ContainerUpdateEvent, 4)
	svc.SubscribeUpdates(ctx, ch)
	require.Eventually(t, func() bool { return h.subscriberCount() == 1 }, 5*time.Second, 10*time.Millisecond)

	next := updateEvent("web", "bbb", at.Add(time.Minute))
	h.record(next, true)
	assert.Equal(t, next, receive(t, ch))
	assertNothing(t, ch)

	cancel()
	require.Eventually(t, func() bool { return h.subscriberCount() == 0 }, 5*time.Second, 10*time.Millisecond, "the agent's subscription ends with the server's")
}

// When the stream breaks, it comes back on its own, and what the agent kept
// meanwhile is sent once, without repeating what was already sent.
func TestSubscribeUpdatesCatchesUpAfterReconnect(t *testing.T) {
	at := time.Date(2026, 10, 3, 4, 0, 0, 0, time.UTC)
	h := newHistoryService()
	agent := agentWithHistory(t, h)
	svc := agentService(t, agent.client())

	ctx := t.Context()
	ch := make(chan container.ContainerUpdateEvent, 4)
	svc.SubscribeUpdates(ctx, ch)
	require.Eventually(t, func() bool { return h.subscriberCount() == 1 }, 5*time.Second, 10*time.Millisecond)

	first := updateEvent("web", "bbb", at)
	h.record(first, true)
	assert.Equal(t, first, receive(t, ch))

	agent.server.Stop()
	require.Eventually(t, func() bool { return h.subscriberCount() == 0 }, 5*time.Second, 10*time.Millisecond)
	missed := updateEvent("db", "ccc", at.Add(time.Minute))
	h.record(missed, false)
	agent.start()

	assert.Equal(t, missed, receive(t, ch), "the event kept while the stream was down")
	require.Eventually(t, func() bool { return h.subscriberCount() == 1 }, 10*time.Second, 10*time.Millisecond)
	assertNothing(t, ch)

	live := updateEvent("cache", "ddd", at.Add(2*time.Minute))
	h.record(live, true)
	assert.Equal(t, live, receive(t, ch))
}

// An agent older than the update RPCs has no history: reads are empty, the
// subscription sends nothing, and that is logged once, at debug.
func TestOldAgentHasNoUpdateHistory(t *testing.T) {
	var buf bytes.Buffer
	prevLogger, prevLevel := log.Logger, zerolog.GlobalLevel()
	log.Logger = zerolog.New(&syncWriter{w: &buf})
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	t.Cleanup(func() { log.Logger = prevLogger; zerolog.SetGlobalLevel(prevLevel) })

	agent := startTestAgent(t, func(s *grpc.Server) { pb.RegisterAgentServiceServer(s, &oldAgent{}) })
	client := agent.client()

	_, err := client.RecentUpdates(context.Background())
	require.ErrorIs(t, err, ErrNoUpdateHistory)

	svc := agentService(t, client)
	assert.Empty(t, svc.RecentUpdates())
	assert.Empty(t, svc.RecentUpdates())

	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan container.ContainerUpdateEvent, 1)
	svc.SubscribeUpdates(ctx, ch)
	assertNothing(t, ch) // several retries at 50ms
	cancel()
	svc.updateStreams.Wait()

	logs := buf.String()
	assert.Equal(t, 1, strings.Count(logs, "agent keeps no update history"), logs)
	assert.Contains(t, logs, `"level":"debug"`)
	assert.NotContains(t, logs, "could not read update events")
}

// A service without a history (not a Docker host) answers like an old agent.
func TestAgentWithoutHistoryAnswersUnimplemented(t *testing.T) {
	agent := startTestAgent(t, func(s *grpc.Server) {
		pb.RegisterAgentServiceServer(s, newServer(&MockedClientService{}, "test", &mockNotificationHandler{}))
	})
	_, err := agent.client().RecentUpdates(context.Background())
	require.ErrorIs(t, err, ErrNoUpdateHistory)
}

type syncWriter struct {
	mu sync.Mutex
	w  *bytes.Buffer
}

func (s *syncWriter) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.w.Write(p)
}

// A container Watchtower updated on an agent host has a rollback target the
// server finds the same way as for a local container: from the agent's events,
// over the labels Watchtower copied.
func TestRollbackTargetFromAgentHistory(t *testing.T) {
	e := updateEvent("web", "bbb", time.Date(2026, 10, 3, 4, 0, 0, 0, time.UTC))
	e.NewID = "0123456789ab"
	e.Source = container.UpdateSourceWatchtower
	svc := NewService(agentWithHistory(t, newHistoryService(e)).client())

	c := container.Container{ID: e.NewID, ImageID: e.ToImageID, Labels: map[string]string{
		container.PreviousImageLabel: "sha256:stale",
		container.UpdateSourceLabel:  container.UpdateSourceRollback,
	}}
	target, err := container.NewContainerService(svc, c).RollbackTarget()
	require.NoError(t, err)
	assert.Equal(t, container.RollbackTarget{ImageID: "sha256:from", Ref: "nginx@sha256:aaa"}, target)
}
