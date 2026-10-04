package web

import (
	"context"
	"slices"
	"sync"

	"github.com/amir20/dozzle/internal/container"
)

// The update marker: a log stream opens on a container that an update created
// with the update that created it, so the viewer can put "1.4.1 → 1.4.2" above
// the new container's lines. It is read from the update history of the
// container's own host, a Docker host's or an agent's, keyed on the container
// the update produced, so a redirect to the new id after an update finds it
// without carrying anything over.

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

// sendUpdateMarker sends the update that created the container, if one did, to
// out. recent reads the host's update events. The marker is dated, so it may
// arrive after the container's first lines; the viewer places it.
func sendUpdateMarker(ctx context.Context, s *container.ContainerService, recent func() []container.ContainerUpdateEvent, out chan<- container.ContainerUpdateEvent) {
	e, ok := updateFor(recent(), s.Container)
	if !ok {
		return
	}
	select {
	case out <- e:
	case <-ctx.Done():
	}
}

// hostUpdates reads each host's update events once for the containers a stream
// opens with. A stack on one agent would otherwise ask that agent once per
// container.
type hostUpdates struct {
	mu     sync.Mutex
	byHost map[string]func() []container.ContainerUpdateEvent
}

func newHostUpdates() *hostUpdates {
	return &hostUpdates{byHost: make(map[string]func() []container.ContainerUpdateEvent)}
}

// get is the shared read of s's host.
func (u *hostUpdates) get(s *container.ContainerService) func() []container.ContainerUpdateEvent {
	u.mu.Lock()
	defer u.mu.Unlock()
	read, ok := u.byHost[s.Container.Host]
	if !ok {
		read = sync.OnceValue(s.RecentUpdates)
		u.byHost[s.Container.Host] = read
	}
	return read
}
