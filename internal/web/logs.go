package web

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/utils"
	"github.com/amir20/dozzle/internal/web/search"
	"github.com/amir20/dozzle/internal/web/sse"
	"github.com/dustin/go-humanize"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

func (h *handler) streamContainerLogs(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	h.streamLogsForContainers(w, r, func(container *container.Container) bool {
		return container.ID == id && container.Host == hostKey(r)
	}, hostKey(r))
}

func (h *handler) streamLogsMerged(w http.ResponseWriter, r *http.Request) {
	ids := make(map[string]bool)
	for id := range strings.SplitSeq(chi.URLParam(r, "ids"), ",") {
		ids[id] = true
	}

	h.streamLogsForContainers(w, r, func(container *container.Container) bool {
		return ids[container.ID] && container.Host == hostKey(r)
	}, hostKey(r))
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
	}, "")
}

func (h *handler) streamGroupedLogs(w http.ResponseWriter, r *http.Request) {
	group := chi.URLParam(r, "group")

	h.streamLogsForContainers(w, r, func(container *container.Container) bool {
		return container.State == "running" && container.Group == group
	}, "")
}

func (h *handler) streamHostGroupLogs(w http.ResponseWriter, r *http.Request) {
	group, err := url.PathUnescape(chi.URLParam(r, "group"))
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
	}, "")
}

func (h *handler) streamHostLogs(w http.ResponseWriter, r *http.Request) {
	host := hostKey(r)
	h.streamLogsForContainers(w, r, func(container *container.Container) bool {
		return container.State == "running" && container.Host == host
	}, host)
}

// listStreamContainers returns the running containers a new stream starts with.
// hostScope, when non-empty, names the single host every container this stream
// can match lives on. Listing just that host skips the fleet-wide fan-out, which
// re-dials every unreachable agent at up to --timeout each before the first log
// line can be read. Streams that legitimately span hosts pass "".
func (h *handler) listStreamContainers(hostScope string, labels container.ContainerLabels, containerFilter container.ContainerFilter) []container.Container {
	if hostScope == "" {
		containers, errs := h.hostService.ListAllContainersFiltered(labels, containerFilter)
		if len(errs) > 0 {
			log.Warn().Err(errs[0]).Msg("error while listing containers")
		}
		return containers
	}

	hostContainers, err := h.hostService.ListContainersForHost(hostScope, labels)
	if err != nil {
		log.Warn().Err(err).Str("host", hostScope).Msg("error while listing containers")
	}
	var containers []container.Container
	for _, c := range hostContainers {
		if containerFilter(&c) {
			containers = append(containers, c)
		}
	}
	return containers
}

func (h *handler) streamLogsForContainers(w http.ResponseWriter, r *http.Request, containerFilter container.ContainerFilter, hostScope string) {
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

	sseWriter, err := sse.NewWriter(r.Context(), w, r)
	if err != nil {
		log.Error().Err(err).Msg("error creating sse writer")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer sseWriter.Close()

	ctx := r.Context()
	userLabels := h.resolveLabels(r)
	existingContainers := h.listStreamContainers(hostScope, userLabels, containerFilter)

	liveLogs := make(chan *container.LogEvent)
	events := make(chan *container.ContainerEvent, 1)
	backfill := make(chan []*container.LogEvent)
	searchStatusCh := make(chan searchStatus)

	// With a narrowing filter the live tail starts now, and everything older
	// arrives through the backfill walk instead of the tail's own history.
	var since time.Time
	if filter.narrowing() {
		since = time.Now()
	}

	// Each container is resolved once and shared by its tail and the backfill
	// walk: for an agent host FindContainer is a gRPC round-trip. Lookups run in
	// parallel so one slow host doesn't hold up every other container's tail.
	services := make([]*container.ContainerService, len(existingContainers))
	var resolved sync.WaitGroup
	resolved.Add(len(existingContainers))
	for i, c := range existingContainers {
		go func() {
			containerService, err := h.hostService.FindContainer(c.Host, c.ID, userLabels)
			if err == nil {
				services[i] = containerService
			}
			resolved.Done()
			if err != nil {
				log.Error().Err(err).Msg("error while finding container")
				return
			}
			tailContainerLogs(ctx, containerService, since, stdTypes, liveLogs, events)
		}()
	}

	if filter.narrowing() {
		go func() {
			resolved.Wait()
			found := slices.DeleteFunc(services, func(s *container.ContainerService) bool { return s == nil })
			searchBackfill(ctx, found, since, stdTypes, filter, backfill, searchStatusCh)
		}()
	}

	newContainers := make(chan container.Container)
	h.hostService.SubscribeContainersStarted(ctx, newContainers, containerFilter)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	sseWriter.Retry(reconnectDelay)
	sseWriter.Ping()
loop:
	for {
		select {
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
				go tailContainerLogs(ctx, containerService, since, stdTypes, liveLogs, events)
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
		finishedAt := c.FinishedAt
		if c.FinishedAt.IsZero() {
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
