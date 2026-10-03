package web

import (
	"context"
	"errors"
	"maps"
	"net"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/hostservice"
	"github.com/amir20/dozzle/internal/web/sse"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	eventBufferSize = 64
	statBufferSize  = 128
	// minimum gap between retries of a host whose container list failed to refresh
	staleHostRetryInterval = 5 * time.Second
	// keeps proxies from closing an otherwise silent stream
	keepAliveInterval = 20 * time.Second
	// how long the browser waits before reconnecting a dropped stream
	reconnectDelay = 3 * time.Second
	// minimum gap between host-id reconciliations, so a burst of reconnecting tabs
	// dials every agent once rather than once each
	hostReconcileInterval = 10 * time.Second
	// how often live host metrics are re-sent to a watching client
	hostMetricRefreshInterval = 15 * time.Second
)

func (h *handler) streamEvents(w http.ResponseWriter, r *http.Request) {
	sseWriter, err := sse.NewWriter(r.Context(), w, r)
	if err != nil {
		log.Error().Err(err).Msg("error creating sse writer")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer sseWriter.Close()

	if err := sseWriter.Retry(reconnectDelay); err != nil {
		log.Debug().Err(err).Msg("error writing retry to event stream")
		return
	}

	// buffered so a momentarily slow client can't stall the shared per-host store loop,
	// which broadcasts to every subscriber with a blocking send
	events := make(chan container.ContainerEvent, eventBufferSize)
	stats := make(chan container.ContainerStat, statBufferSize)
	availableHosts := make(chan container.Host)
	hostMetricsUpdates := make(chan []hostMetricsEvent, 1)

	h.hostService.SubscribeEventsAndStats(container.WithSubscriberName(r.Context(), "sse-events"), events, stats)
	h.hostService.SubscribeAvailableHosts(r.Context(), availableHosts)

	// One shared ticker feeds every tab; this just registers this stream's queue
	// and drops it on the way out.
	unsubscribeHostMetrics := h.subscribeHostMetrics(hostMetricsUpdates)
	defer unsubscribeHostMetrics()

	// An agent mints its id when its process starts, so one that restarted since the
	// hub last looked answers under an id nothing here is keyed by. Hosts() repairs
	// that and publishes the new host to the subscription above, which reaches this
	// stream as an update-host carrying replacesId. Without this, a dashboard that was
	// already open when the agent restarted keeps the stale host until someone reloads.
	h.reconcileHosts()

	userLabels := h.config.Labels
	if h.config.Authorization.Provider != NONE {
		user := auth.UserFromContext(r.Context())
		if user.ContainerLabels.Exists() {
			userLabels = user.ContainerLabels
		}
	}

	allContainers, errors := h.hostService.ListAllContainers(userLabels)

	// per-host set of container IDs the caller may see, so stat/event channels stay filtered like the list
	visibleByHost := make(map[string]map[string]struct{})
	setVisible := func(host string, containers []container.Container) {
		ids := make(map[string]struct{}, len(containers))
		for _, c := range containers {
			ids[c.ID] = struct{}{}
		}
		visibleByHost[host] = ids
	}

	// A host whose refresh failed keeps a set that is missing containers started since,
	// and nothing else repopulates it. Without a retry those containers stay invisible for
	// the life of this stream, so the client never sees them start, stop or update again.
	// Retries are driven by the host's own traffic and throttled, so a host that is down
	// and silent is never polled and can't block this loop on every attempt.
	staleHosts := make(map[string]time.Time) // host -> last failed attempt
	refreshHost := func(host string) ([]container.Container, bool) {
		containers, err := h.hostService.ListContainersForHost(host, userLabels)
		if err != nil {
			log.Warn().Err(err).Str("host", host).Msg("failed to refresh containers, will retry")
			staleHosts[host] = time.Now()
			return nil, false
		}
		delete(staleHosts, host)
		setVisible(host, containers)
		return containers, true
	}

	// A repair lists containers on a host that just failed, so it can take the whole
	// --timeout (10s by default) to come back. Run off the loop below: inline, one stale
	// host plus a container with a 5s healthcheck kept this handler inside a Docker call
	// more often than not, and every event it did not read in the meantime was dropped by
	// the store for good.
	type hostRefresh struct {
		host       string
		containers []container.Container
		err        error
	}
	refreshes := make(chan hostRefresh, 8)
	repairing := make(map[string]bool)
	// requestRepair starts a throttled background retry of a host whose list failed
	requestRepair := func(host string) {
		last, stale := staleHosts[host]
		if !stale || repairing[host] || time.Since(last) < staleHostRetryInterval {
			return
		}
		repairing[host] = true
		go func() {
			containers, err := h.hostService.ListContainersForHost(host, userLabels)
			select {
			case refreshes <- hostRefresh{host: host, containers: containers, err: err}:
			case <-r.Context().Done():
			}
		}()
	}
	isVisible := func(host, id string) bool {
		if host != "" {
			ids, ok := visibleByHost[host]
			if !ok {
				return false
			}
			_, ok = ids[id]
			return ok
		}
		// container-stat payloads carry no host, so fall back to scanning all hosts
		for _, ids := range visibleByHost {
			if _, ok := ids[id]; ok {
				return true
			}
		}
		return false
	}

	for _, c := range allContainers {
		ids, ok := visibleByHost[c.Host]
		if !ok {
			ids = make(map[string]struct{})
			visibleByHost[c.Host] = ids
		}
		ids[c.ID] = struct{}{}
	}

	for _, err := range errors {
		log.Warn().Err(err).Msg("error listing containers")
		if hostNotAvailableError, ok := err.(*hostservice.HostUnavailableError); ok {
			// this host has no visible set at all, so retry as soon as it produces traffic
			staleHosts[hostNotAvailableError.Host.ID] = time.Time{}
			if err := sseWriter.Event("update-host", hostNotAvailableError.Host); err != nil {
				logWriteError(err, "error writing event to event stream")
			}
		}
	}

	// sent on every (re)connect so a long-lived tab can tell it is running UI from an older build
	if err := sseWriter.Event("server-version", map[string]string{"version": h.config.Version}); err != nil {
		logWriteError(err, "error writing version to event stream")
	}

	if err := sseWriter.Event("containers-changed", allContainers); err != nil {
		logWriteError(err, "error writing containers to event stream")
	}

	// The path is read here, not in the goroutine: it is a test seam tests swap.
	var beaconContainers []container.Container
	if len(errors) == 0 && maps.EqualFunc(userLabels, h.config.Labels, slices.Equal[[]string]) {
		beaconContainers = allContainers
	}
	go sendBeaconEvent(h, r.UserAgent(), beaconContainers, len(allContainers), setupConfigPath)

	// a host whose containers are all filtered out or stopped emits no stats, so without
	// this the stream is silent and an idle proxy timeout (nginx defaults to 60s) drops it
	ticker := time.NewTicker(keepAliveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// a named event rather than a comment, so the UI can tell a connection that a
			// sleeping machine left half-open from one that is merely quiet
			if err := sseWriter.Event("ping", struct{}{}); err != nil {
				log.Debug().Err(err).Msg("error writing keep-alive to event stream")
				return
			}
		case refresh := <-refreshes:
			delete(repairing, refresh.host)
			if refresh.err != nil {
				// a start/rename may have repaired this host while the retry was in flight
				if _, stale := staleHosts[refresh.host]; !stale {
					continue
				}
				log.Warn().Err(refresh.err).Str("host", refresh.host).Msg("failed to refresh containers, will retry")
				staleHosts[refresh.host] = time.Now()
				continue
			}
			// Same race the error branch above guards, and it matters more here: a
			// start or rename refreshes the host inline while this retry is in flight,
			// and that list is newer than the one we are holding. Applying ours would
			// hide a container that has already started -- from `visibleByHost`, so its
			// stats and events are gated out, and from the client, which treats every
			// `containers-changed` as authoritative for the hosts it names and drops
			// what is missing.
			if _, stale := staleHosts[refresh.host]; !stale {
				log.Debug().Str("host", refresh.host).Msg("discarding stale refresh, host already recovered")
				continue
			}
			delete(staleHosts, refresh.host)
			setVisible(refresh.host, refresh.containers)
			log.Debug().Str("host", refresh.host).Int("count", len(refresh.containers)).Msg("recovered stale host")
			if err := sseWriter.Event("containers-changed", refresh.containers); err != nil {
				logWriteError(err, "error writing containers to event stream")
				return
			}
		case host := <-availableHosts:
			// an agent that reconnected has no visible set yet; its first traffic fills it
			if _, ok := visibleByHost[host.ID]; host.Available && !ok {
				staleHosts[host.ID] = time.Time{}
			}
			if err := sseWriter.Event("update-host", host); err != nil {
				logWriteError(err, "error writing event to event stream")
				return
			}
		case stat := <-stats:
			if !isVisible("", stat.ID) {
				// an unknown ID may belong to a container a stale host never got to report.
				// a just-started container produces a stat every second, so this also covers
				// a host that is otherwise quiet. The repair lands on the refreshes channel,
				// so this stat is skipped and the next one a second later gets through.
				for host := range staleHosts {
					requestRepair(host)
				}
				continue
			}
			if err := sseWriter.Event("container-stat", stat); err != nil {
				logWriteError(err, "error writing event to event stream")
				return
			}
		case event, ok := <-events:
			if !ok {
				return
			}
			log.Trace().Str("event", event.Name).Str("id", event.ActorID).Msg("container event from store")

			// start/rename refresh on their own below; anything else from a stale host is a
			// chance to repair it
			if event.Name != "start" && event.Name != "rename" {
				requestRepair(event.Host)
			}

			switch event.Name {
			case "start", "die", "destroy", "rename", "pause", "unpause":
				var refreshed []container.Container
				if event.Name == "start" || event.Name == "rename" {
					if containers, ok := refreshHost(event.Host); ok {
						log.Debug().Str("host", event.Host).Int("count", len(containers)).Msg("updating containers for host")
						refreshed = containers
					}
				}

				// gate both containers-changed and the raw event so out-of-scope
				// containers don't leak via payload or as a timing side-channel
				if !isVisible(event.Host, event.ActorID) {
					continue
				}

				if refreshed != nil {
					if err := sseWriter.Event("containers-changed", refreshed); err != nil {
						logWriteError(err, "error writing containers to event stream")
						return
					}
				}

				if err := sseWriter.Event("container-event", event); err != nil {
					logWriteError(err, "error writing event to event stream")
					return
				}

			case "update":
				if event.Container == nil || !isVisible(event.Host, event.Container.ID) {
					continue
				}
				if err := sseWriter.Event("container-updated", event.Container); err != nil {
					logWriteError(err, "error writing event to event stream")
					return
				}
			case "health_status: healthy", "health_status: unhealthy":
				if !isVisible(event.Host, event.ActorID) {
					continue
				}
				healthy := "unhealthy"
				if event.Name == "health_status: healthy" {
					healthy = "healthy"
				}
				payload := map[string]string{
					"actorId": event.ActorID,
					"health":  healthy,
				}

				if err := sseWriter.Event("container-health", payload); err != nil {
					logWriteError(err, "error writing event to event stream")
					return
				}
			}
		case batch := <-hostMetricsUpdates:
			// Its own event rather than update-host: the raw client's Host() has no
			// Available and no mode-specific Type, and update-host replaces the whole
			// host, so reusing it would mark the host offline every tick.
			for _, metrics := range batch {
				if err := sseWriter.Event("host-metrics", metrics); err != nil {
					logWriteError(err, "error writing host metrics to event stream")
					return
				}
			}
		case <-r.Context().Done():
			return
		}
	}
}

