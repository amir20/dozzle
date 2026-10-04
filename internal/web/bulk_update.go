package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/amir20/dozzle/internal/analytics"
	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/web/sse"
	"github.com/rs/zerolog/log"
)

// A bulk update recreates many containers in one go, from the dashboard or on
// the auto-update schedule. It runs detached from any request: closing the tab
// that started it must not leave a container stopped halfway through a swap.
//
// Hosts run in parallel and each host one container at a time, so one slow
// registry or broken container never holds up another host and no daemon is
// asked to pull ten images at once. Dozzle's own container always goes last,
// because updating it ends this process.

const (
	bulkQueued   = "queued"
	bulkUpToDate = "up-to-date"
	bulkDone     = "done"
	bulkError    = "error"
	// bulkRolledBack is final: the new container failed and the old one is back.
	bulkRolledBack = container.UpdateRolledBack

	// Generous, since a pull of a multi-gigabyte image on a slow link is
	// legitimate. It only exists so a wedged daemon cannot pin the job forever.
	bulkItemTimeout = 20 * time.Minute
	// Pull progress arrives many times a second per layer. Watchers get the
	// latest state at most this often.
	bulkNotifyInterval = 250 * time.Millisecond
)

var errBulkUpdateBusy = errors.New("an update is already running")

type bulkUpdateItem struct {
	Host   string `json:"host"`
	ID     string `json:"id"`
	Name   string `json:"name"`
	Image  string `json:"image"`
	Self   bool   `json:"self,omitempty"`
	Status string `json:"status"`
	// Current and Total sum every layer of the pull seen so far.
	Current int64  `json:"current,omitempty"`
	Total   int64  `json:"total,omitempty"`
	Error   string `json:"error,omitempty"`

	service *container.ContainerService
	layers  map[string][2]int64
}

type bulkUpdateJob struct {
	Trigger string `json:"trigger"` // "manual" or "schedule"
	// requestedBy is the user who started a manual job. Every container in it
	// was resolved against that user's labels.
	requestedBy string
	// flushUsage sends the day's usage before Dozzle replaces itself. May be nil.
	flushUsage func()
	StartedAt  time.Time         `json:"startedAt"`
	FinishedAt *time.Time        `json:"finishedAt,omitempty"`
	Items      []*bulkUpdateItem `json:"items"`
}

type bulkUpdater struct {
	mu       sync.Mutex
	job      *bulkUpdateJob
	running  bool
	done     chan struct{}
	watchers map[chan struct{}]struct{}
}

// selfSwarmService is the swarm service Dozzle's own container belongs to, or
// empty when it is not a swarm task or is not among containers.
func selfSwarmService(containers []container.Container) string {
	selfID := setupSelfID()
	for _, c := range containers {
		if selfID != "" && len(c.ID) >= 12 && strings.HasPrefix(selfID, c.ID) {
			return c.Labels[container.SwarmServiceIDLabel]
		}
	}
	return ""
}

// isSelfContainer reports whether updating c replaces this process: it is
// Dozzle's own container, or any replica of Dozzle's swarm service, since
// rolling one rolls them all.
func isSelfContainer(c container.Container, selfService string) bool {
	selfID := setupSelfID()
	if selfID != "" && len(c.ID) >= 12 && strings.HasPrefix(selfID, c.ID) {
		return true
	}
	return selfService != "" && c.Labels[container.SwarmServiceIDLabel] == selfService
}

// bulkUpdates is shared by the handler and the scheduler, which are built
// separately, so a scheduled run and a click cannot overlap.
var bulkUpdates = &bulkUpdater{watchers: make(map[chan struct{}]struct{})}

// Start queues services and runs them in the background. The returned channel
// closes when every one has finished.
// selfService is Dozzle's own swarm service, if any (see selfSwarmService).
func (u *bulkUpdater) Start(services []*container.ContainerService, trigger, selfService, requestedBy string, flushUsage func()) (<-chan struct{}, error) {
	u.mu.Lock()
	if u.running {
		u.mu.Unlock()
		return nil, errBulkUpdateBusy
	}

	seen := make(map[string]*bulkUpdateItem, len(services))
	job := &bulkUpdateJob{Trigger: trigger, StartedAt: time.Now(), requestedBy: requestedBy, flushUsage: flushUsage}
	for _, service := range services {
		c := service.Container
		self := isSelfContainer(c, selfService)
		// Every replica of a swarm service is the same update: the first one
		// rolls the whole service, and the rest would roll it again. If any of
		// them is Dozzle's, the one kept still has to go last.
		key := c.Host + "/" + c.ID
		if id := c.Labels[container.SwarmServiceIDLabel]; id != "" {
			key = "service/" + id
		}
		if kept, ok := seen[key]; ok {
			kept.Self = kept.Self || self
			continue
		}
		item := &bulkUpdateItem{
			Host:    c.Host,
			ID:      c.ID,
			Name:    c.Name,
			Image:   c.Image,
			Self:    self,
			Status:  bulkQueued,
			service: service,
		}
		seen[key] = item
		job.Items = append(job.Items, item)
	}

	done := make(chan struct{})
	u.job = job
	u.running = true
	u.done = done
	u.mu.Unlock()
	u.notify()

	go func() {
		defer close(done)
		u.run(job)
	}()
	return done, nil
}

