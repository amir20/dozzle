package web

import (
	"crypto/tls"
	"net/http"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/analytics"
	"github.com/amir20/dozzle/internal/config"
	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/container/docker"
	"github.com/amir20/dozzle/internal/hostservice"
	"github.com/amir20/dozzle/internal/notification"
	"github.com/amir20/dozzle/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// alertHosts reports a fixed set of rules and destinations, so the beacon can be
// checked without starting a notification manager.
type alertHosts struct {
	*hostservice.MultiHostService
	subs  []*notification.Subscription
	dests []notification.DispatcherConfig
}

// downHosts replaces the host list, to put hosts of each kind offline.
type downHosts struct {
	*alertHosts
	hosts []container.Host
}

func (d *downHosts) Hosts() []container.Host { return d.hosts }

func (a *alertHosts) Subscriptions() []*notification.Subscription  { return a.subs }
func (a *alertHosts) Dispatchers() []notification.DispatcherConfig { return a.dests }

func beaconHandler(t *testing.T, cfg Config) *handler {
	t.Helper()
	client := new(MockedClient)
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]container.Container{}, nil)
	client.On("Host").Return(container.Host{ID: "localhost", Type: "local"})
	client.On("ContainerEvents", mock.Anything, mock.AnythingOfType("chan<- container.ContainerEvent")).Return(nil)
	manager := hostservice.NewRetriableClientManager(nil, nil, time.Second, tls.Certificate{}, docker.NewService(client, container.ContainerLabels{}))
	return &handler{hostService: hostservice.NewMultiHostService(manager, time.Second), config: &cfg}
}

func TestBeaconFacts(t *testing.T) {
	setupTestEnv(t, true)
	h := beaconHandler(t, Config{Base: "/", Addr: ":8080", Version: "v1", Mode: "server",
		Authorization: Authorization{Provider: NONE},
		Beacon:        types.BeaconEvent{Mode: "server", Users: "2-5"}})

	h.hostService = &alertHosts{
		MultiHostService: h.hostService.(*hostservice.MultiHostService),
		subs: []*notification.Subscription{
			{Name: "a", LogExpression: `level == "error"`},
			{Name: "b", EventExpression: `name == "die"`},
			{Name: "c", MetricExpression: `cpu > 90`},
		},
		dests: []notification.DispatcherConfig{{Type: "webhook"}, {Type: "webhook"}, {Type: "cloud"}},
	}

	containers := []container.Container{
		{ID: "1", Labels: map[string]string{"dev.dozzle.name": "x", "dev.dozzle.url": "https://example.com"}},
		{ID: "2", Labels: map[string]string{"dev.dozzle.group": "g"}},
		{ID: "3"},
	}
	b := h.beaconFacts(containers, setupConfigPath)

	assert.Equal(t, map[string]int{"local": 1}, b.HostsByType)
	assert.Equal(t, 0, b.AgentsDown)
	assert.Equal(t, 1, b.Clients)
	assert.Equal(t, map[string]int{"log": 1, "event": 1, "metric": 1}, b.AlertRules)
	assert.Equal(t, map[string]int{"webhook": 2, "cloud": 1}, b.Destinations)
	assert.False(t, b.CloudLinked)
	assert.Equal(t, "off", b.AutoUpdate)
	require.NotNil(t, b.ContainersTotal)
	assert.Equal(t, 3, *b.ContainersTotal)
	assert.Equal(t, map[string]int{"name": 1, "url": 1, "group": 1}, b.Labels)
	assert.Equal(t, "2-5", b.Users)
	assert.Equal(t, "server", b.Mode)

	// Not listed, or a host failed to list: no container facts at all.
	b = h.beaconFacts(nil, setupConfigPath)
	assert.Nil(t, b.ContainersTotal)
	assert.Nil(t, b.Labels)
}

func TestReportUsage(t *testing.T) {
	setupTestEnv(t, true)
	analytics.Default.Take()
	h := createRouter(beaconHandler(t, Config{Base: "/", Authorization: Authorization{Provider: NONE}}))

	rr := doSetup(h, "POST", "/api/usage", `{"counts":{"logs.sql":3,"palette.open":99999,"container.secret-name":5,"action.remove":7,"notify.log":2},"locale":"de","activeMinutes":5000}`)
	require.Equal(t, http.StatusNoContent, rr.Code)

	s := analytics.Default.Take()
	// Server-side counters like action.remove can't be reported by a browser.
	assert.Equal(t, map[string]int{"logs.sql": 3, "palette.open": maxUsagePerRequest}, s.Counts)
	assert.Equal(t, map[string]int{"de": 1}, s.Locales)
	assert.Equal(t, maxUsageMinutesPerRequest, s.Minutes, "one report covers at most a day")

	assert.Equal(t, http.StatusBadRequest, doSetup(h, "POST", "/api/usage", `not json`).Code)
	// A cross-site form can only send text/plain without a preflight.
	rr = doSetup(h, "POST", "/api/usage", `{"counts":{"logs.sql":3}}`, "Content-Type", "text/plain")
	assert.Equal(t, http.StatusUnsupportedMediaType, rr.Code)
	assert.True(t, analytics.Default.Take().Empty())
}