// logWriteError logs a failed write at debug when the client is at fault (a write
// deadline, a reset or a broken pipe all surface as net.Error). Those are routine: the
// handler returns and the browser reconnects. Anything else, like a marshal failure, is ours.
func logWriteError(err error, msg string) {
	level := zerolog.ErrorLevel
	if _, ok := errors.AsType[net.Error](err); ok {
		level = zerolog.DebugLevel
	}
	log.WithLevel(level).Err(err).Msg(msg)
}

// reconcileHosts re-reads host ids off the back of a stream connecting, throttled and
// off this request so the first payload is never held behind a dial to a down agent.
// This is deliberately driven by a watching client rather than a ticker: dialing every
// agent on a timer is churn for a fleet nobody is looking at.
func (h *handler) reconcileHosts() {
	h.reconcileMu.Lock()
	if h.reconciling || (!h.reconciledAt.IsZero() && time.Since(h.reconciledAt) < hostReconcileInterval) {
		h.reconcileMu.Unlock()
		return
	}
	h.reconciling = true
	h.reconcileMu.Unlock()

	go func() {
		defer func() {
			h.reconcileMu.Lock()
			h.reconciling = false
			h.reconciledAt = time.Now()
			h.reconcileMu.Unlock()
		}()
		h.hostService.Hosts()
	}()
}