// idle closes once no job is running.
func (u *bulkUpdater) idle() <-chan struct{} {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.running {
		return u.done
	}
	closed := make(chan struct{})
	close(closed)
	return closed
}

func (u *bulkUpdater) run(job *bulkUpdateJob) {
	var self *bulkUpdateItem
	byHost := make(map[string][]*bulkUpdateItem)
	for _, item := range job.Items {
		if item.Self {
			self = item
			continue
		}
		byHost[item.Host] = append(byHost[item.Host], item)
	}

	var wg sync.WaitGroup
	for _, items := range byHost {
		wg.Go(func() {
			for _, item := range items {
				u.runItem(item, updateSource(job.Trigger))
			}
		})
	}
	wg.Wait()

	// Everything else is finished, so marking the job done here lets watchers
	// see the full result before Dozzle goes away.
	if self != nil {
		u.runItem(self, updateSource(job.Trigger))
	}

	u.mu.Lock()
	now := time.Now()
	job.FinishedAt = &now
	u.running = false
	u.mu.Unlock()
	u.notify()
}

// updateSource is what the update record says started a job's updates.
func updateSource(trigger string) string {
	if trigger == "schedule" {
		return container.UpdateSourceSchedule
	}
	return container.UpdateSourceDozzle
}

func (u *bulkUpdater) runItem(item *bulkUpdateItem, source string) {
	// A stopped container is never updated. The host refuses one too, but an
	// older agent would not.
	if item.service.Container.State != "running" {
		u.mu.Lock()
		item.Status = bulkError
		item.Error = container.ErrNotRunning.Error()
		u.mu.Unlock()
		u.notify()
		log.Info().Str("container", item.Name).Msg("bulk update: container not updated, it is not running")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), bulkItemTimeout)
	defer cancel()

	progressCh := make(chan container.UpdateProgress, 50)
	errCh := make(chan error, 1)
	go func() {
		_, err := item.service.Update(ctx, source, progressCh)
		errCh <- err
	}()

	for progress := range progressCh {
		u.apply(item, progress)
	}

	err := <-errCh
	u.mu.Lock()
	// A rolled back update already says what went wrong, and that the old
	// container is back.
	if err != nil && item.Status != bulkError && item.Status != bulkRolledBack {
		item.Status = bulkError
		item.Error = err.Error()
	}
	u.mu.Unlock()
	u.notify()

	if err != nil {
		log.Error().Err(err).Str("container", item.Name).Msg("bulk update: container update failed")
	} else {
		log.Info().Str("container", item.Name).Str("status", item.Status).Msg("bulk update: container finished")
	}
}

func (u *bulkUpdater) apply(item *bulkUpdateItem, p container.UpdateProgress) {
	u.mu.Lock()
	item.Status = p.Status
	if p.Status == "pulling" && p.Layer != "" && p.Total > 0 {
		if item.layers == nil {
			item.layers = make(map[string][2]int64)
		}
		item.layers[p.Layer] = [2]int64{p.Current, p.Total}
		item.Current, item.Total = 0, 0
		for _, layer := range item.layers {
			item.Current += layer[0]
			item.Total += layer[1]
		}
	}
	if p.Status == bulkError || p.Status == bulkRolledBack {
		item.Error = p.Error
	}
	self := item.Self
	flushUsage := u.job.flushUsage
	u.mu.Unlock()
	u.notify()

	// Same as runSelfUpdate: the helper is about to replace this process, so
	// this is the last chance to send the day's counters.
	if self && p.Status == "recreating" {
		analytics.Count("image.update")
		if flushUsage != nil {
			flushUsage()
		}
	}
}

