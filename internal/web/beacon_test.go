package web

import (
	"crypto/tls"
	"net/http"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/analytics"
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

func (a *alertHosts) Subscriptions() []*notification.Subscription  { return a.subs }
func (a *alertHosts) Dispatchers() []notification.DispatcherConfig { return a.dests }

func beaconHandler(t *testing.T, cfg Config) *handler {
	t.Helper()
	client := new(MockedClient)
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]container.Container{}, nil)
	client.On("Host").Return(container.Host{ID: "localhost", Type: "local"})
	client.On("ContainerEvents", mock.Anything, mock.AnythingOfType("chan<- container.ContainerEvent")).Return(nil)
	manager := hostservice.NewRetriableClientManager(nil, time.Second, tls.Certificate{}, docker.NewService(client, container.ContainerLabels{}))
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
	assert.Equal(t, 3, b.ContainersTotal)
	assert.Equal(t, map[string]int{"name": 1, "url": 1, "group": 1}, b.Labels)
	assert.Equal(t, "2-5", b.Users)
	assert.Equal(t, "server", b.Mode)
}

func TestReportUsage(t *testing.T) {
	setupTestEnv(t, true)
	analytics.Default.Take()
	h := createRouter(beaconHandler(t, Config{Base: "/", Authorization: Authorization{Provider: NONE}}))

	rr := doSetup(h, "POST", "/api/usage", `{"counts":{"logs.sql":3,"palette.open":99999,"container.secret-name":5},"locale":"de","activeMinutes":500}`)
	require.Equal(t, http.StatusNoContent, rr.Code)

	s := analytics.Default.Take()
	assert.Equal(t, map[string]int{"logs.sql": 3, "palette.open": maxUsagePerRequest}, s.Counts)
	assert.Equal(t, map[string]int{"de": 1}, s.Locales)
	assert.Equal(t, 60, s.Minutes, "one report covers at most an hour")

	assert.Equal(t, http.StatusBadRequest, doSetup(h, "POST", "/api/usage", `not json`).Code)
}

func TestReportUsageIgnoredWithNoAnalytics(t *testing.T) {
	setupTestEnv(t, true)
	analytics.Default.Take()
	h := createRouter(beaconHandler(t, Config{Base: "/", NoAnalytics: true, Authorization: Authorization{Provider: NONE}}))

	rr := doSetup(h, "POST", "/api/usage", `{"counts":{"logs.sql":3}}`)
	assert.Equal(t, http.StatusNoContent, rr.Code)
	assert.True(t, analytics.Default.Take().Empty())
}

func TestAddHostOutcome(t *testing.T) {
	cases := map[string]struct {
		status int
		body   string
	}{
		"host.add.ok":        {http.StatusCreated, ""},
		"host.add.duplicate": {http.StatusConflict, "this agent is already added"},
		"host.add.cert":      {http.StatusBadGateway, "could not connect to agent: tls: unknown certificate authority"},
		"host.add.timeout":   {http.StatusBadGateway, "could not connect to agent: context deadline exceeded"},
		"host.add.refused":   {http.StatusBadGateway, "could not connect to agent: connection refused"},
		"host.add.other":     {http.StatusBadRequest, "address must be host:port"},
	}
	for want, c := range cases {
		assert.Equal(t, want, addHostOutcome(c.status, c.body))
	}
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
