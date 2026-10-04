package web

import "github.com/amir20/dozzle/internal/container"

// updateMarkers decides which recorded updates one log stream sends: those
// that created a container the stream shows. It is used from the stream's own
// goroutine only.
type updateMarkers struct {
	records *container.UpdateRecords
	// shown is every container the stream has shown.
	shown map[markerKey]bool
}

type markerKey struct{ host, id string }

func newUpdateMarkers(records *container.UpdateRecords, containers []container.Container) *updateMarkers {
	m := &updateMarkers{records: records, shown: make(map[markerKey]bool, len(containers))}
	for _, c := range containers {
		m.shown[markerKey{c.Host, c.ID}] = true
	}
	return m
}

// opening is the updates that created the containers the stream opened with.
func (m *updateMarkers) opening() []container.UpdateRecord {
	var out []container.UpdateRecord
	for key := range m.shown {
		if u, ok := m.records.Latest(key.host, key.id); ok {
			out = append(out, u)
		}
	}
	return out
}

// started adds a container that started while the stream was open, and
// returns the update that created it if it is already recorded.
func (m *updateMarkers) started(c container.Container) (container.UpdateRecord, bool) {
	m.shown[markerKey{c.Host, c.ID}] = true
	return m.records.Latest(c.Host, c.ID)
}

// shows reports whether u created a container the stream shows.
func (m *updateMarkers) shows(u container.UpdateRecord) bool {
	return m.shown[markerKey{u.Host, u.NewID}]
}