// notify wakes every watcher without blocking. A watcher that has not caught
// up yet already has a wake pending, and it reads the latest state anyway.
func (u *bulkUpdater) notify() {
	u.mu.Lock()
	defer u.mu.Unlock()
	for ch := range u.watchers {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

func (u *bulkUpdater) watch() (<-chan struct{}, func()) {
	ch := make(chan struct{}, 1)
	u.mu.Lock()
	u.watchers[ch] = struct{}{}
	u.mu.Unlock()
	return ch, func() {
		u.mu.Lock()
		delete(u.watchers, ch)
		u.mu.Unlock()
	}
}

// snapshot copies the job so it can be encoded outside the lock. visible,
// when set, drops containers the caller is not allowed to see.
func (u *bulkUpdater) snapshot(visible func(job *bulkUpdateJob, item *bulkUpdateItem) bool) (bulkUpdateJob, bool) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.job == nil {
		return bulkUpdateJob{}, u.running
	}
	job := *u.job
	job.Items = make([]*bulkUpdateItem, 0, len(u.job.Items))
	for _, item := range u.job.Items {
		if visible != nil && !visible(u.job, item) {
			continue
		}
		copied := *item
		copied.layers = nil
		job.Items = append(job.Items, &copied)
	}
	return job, u.running
}

type bulkUpdateRequest struct {
	Containers []struct {
		Host string `json:"host"`
		ID   string `json:"id"`
	} `json:"containers"`
}

// startBulkUpdate resolves each container the way a single update would, so a
// user can never reach a container through here that their labels hide.
func (h *handler) startBulkUpdate(w http.ResponseWriter, r *http.Request) {
	if !h.permitActions(r) {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}

	if !isJSONRequest(r) {
		http.Error(w, "expected application/json", http.StatusUnsupportedMediaType)
		return
	}

	var req bulkUpdateRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(req.Containers) == 0 {
		http.Error(w, "no containers", http.StatusBadRequest)
		return
	}

	labels := h.resolveLabels(r)
	services := make([]*container.ContainerService, 0, len(req.Containers))
	for _, c := range req.Containers {
		service, err := h.hostService.FindContainer(c.Host, c.ID, labels)
		if err != nil {
			// Gone since the list was drawn, most likely updated some other way.
			log.Debug().Err(err).Str("host", c.Host).Str("container", c.ID).Msg("bulk update: skipping container")
			continue
		}
		services = append(services, service)
	}
	if len(services) == 0 {
		http.Error(w, "no containers found", http.StatusNotFound)
		return
	}

	all, _ := h.hostService.ListAllContainers(h.config.Labels)
	requestedBy := ""
	if h.config.Authorization.Provider != NONE {
		requestedBy = auth.UserFromContext(r.Context()).Username
	}
	if _, err := bulkUpdates.Start(services, "manual", selfSwarmService(all), requestedBy, h.flushUsage); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	// Counted per container, so the number means the same as a single update.
	analytics.Default.Add("action.update", len(services))
	log.Info().Int("count", len(services)).Msg("bulk update started")
	// The caller tells its own job apart from the one before by startedAt.
	job, _ := bulkUpdates.snapshot(nil)
	writeJSON(w, http.StatusAccepted, map[string]time.Time{"startedAt": job.StartedAt})
}

// streamBulkUpdate sends the current job, then every change to it. A tab that
// opens mid-update picks up where things are, not where it would have started.
func (h *handler) streamBulkUpdate(w http.ResponseWriter, r *http.Request) {
	// A restricted user sees all of a job they started, since every container
	// in it was checked against their labels. In anyone else's job they only
	// see containers still visible to them by id: a name can be reused by a
	// container they may see after the one they may not was removed.
	var visible func(job *bulkUpdateJob, item *bulkUpdateItem) bool
	if h.restrictedUser(r) {
		username := auth.UserFromContext(r.Context()).Username
		containers, _ := h.hostService.ListAllContainers(h.resolveLabels(r))
		allowed := make(map[string]struct{}, len(containers))
		for _, c := range containers {
			allowed[c.Host+"/"+c.ID] = struct{}{}
		}
		visible = func(job *bulkUpdateJob, item *bulkUpdateItem) bool {
			if username != "" && job.requestedBy == username {
				return true
			}
			_, ok := allowed[item.Host+"/"+item.ID]
			return ok
		}
	}

	sseWriter, err := sse.NewWriter(r.Context(), w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer sseWriter.Close()

	wake, stop := bulkUpdates.watch()
	defer stop()

	send := func() error {
		job, running := bulkUpdates.snapshot(visible)
		return sseWriter.Event("bulk-update", struct {
			bulkUpdateJob
			Running bool `json:"running"`
		}{job, running})
	}
	if err := send(); err != nil {
		return
	}

	ticker := time.NewTicker(keepAliveInterval)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			if err := sseWriter.Ping(); err != nil {
				return
			}
		case <-wake:
			if err := send(); err != nil {
				return
			}
			// Coalesces a burst of pull progress into one event.
			select {
			case <-r.Context().Done():
				return
			case <-time.After(bulkNotifyInterval):
			}
		}
	}
}
