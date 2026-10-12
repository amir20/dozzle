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

func newUpdateMarkers(records *container.UpdateRecords) *updateMarkers {
	return &updateMarkers{records: records, shown: make(map[markerKey]bool)}
}

// opened adds the containers the stream opened with, once they have resolved,
// and returns the updates that created them.
func (m *updateMarkers) opened(services []*container.ContainerService) []container.UpdateRecord {
	var out []container.UpdateRecord
	for _, s := range services {
		m.shown[markerKey{s.Container.Host, s.Container.ID}] = true
		if u, ok := m.records.Latest(s.Container.Host, s.Container.ID); ok {
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
