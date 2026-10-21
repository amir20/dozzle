package web

import (
	"slices"
	"time"

	"github.com/amir20/dozzle/internal/container"
)

// The megaphone lists the updates Dozzle made lately. It reads them from
// memory, so a restart empties it; the window and the cap keep an instance
// that runs for months from listing old news.
const (
	updateFeedWindow = 7 * 24 * time.Hour
	updateFeedLimit  = 20
)

// autoPruneFeed is run if the megaphone shows it, nil otherwise: it shows one
// from the same window as the updates that did something, freed space or failed
// somewhere.
func autoPruneFeed(run *autoPruneRun, now time.Time) *autoPruneRun {
	if run == nil || now.Sub(run.At) > updateFeedWindow || (run.SpaceReclaimed == 0 && run.Failed == 0) {
		return nil
	}
	return run
}

// recentUpdates is the newest updateFeedLimit records from the last
// updateFeedWindow that visible lets through, oldest first.
func recentUpdates(records []container.UpdateRecord, now time.Time, visible func(container.UpdateRecord) bool) []container.UpdateRecord {
	var out []container.UpdateRecord
	for _, r := range slices.Backward(records) {
		if len(out) == updateFeedLimit || now.Sub(r.At) > updateFeedWindow {
			break
		}
		if visible(r) {
			out = append(out, r)
		}
	}
	slices.Reverse(out)
	return out
}
