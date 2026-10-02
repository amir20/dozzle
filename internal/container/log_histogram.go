package container

import "time"

// LogHistogram counts a container's log lines in fixed-width buckets, with the
// error lines among them counted again. Bucket i covers
// [Start + i*Width, Start + (i+1)*Width).
type LogHistogram struct {
	Start  time.Time
	Width  time.Duration
	Total  []uint32
	Errors []uint32
	// ScannedFrom is set when the scan stopped before reaching Start to stay
	// inside its budget. Buckets that end before it are unknown, not empty.
	ScannedFrom time.Time
	// Only for a window with no lines at all: the newest line before it and the
	// oldest after it, so an empty view can say where the log does have lines.
	// Zero when there is none, or when the search ran out of budget.
	Before time.Time
	After  time.Time
}

// Len is the number of buckets.
func (h LogHistogram) Len() int { return len(h.Total) }
