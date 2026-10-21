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
