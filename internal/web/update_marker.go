package web

import (
	"slices"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/web/sse"
	"github.com/rs/zerolog/log"
)

// The update marker: a log stream opens on a container that an update created
// with the update that created it, so the viewer can put "1.4.1 → 1.4.2" at the
// top of the new container's lines. It is read from the host's update history,
// keyed on the container the update produced, so a redirect to the new id after
// an update finds it without carrying anything over.

// recentUpdates is the kept update events of every host that keeps them. An
// agent keeps its own on the agent, so its containers get no marker yet.
func (h *handler) recentUpdates() []container.ContainerUpdateEvent {
	var all []container.ContainerUpdateEvent
	for _, s := range h.hostService.ClientServices(false) {
		if history, ok := s.(container.UpdateHistory); ok {
			all = append(all, history.RecentUpdates()...)
		}
	}
	return all
}

// updateFor is the newest update that left c running under its name: the one
// whose new container is c. A rolled back swap counts too, since the container
// it put back is the one now running.
func updateFor(events []container.ContainerUpdateEvent, c container.Container) (container.ContainerUpdateEvent, bool) {
	for _, e := range slices.Backward(events) {
		if e.NewID == c.ID && (e.Host == "" || e.Host == c.Host) {
			return e, true
		}
	}
	return container.ContainerUpdateEvent{}, false
}

// writeUpdateMarkers sends a container-update event for each container in cs
// that an update created. events is read once by the caller and shared.
func writeUpdateMarkers(w *sse.Writer, events []container.ContainerUpdateEvent, cs ...container.Container) {
	if len(events) == 0 {
		return
	}
	for _, c := range cs {
		if e, ok := updateFor(events, c); ok {
			if err := w.Event("container-update", e); err != nil {
				log.Error().Err(err).Msg("error encoding container update")
			}
		}
	}
}
