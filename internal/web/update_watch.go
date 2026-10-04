package web

import (
	"sync"
	"time"

	"github.com/amir20/dozzle/internal/container"
)

// Someone who starts an update from the Dozzle UI can ask Dozzle Cloud to watch
// it ("Have Dozzle Cloud watch this update"). That choice is made on the
// request, and the update event it applies to is produced later by the store,
// so it is kept here by the container being replaced until the event arrives.
//
// It lives in memory only. A Dozzle that restarts mid-update has lost the
// event anyway, and the image snapshot it sends on its next connect covers it.

// updateWatchTTL is how long a choice waits for its update. Longer than the
// slowest bulk run is likely to reach a container, since every container on a
// host waits for the ones queued before it.
const updateWatchTTL = 24 * time.Hour

type updateWatchList struct {
	mu  sync.Mutex
	at  map[string]time.Time
	now func() time.Time
}

var updateWatches = &updateWatchList{at: make(map[string]time.Time), now: time.Now}

// watchKey identifies the container an update replaces. Events carry the
// 12-character id the store uses, and a request may carry the full one.
func watchKey(host, id string) string {
	if len(id) > 12 {
		id = id[:12]
	}
	return host + "/" + id
}

// set records whether the update about to replace host/id is watched. Every
// update records its choice, so an unticked retry of a container whose earlier
// update was watched and rolled back is not watched.
func (w *updateWatchList) set(host, id string, watched bool) {
	key := watchKey(host, id)
	w.mu.Lock()
	defer w.mu.Unlock()
	now := w.now()
	for k, at := range w.at {
		if now.Sub(at) > updateWatchTTL {
			delete(w.at, k)
		}
	}
	if watched {
		w.at[key] = now
	} else {
		delete(w.at, key)
	}
}

// setAll sets the choice for every service, and returns a func that puts back
// what each one had before.
func (w *updateWatchList) setAll(services []*container.ContainerService, watched bool) (restore func()) {
	w.mu.Lock()
	before := make(map[string]time.Time, len(services))
	for _, s := range services {
		key := watchKey(s.Container.Host, s.Container.ID)
		if at, ok := w.at[key]; ok {
			before[key] = at
		}
	}
	w.mu.Unlock()

	for _, s := range services {
		w.set(s.Container.Host, s.Container.ID, watched)
	}
	return func() {
		w.mu.Lock()
		defer w.mu.Unlock()
		for _, s := range services {
			key := watchKey(s.Container.Host, s.Container.ID)
			if at, ok := before[key]; ok {
				w.at[key] = at
			} else {
				delete(w.at, key)
			}
		}
	}
}

// settle drops the choice for an update that ended without a swap: already up
// to date, failed, or abandoned. No event comes for it, and a choice left
// behind would wait under a container id that is still running.
func (w *updateWatchList) settle(host, id, status string) {
	if status == container.UpdateDone || status == container.UpdateRolledBack {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.at, watchKey(host, id))
}

// watched reports whether the update event was started with the checkbox
// ticked. A rolled back swap reports the container it tried to replace as
// OldID too, so both outcomes of a watched update are watched. Only an update
// Dozzle made can match: the checkbox says nothing about one Watchtower or
// compose makes to the same container later.
func (w *updateWatchList) watched(event container.ContainerUpdateEvent) bool {
	if event.Source != container.UpdateSourceDozzle {
		return false
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	at, ok := w.at[watchKey(event.Host, event.OldID)]
	return ok && w.now().Sub(at) <= updateWatchTTL
}

// UpdateWatched reports whether the user asked Dozzle Cloud to watch the update
// event when they started it. The cloud client reads it to decide whether it
// may push the update.
func UpdateWatched(event container.ContainerUpdateEvent) bool {
	return updateWatches.watched(event)
}
