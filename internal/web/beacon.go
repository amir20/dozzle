package web

import (
	"context"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/amir20/dozzle/internal/analytics"
	"github.com/amir20/dozzle/internal/config"
	"github.com/amir20/dozzle/internal/container"
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
	for _, host := range hosts {
		if knownHostTypes[host.Type] {
			b.HostsByType[host.Type]++
		}
		if host.Type == "agent" && !host.Available {
			b.AgentsDown++
		}
	}

	// Agents added from the UI since startup count too, the same way startup
	// reads them: trimmed, deduplicated, and never one the env var already has.
	if file, err := config.Load(configPath); err == nil {
		b.FileAgents, b.PrivateAgents = 0, 0
		for _, a := range h.setupAgents(file) {
			if a.Locked {
				continue
			}
			b.FileAgents++
			if a.Private {
				b.PrivateAgents++
			}
		}
	}
	if settings, err := effectiveAutoUpdateAt(h.config.Setup, configPath); err == nil {
		b.AutoUpdate = settings.Mode
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

	if local, err := h.hostService.LocalHost(); err == nil {
		b.ServerID = local.ID
	}
	return b
}

func sendBeaconEvent(h *handler, r *http.Request, containers []container.Container, complete bool, configPath string) {
	if h.config.NoAnalytics {
		return
	}
	var listed []container.Container
	if complete {
		listed = containers
	}
	b := h.beaconFacts(listed, configPath)
	b.Name = "events"
	b.Browser = r.Header.Get("User-Agent")
	b.RunningContainers = len(containers)

	if err := sendBeacon(b); err != nil {
		log.Debug().Err(err).Msg("error sending beacon")
	}
}

// runUsageBeacon sends the daily usage beacon for the life of the process.
func (h *handler) runUsageBeacon(ctx context.Context) {
	if h.config.NoAnalytics {
		return
	}
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

// usageFlusher is set by CreateServer so main and the self-update path, which
// have no handler, can send what is counted before the process goes away.
var usageFlusher atomic.Pointer[func()]

// FlushUsage sends the usage counted since the last beacon, if any, waiting at
// most usageFlushTimeout. Counters otherwise only leave on the 24h tick, so
// without this every restart would drop up to a day of them.
func FlushUsage() {
	if f := usageFlusher.Load(); f != nil {
		(*f)()
	}
}

func (h *handler) flushUsage() {
	if h.config.NoAnalytics {
		return
	}
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
	// A report covers at most one flush interval, so more than an hour is a bug.
	analytics.Default.AddMinutes(min(report.ActiveMinutes, 60))
	w.WriteHeader(http.StatusNoContent)
}
