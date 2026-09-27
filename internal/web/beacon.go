package web

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/amir20/dozzle/internal/analytics"
	"github.com/amir20/dozzle/internal/config"
	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/container/agent"
	"github.com/amir20/dozzle/types"
	"github.com/rs/zerolog/log"
)

// Seams for tests.
var (
	usageBeaconInterval = 24 * time.Hour
	sendBeacon          = analytics.SendBeacon
)

// dozzleLabels are the labels the beacon counts containers by, keyed by the
// name the beacon reports them under.
var dozzleLabels = map[string]string{
	"name":  "dev.dozzle.name",
	"group": "dev.dozzle.group",
	"url":   "dev.dozzle.url",
	"icon":  "dev.dozzle.icon",
}

var knownHostTypes = map[string]bool{"local": true, "agent": true, "remote": true, "swarm": true, "k8s": true}

// beaconFacts is what the events and usage beacons say about this install right
// now: how it is set up, never what is in it. containers is nil when the caller
// has not listed them, or when a host failed to list: a partial count would read
// as a smaller install, so the container facts are left out instead. configPath
// is dozzle.yml, passed in because the path is a test seam and callers run this
// on their own goroutine.
func (h *handler) beaconFacts(containers []container.Container, configPath string) types.BeaconEvent {
	// Starts from the install facts the start beacon carries, so the dashboard,
	// which reads these rows, stops seeing agents and shell as always off.
	b := h.config.Beacon
	b.AuthProvider = string(h.config.Authorization.Provider)
	b.HasActions = h.config.EnableActions
	b.HasShell = h.config.EnableShell
	b.HasCustomAddress = h.config.Addr != ":8080"
	b.HasCustomBase = h.config.Base != "/"
	b.HasHostname = h.config.Hostname != ""
	b.Version = h.config.Version

	hosts := h.hostService.Hosts()
	b.Clients = len(hosts)
	b.HostsByType = map[string]int{}
	b.AgentsDown = 0
	b.ServerID = ""
	for _, host := range hosts {
		if knownHostTypes[host.Type] {
			b.HostsByType[host.Type]++
		}
		if host.Type == "agent" && !host.Available {
			b.AgentsDown++
		}
		if host.Type == "local" && b.ServerID == "" {
			b.ServerID = host.ID
		}
	}
	// Swarm marks every host "swarm", so only then ask for the local one, which
	// lists the hosts again.
	if b.ServerID == "" {
		if local, err := h.hostService.LocalHost(); err == nil {
			b.ServerID = local.ID
		}
	}

	// Agents added from the UI since startup count too, the same way startup
	// reads them: trimmed, deduplicated, and never one the env var already has.
	if file, err := config.Load(configPath); err == nil {
		// Only a mode that can add agents dials the ones in dozzle.yml.
		b.FileAgents, b.PrivateAgents = 0, 0
		if _, ok := h.agentService(); ok {
			b.FileAgents, b.PrivateAgents = fileAgentCounts(h.config.Setup.EnvAgents, file)
		}
		// The scheduler only runs in server mode, so elsewhere the file's schedule
		// does nothing.
		if h.config.Mode == "server" {
			b.AutoUpdate = autoUpdateFrom(h.config.Setup, file).Mode
		}
	}

	b.CloudLinked = h.hostService.CloudConfig() != nil
	b.AlertRules = map[string]int{}
	for _, sub := range h.hostService.Subscriptions() {
		// By expression rather than IsLogAlert and friends, which also need the
		// compiled program: a rule that failed to compile is still a rule.
		switch {
		case sub.EventExpression != "":
			b.AlertRules["event"]++
		case sub.MetricExpression != "":
			b.AlertRules["metric"]++
		case sub.LogExpression != "":
			b.AlertRules["log"]++
		}
	}
	b.Destinations = map[string]int{}
	for _, d := range h.hostService.Dispatchers() {
		if d.Type != "" {
			b.Destinations[d.Type]++
		}
	}

	if containers != nil {
		total := len(containers)
		b.ContainersTotal = &total
		b.Labels = map[string]int{}
		for _, c := range containers {
			for key, label := range dozzleLabels {
				if c.Labels[label] != "" {
					b.Labels[key]++
				}
			}
		}
	}
	return b
}

// fileAgentCounts counts dozzle.yml's agents the way setupAgents lists them,
// without asking the host service about each one.
func fileAgentCounts(envAgents []string, file config.File) (agents, private int) {
	seen := map[string]bool{}
	for _, endpoint := range envAgents {
		if address, _, _, err := agent.ParseEndpoint(endpoint); err == nil {
			seen[address] = true
		}
	}
	for _, endpoint := range file.RemoteAgents {
		endpoint = strings.TrimSpace(endpoint)
		address, _, _, err := agent.ParseEndpoint(endpoint)
		if endpoint == "" || err != nil || seen[address] {
			continue
		}
		seen[address] = true
		agents++
		if slices.ContainsFunc(file.PrivateAgents, sameEndpoint(endpoint)) {
			private++
		}
	}
	return agents, private
}

