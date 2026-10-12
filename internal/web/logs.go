package web

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"time"

	"github.com/amir20/dozzle/internal/analytics"
	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/utils"
	"github.com/amir20/dozzle/internal/web/search"
	"github.com/amir20/dozzle/internal/web/sse"
	"github.com/dustin/go-humanize"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// streamContainerLogs streams any set of containers, on any hosts, each named as
// `container=<host>/<id>` the way /api/containers/logs names them. One container
// is the container page, several are a merged view.
func (h *handler) streamContainerLogs(w http.ResponseWriter, r *http.Request) {
	refs, err := parseContainerRefs(r.URL.Query()["container"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(refs) == 1 {
		analytics.Count("view.container")
	} else {
		analytics.Count("view.merged")
	}

	wanted := make(map[containerRef]struct{}, len(refs))
	unique := make([]containerRef, 0, len(refs))
	for _, ref := range refs {
		if _, seen := wanted[ref]; !seen {
			wanted[ref] = struct{}{}
			unique = append(unique, ref)
		}
	}

	h.streamLogsForContainers(w, r, func(c *container.Container) bool {
		_, ok := wanted[containerRef{c.Host, c.ID}]
		return ok
	}, streamScope{containers: unique})
}

func (h *handler) streamLogsWithLabels(w http.ResponseWriter, r *http.Request) {
	// Parse label filters from URL path
	// Expected format: /labels/key1:value1,key2:value2/logs/stream
	labelsParam := chi.URLParam(r, "labels")
	labelFilters := make(map[string]string)

	if labelsParam != "" {
		for pair := range strings.SplitSeq(labelsParam, ",") {
			parts := strings.SplitN(pair, ":", 2)
			if len(parts) == 2 {
				labelFilters[parts[0]] = parts[1]
			}
		}
	}

	analytics.Count(labelView(labelFilters))

	// all=1 comes from the Kubernetes tab with "Show all containers" on, where a
	// finished Job pod is still listed and should open to its logs. A created
	// container has none to read yet.
	all := r.URL.Query().Get("all") == "1"

	h.streamLogsForContainers(w, r, func(container *container.Container) bool {
		if container.State != "running" && (!all || container.State == "created") {
			return false
		}

		// Check if all label filters match
		for key, value := range labelFilters {
			if container.Labels[key] != value {
				return false
			}
		}

		return len(labelFilters) > 0
	}, streamScope{})
}

// labelView names the view a label stream serves, by the label the page filters
// on: a swarm stack, a swarm service, or a Kubernetes namespace or workload.
func labelView(filters map[string]string) string {
	switch {
	case filters["com.docker.stack.namespace"] != "":
		return "view.stack"
	case filters["com.docker.swarm.service.name"] != "":
		return "view.service"
	default:
		return "view.namespace"
	}
}

// pathParam reads a URL param that may hold a "/". chi routes on RawPath, and so
// returns the param still escaped, only when Go set RawPath (in practice for %2F);
// otherwise the param is already decoded and unescaping again would mangle a "%".
func pathParam(r *http.Request, key string) (string, error) {
	value := chi.URLParam(r, key)
	if r.URL.RawPath == "" {
		return value, nil
	}
	return url.PathUnescape(value)
}

func (h *handler) streamGroupedLogs(w http.ResponseWriter, r *http.Request) {
	analytics.Count("view.group")
	group, err := pathParam(r, "group")
	if err != nil || group == "" {
		http.Error(w, "invalid group", http.StatusBadRequest)
		return
	}

	h.streamLogsForContainers(w, r, func(container *container.Container) bool {
		return container.State == "running" && container.Group == group
	}, streamScope{})
}

func (h *handler) streamHostGroupLogs(w http.ResponseWriter, r *http.Request) {
	analytics.Count("view.host")
	group, err := pathParam(r, "group")
	if err != nil || group == "" {
		http.Error(w, "invalid group", http.StatusBadRequest)
		return
	}

	hostIDs := make(map[string]struct{})
	for _, host := range h.hostService.Hosts() {
		if host.Group == group {
			hostIDs[host.ID] = struct{}{}
		}
	}

	h.streamLogsForContainers(w, r, func(c *container.Container) bool {
		_, ok := hostIDs[c.Host]
		return c.State == "running" && ok
	}, streamScope{})
}

func (h *handler) streamHostLogs(w http.ResponseWriter, r *http.Request) {
	analytics.Count("view.host")
	host := hostKey(r)
	h.streamLogsForContainers(w, r, func(container *container.Container) bool {
		return container.State == "running" && container.Host == host
	}, streamScope{host: host})
}

// streamScope says where a stream looks for the containers it starts with. The
// zero value lists every host, for streams that legitimately span hosts.
type streamScope struct {
	// host names the single host every container the stream can match lives on.
	// Listing just that host skips the fleet-wide fan-out, which re-dials every
	// unreachable agent at up to --timeout each before the first log line is read.
	host string
	// containers names exactly the containers the stream shows, so nothing is
	// listed at all: each is found in its own host's store when its tail starts.
	containers []containerRef
}

// listStreamContainers returns the containers a new stream starts with. Each is
// still resolved, and ACL checked, before anything about it is sent.
func (h *handler) listStreamContainers(scope streamScope, labels container.ContainerLabels, containerFilter container.ContainerFilter) []containerRef {
	if scope.containers != nil {
		return scope.containers
	}

	if scope.host == "" {
		containers, errs := h.hostService.ListAllContainersFiltered(labels, containerFilter)
		if len(errs) > 0 {
			log.Warn().Err(errs[0]).Msg("error while listing containers")
		}
		refs := make([]containerRef, len(containers))
		for i, c := range containers {
			refs[i] = containerRef{c.Host, c.ID}
		}
		return refs
	}

	hostContainers, err := h.hostService.ListContainersForHost(scope.host, labels)
	if err != nil {
		log.Warn().Err(err).Str("host", scope.host).Msg("error while listing containers")
	}
	var refs []containerRef
	for _, c := range hostContainers {
		if containerFilter(&c) {
			refs = append(refs, containerRef{c.Host, c.ID})
		}
	}
	return refs
}

func (h *handler) streamLogsForContainers(w http.ResponseWriter, r *http.Request, containerFilter container.ContainerFilter, scope streamScope) {
	stdTypes := parseStdTypes(r)
	if stdTypes == 0 {
		http.Error(w, "stdout or stderr is required", http.StatusBadRequest)
		return
	}

	filter, err := parseLogFilter(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// `since` is the start the person picked ("live, from 15 minutes ago"). The
	// tail never replays older lines, and a search's backfill stops there too.
	var floor time.Time
	if r.URL.Query().Has("since") {
		floor, err = time.Parse(time.RFC3339Nano, r.URL.Query().Get("since"))
		if err != nil {
			http.Error(w, "since must be an RFC 3339 time", http.StatusBadRequest)
			return
		}
	}

	sseWriter, err := sse.NewWriter(r.Context(), w, r)
	if err != nil {
		log.Error().Err(err).Msg("error creating sse writer")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer sseWriter.Close()

	ctx := r.Context()
	userLabels := h.resolveLabels(r)
	refs := h.listStreamContainers(scope, userLabels, containerFilter)

	liveLogs := make(chan *container.LogEvent)
	events := make(chan *container.ContainerEvent, 1)
	backfill := make(chan []*container.LogEvent)
	searchStatusCh := make(chan searchStatus)

	// With a narrowing filter the live tail starts now, and everything older
	// arrives through the backfill walk instead of the tail's own history.
	since := floor
	if filter.narrowing() {
		since = time.Now()
	}

	// Each container is resolved once and shared by its tail and the backfill
	// walk: for an agent host FindContainer is a gRPC round-trip. A tail starts as
	// soon as its own container resolves, so one slow host doesn't hold up the rest.
	opened := make(chan []*container.ContainerService, 1)
	go func() {
		services, _ := h.resolveContainers(refs, userLabels, func(s *container.ContainerService) {
			go tailContainerLogs(ctx, s, since, stdTypes, liveLogs, events)
		})
		opened <- services
		if filter.narrowing() {
			searchBackfill(ctx, services, since, floor, stdTypes, filter, backfill, searchStatusCh)
		}
	}()

	newContainers := make(chan container.Container)
	h.hostService.SubscribeContainersStarted(ctx, newContainers, containerFilter)

	// Update markers: the update that created a container this stream shows,
	// so the viewer can put "1.4.1 → 1.4.2" above its first lines. An update
	// is recorded once its swap is done, after the new container started, so
	// one that finishes while the stream is open arrives on updates. Only a
	// container that resolved is marked, since the lookup is the ACL check.
	markers := newUpdateMarkers(container.Updates)
	updates := make(chan container.UpdateRecord, 16)
	markers.records.Subscribe(ctx, updates)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	sseWriter.Retry(reconnectDelay)
	sseWriter.Ping()
loop:
	for {
		select {
		case services := <-opened:
			for _, u := range markers.opened(services) {
				if err := sseWriter.Event("container-update", u); err != nil {
					log.Error().Err(err).Msg("error encoding container update")
				}
			}

		case logEvent := <-liveLogs:
			if !filter.matches(logEvent) {
				continue
			}

			search.EscapeHTMLValues(logEvent)
			sseWriter.Message(logEvent)
		case c := <-newContainers:
			// The lookup doubles as the ACL check, so hand the resolved service
			// straight to the streamer instead of resolving the same container twice.
			if containerService, err := h.hostService.FindContainer(c.Host, c.ID, userLabels); err == nil {
				// Written straight to the client instead of pushed through `events`.
				// This case runs on the same goroutine that drains `events`, so a send
				// here waits on a reader that is this very statement: with the buffer
				// already holding a container-stopped from a tail — which is what a
				// redeploy produces — the handler deadlocks for good. It then stops
				// draining newContainers, which backs up into the shared container store
				// and freezes it for every client on that host.
				event := &container.ContainerEvent{ActorID: c.ID, Name: "container-started", Host: c.Host, Time: time.Now()}
				if err := sseWriter.Event("container-event", event); err != nil {
					log.Error().Err(err).Msg("error encoding container event")
				}
				if u, ok := markers.started(c); ok {
					if err := sseWriter.Event("container-update", u); err != nil {
						log.Error().Err(err).Msg("error encoding container update")
					}
				}
				go tailContainerLogs(ctx, containerService, since, stdTypes, liveLogs, events)
			}

		case u := <-updates:
			if markers.shows(u) {
				if err := sseWriter.Event("container-update", u); err != nil {
					log.Error().Err(err).Msg("error encoding container update")
				}
			}

		case event := <-events:
			log.Debug().Str("event", event.Name).Str("container", event.ActorID).Msg("received event")
			if err := sseWriter.Event("container-event", event); err != nil {
				log.Error().Err(err).Msg("error encoding container event")
			}

		case backfillEvents := <-backfill:
			for _, event := range backfillEvents {
				search.EscapeHTMLValues(event)
			}
			if err := sseWriter.Event("logs-backfill", backfillEvents); err != nil {
				log.Error().Err(err).Msg("error encoding container event")
			}

		case s := <-searchStatusCh:
			if err := sseWriter.Event("search-status", s); err != nil {
				log.Error().Err(err).Msg("error encoding search status")
			}

		case <-ticker.C:
			sseWriter.Ping()

		case <-ctx.Done():
			break loop
		}
	}

	logMemStats()
}

// tailContainerLogs streams one container's logs from `since` (or its start, if
// later) into logs, and reports a container-stopped on events when it ends.
func tailContainerLogs(ctx context.Context, containerService *container.ContainerService, since time.Time, stdTypes container.StdType, logs chan<- *container.LogEvent, events chan<- *container.ContainerEvent) {
	c := containerService.Container
	start := utils.Max(since, c.StartedAt)
	err := containerService.StreamLogs(ctx, start, stdTypes, logs)
	if err == nil {
		return
	}

	if errors.Is(err, io.EOF) {
		log.Debug().Str("container", c.ID).Msg("streaming ended")
		// c is the container as it was when the tail began, so a container that
		// restarts still carries its previous run's FinishedAt. Only trust it when
		// it belongs to this run.
		finishedAt := c.FinishedAt
		if !finishedAt.After(c.StartedAt) {
			finishedAt = time.Now()
		}
		select {
		case events <- &container.ContainerEvent{
			ActorID: c.ID,
			Name:    "container-stopped",
			Host:    c.Host,
			Time:    finishedAt,
		}:
		case <-ctx.Done():
		}
	} else if errors.Is(err, context.Canceled) || ctx.Err() != nil {
		// the client went away; a read already in flight comes back as
		// "use of closed network connection" instead of a cancellation
		log.Debug().Err(err).Str("container", c.ID).Msg("streaming stopped after client disconnected")
	} else {
		log.Error().Err(err).Str("container", c.ID).Msg("unknown error while streaming logs")
	}
}

func logMemStats() {
	if e := log.Debug(); e.Enabled() {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		e.Str("allocated", humanize.Bytes(m.Alloc)).
			Str("totalAllocated", humanize.Bytes(m.TotalAlloc)).
			Str("system", humanize.Bytes(m.Sys)).
			Int("routines", runtime.NumGoroutine()).
			Msg("runtime mem stats")
	}
}