// subscribeHostMetrics registers a stream's queue for host metrics updates and
// returns the function that removes it. The first subscriber starts the one
// shared ticker, so N tabs cost one read of each host per interval rather than
// N reads the way a per-stream Hosts() call did.
func (h *handler) subscribeHostMetrics(ch chan []hostMetricsEvent) func() {
	h.hostMetricsOnce.Do(func() {
		h.hostMetricsSubs = make(map[chan []hostMetricsEvent]struct{})
		go h.broadcastHostMetrics()
	})
	h.hostMetricsMu.Lock()
	h.hostMetricsSubs[ch] = struct{}{}
	h.hostMetricsMu.Unlock()

	return func() {
		h.hostMetricsMu.Lock()
		delete(h.hostMetricsSubs, ch)
		h.hostMetricsMu.Unlock()
	}
}

func (h *handler) broadcastHostMetrics() {
	ticker := time.NewTicker(hostMetricRefreshInterval)
	defer ticker.Stop()
	for range ticker.C {
		h.hostMetricsMu.Lock()
		hasSubscribers := len(h.hostMetricsSubs) > 0
		h.hostMetricsMu.Unlock()
		if !hasSubscribers {
			continue
		}

		batch := h.collectHostMetrics()
		if len(batch) == 0 {
			continue
		}

		h.hostMetricsMu.Lock()
		for ch := range h.hostMetricsSubs {
			select {
			case ch <- batch:
			default:
				// A tab slow to drain keeps the values it has; it will get the next batch.
			}
		}
		h.hostMetricsMu.Unlock()
	}
}