// eventsBeaconInFlight lets one events beacon run at a time. Every tab open and
// reconnect starts one, and each dials every host, so while the beacon endpoint
// or a host is slow they would otherwise pile up without bound.
var eventsBeaconInFlight atomic.Bool

// eventsBeaconInterval is the least time between two events beacons. A flapping
// connection or a room of open tabs reconnects far more often than the facts
// change.
var eventsBeaconInterval = 5 * time.Minute

// eventsBeaconLast is when the last events beacon was delivered, in unix nanoseconds.
var eventsBeaconLast atomic.Int64

// sendBeaconEvent sends the events beacon for a new events stream. containers is
// nil when the list is partial or filtered to one user's labels, since either
// would read as a smaller install; running is how many the stream listed.
func sendBeaconEvent(h *handler, userAgent string, containers []container.Container, running int, configPath string) {
	if h.config.NoAnalytics || !eventsBeaconInFlight.CompareAndSwap(false, true) {
		return
	}
	defer eventsBeaconInFlight.Store(false)
	now := time.Now()
	if last := eventsBeaconLast.Load(); last != 0 && now.Sub(time.Unix(0, last)) < eventsBeaconInterval {
		return
	}
	b := h.beaconFacts(containers, configPath)
	b.Name = "events"
	b.Browser = userAgent
	b.RunningContainers = running

	if err := sendBeacon(b); err != nil {
		log.Debug().Err(err).Msg("error sending beacon")
		return
	}
	// Only a beacon that went out starts the wait, so a failed one is retried
	// on the next stream. The in-flight flag above keeps tabs from racing.
	eventsBeaconLast.Store(now.UnixNano())
}

// runUsageBeacon sends the daily usage beacon until ctx ends.
func (h *handler) runUsageBeacon(ctx context.Context) {
	analytics.RunUsageBeacon(ctx, analytics.Default, usageBeaconInterval, h.usageFacts, sendBeacon)
}

func (h *handler) usageFacts() types.BeaconEvent {
	containers, errs := h.hostService.ListAllContainers(h.config.Labels)
	if len(errs) > 0 {
		containers = nil
	}
	return h.beaconFacts(containers, setupConfigPath)
}

// usageFlushTimeout bounds the last usage beacon, sent on the way out. Shutdown
// and a self-update both have somewhere to be.
var usageFlushTimeout = 3 * time.Second

// flushUsage sends the usage counted since the last beacon, if any, waiting at
// most usageFlushTimeout. Counters otherwise only leave on the 24h tick, so
// without this every restart would drop up to a day of them.
func (h *handler) flushUsage() {
	done := make(chan struct{})
	go func() {
		defer close(done)
		analytics.SendUsage(analytics.Default, h.usageFacts, sendBeacon)
	}()
	select {
	case <-done:
	case <-time.After(usageFlushTimeout):
		log.Debug().Msg("gave up waiting for the last usage beacon")
	}
}

// maxUsagePerRequest caps what one browser report can add to a counter, so a
// buggy or hostile tab cannot turn a counter into noise.
const maxUsagePerRequest = 1000

const maxUsageMinutesPerRequest = 24 * 60

type usageReport struct {
	Counts        map[string]int `json:"counts"`
	Locale        string         `json:"locale"`
	ActiveMinutes int            `json:"activeMinutes"`
}

// reportUsage takes the counts only the browser can see (searches, the command
// palette, the wizard). Unknown keys are dropped by the counter store.
func (h *handler) reportUsage(w http.ResponseWriter, r *http.Request) {
	if h.config.NoAnalytics {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	// decodeSetupBody takes only JSON, so a cross-site form cannot inflate the
	// counters.
	var report usageReport
	if !decodeSetupBody(w, r, &report) {
		return
	}
	for key, n := range report.Counts {
		if analytics.BrowserUsageKeys[key] {
			analytics.Default.Add(key, min(n, maxUsagePerRequest))
		}
	}
	if report.Locale != "" {
		analytics.Default.AddLocale(report.Locale)
	}
	// A tab keeps what a failed report carried and sends it with the next one (an
	// expired session, the hub restarting), so a report can span hours. More than
	// a day is still a bug.
	analytics.Default.AddMinutes(min(report.ActiveMinutes, maxUsageMinutesPerRequest))
	w.WriteHeader(http.StatusNoContent)
}