func TestReportUsageIgnoredWithNoAnalytics(t *testing.T) {
	setupTestEnv(t, true)
	analytics.Default.Take()
	h := createRouter(beaconHandler(t, Config{Base: "/", NoAnalytics: true, Authorization: Authorization{Provider: NONE}}))

	rr := doSetup(h, "POST", "/api/usage", `{"counts":{"logs.sql":3}}`)
	assert.Equal(t, http.StatusNoContent, rr.Code)
	assert.True(t, analytics.Default.Take().Empty())
}

func TestFlushUsageSendsWhatIsCounted(t *testing.T) {
	setupTestEnv(t, true)
	analytics.Default.Take()
	var sent []types.BeaconEvent
	old := sendBeacon
	sendBeacon = func(b types.BeaconEvent) error { sent = append(sent, b); return nil }
	t.Cleanup(func() { sendBeacon = old })

	h := beaconHandler(t, Config{Base: "/", Authorization: Authorization{Provider: NONE}})
	h.flushUsage()
	assert.Empty(t, sent, "nothing counted, nothing sent")

	analytics.Count("logs.download")
	h.flushUsage()
	require.Len(t, sent, 1)
	assert.Equal(t, "usage", sent[0].Name)
	assert.Equal(t, 1, sent[0].Usage["logs.download"])
}

// Every tab open and reconnect asks for an events beacon; only the first in a
// while sends one.
func TestSendBeaconEventIsThrottled(t *testing.T) {
	setupTestEnv(t, true)
	var sent []types.BeaconEvent
	old, oldLast := sendBeacon, eventsBeaconLast.Load()
	sendBeacon = func(b types.BeaconEvent) error { sent = append(sent, b); return nil }
	eventsBeaconLast.Store(0)
	t.Cleanup(func() { sendBeacon = old; eventsBeaconLast.Store(oldLast) })

	h := beaconHandler(t, Config{Base: "/", Mode: "server", Authorization: Authorization{Provider: NONE}})
	sendBeaconEvent(h, "ua", nil, 0, setupConfigPath)
	sendBeaconEvent(h, "ua", nil, 0, setupConfigPath)
	require.Len(t, sent, 1)
	assert.Equal(t, "events", sent[0].Name)
	assert.Equal(t, "localhost", sent[0].ServerID)

	eventsBeaconLast.Store(time.Now().Add(-eventsBeaconInterval).UnixNano())
	sendBeaconEvent(h, "ua", nil, 0, setupConfigPath)
	assert.Len(t, sent, 2)
}

// The scheduler only runs in server mode, so a schedule elsewhere is not one.
func TestBeaconFactsAutoUpdateOnlyInServerMode(t *testing.T) {
	setupTestEnv(t, true)
	daily := "daily"
	setup := SetupConfig{AutoUpdateMode: &daily}
	h := beaconHandler(t, Config{Base: "/", Mode: "server", Setup: setup, Authorization: Authorization{Provider: NONE}})
	assert.Equal(t, "daily", h.beaconFacts(nil, setupConfigPath).AutoUpdate)

	h = beaconHandler(t, Config{Base: "/", Mode: "swarm", Setup: setup, Authorization: Authorization{Provider: NONE}})
	assert.Empty(t, h.beaconFacts(nil, setupConfigPath).AutoUpdate)
}

func TestFileAgentCounts(t *testing.T) {
	file := config.File{
		RemoteAgents:  []string{"nas:7007|nas", " pi:7007 ", "env:7007|renamed", "nas:7007", "", "a|b|c|d"},
		PrivateAgents: []string{"pi:7007"},
	}
	agents, private := fileAgentCounts([]string{"env:7007"}, file)
	assert.Equal(t, 2, agents)
	assert.Equal(t, 1, private)
}

func TestSetupAgentsCountsOutcome(t *testing.T) {
	setupTestEnv(t, true)
	analytics.Default.Take()
	hosts := &fakeAgentHosts{connected: map[string]string{}}
	h := agentsHandler(hosts, SetupConfig{})

	require.Equal(t, http.StatusCreated, doSetup(h, "POST", "/api/setup/agents", `{"address":"nas:7007"}`).Code)
	require.Equal(t, http.StatusConflict, doSetup(h, "POST", "/api/setup/agents", `{"address":"nas:7007"}`).Code)

	s := analytics.Default.Take()
	assert.Equal(t, 1, s.Counts["host.add.ok"])
	assert.Equal(t, 1, s.Counts["host.add.duplicate"])
}

// Only agents count toward agentsDown: a local engine or a remote socket being
// unreachable is a different problem.
func TestBeaconFactsAgentsDownCountsOnlyAgents(t *testing.T) {
	setupTestEnv(t, true)
	h := beaconHandler(t, Config{Base: "/", Authorization: Authorization{Provider: NONE}})
	h.hostService = &downHosts{
		alertHosts: &alertHosts{MultiHostService: h.hostService.(*hostservice.MultiHostService)},
		hosts: []container.Host{
			{ID: "a", Type: "agent", Available: false},
			{ID: "b", Type: "agent", Available: true},
			{ID: "c", Type: "local", Available: false},
			{ID: "d", Type: "remote", Available: false},
		},
	}
	b := h.beaconFacts(nil, setupConfigPath)
	assert.Equal(t, 1, b.AgentsDown)
	assert.Equal(t, map[string]int{"agent": 2, "local": 1, "remote": 1}, b.HostsByType)
}