// hostMetricsCallTimeout bounds one host's read, so an agent that stopped
// answering costs that host one tick rather than holding up every host.
const hostMetricsCallTimeout = 3 * time.Second

// collectHostMetrics reads every host's metrics in parallel: the local engine off
// this machine, and each agent over the connection it already holds. Hosts with
// nothing to show, an older agent or a remote engine, are left out.
func (h *handler) collectHostMetrics() []hostMetricsEvent {
	services := h.hostService.ClientServices(false)
	results := make([]*hostMetricsEvent, len(services))
	var wg sync.WaitGroup
	for i, service := range services {
		wg.Go(func() {
			ctx, cancel := context.WithTimeout(context.Background(), hostMetricsCallTimeout)
			defer cancel()
			host, err := service.Host(ctx)
			if err != nil || !hostHasMetrics(host) {
				return
			}
			event := newHostMetricsEvent(host)
			results[i] = &event
		})
	}
	wg.Wait()

	return collapseHostMetrics(results)
}

// collapseHostMetrics drops the hosts that had nothing to show and keeps one
// event per host id. In swarm mode a node can be reached both as the local
// client and as an agent; both report the same machine, so one event is enough.
func collapseHostMetrics(results []*hostMetricsEvent) []hostMetricsEvent {
	batch := make([]hostMetricsEvent, 0, len(results))
	seen := make(map[string]struct{}, len(results))
	for _, event := range results {
		if event == nil {
			continue
		}
		if _, dup := seen[event.ID]; dup {
			continue
		}
		seen[event.ID] = struct{}{}
		batch = append(batch, *event)
	}
	return batch
}

// hostMetricsEvent is the payload of the host-metrics SSE event: the host's id and
// its metric fields, nothing else, so the client merges it into the host it has.
// No omitempty, so a value that drops to zero overwrites the stale one.
type hostMetricsEvent struct {
	ID               string  `json:"id"`
	MetricsAvailable bool    `json:"metricsAvailable"`
	Load1            float64 `json:"load1"`
	Load5            float64 `json:"load5"`
	Load15           float64 `json:"load15"`
	Uptime           uint64  `json:"uptime"`
	DiskTotal        uint64  `json:"diskTotal"`
	DiskFree         uint64  `json:"diskFree"`
	// Always an array, never null, so a drive that was unmounted clears.
	Disks []container.Disk `json:"disks"`
}

func newHostMetricsEvent(host container.Host) hostMetricsEvent {
	disks := host.Disks
	if disks == nil {
		disks = []container.Disk{}
	}
	return hostMetricsEvent{
		ID:               host.ID,
		MetricsAvailable: host.MetricsAvailable,
		Load1:            host.Load1,
		Load5:            host.Load5,
		Load15:           host.Load15,
		Uptime:           host.Uptime,
		DiskTotal:        host.DiskTotal,
		DiskFree:         host.DiskFree,
		Disks:            disks,
	}
}

// hostHasMetrics is the send gate for the metrics ticker: nothing is broadcast
// for a host with no metrics to show. Disk counts on its own because it comes
// from the engine's data directory and does not need the host /proc mounted, so
// metricsAvailable is not the whole story.
func hostHasMetrics(host container.Host) bool {
	return host.MetricsAvailable || host.DiskTotal > 0 || len(host.Disks) > 0
}
