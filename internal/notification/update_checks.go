package notification

import (
	"context"
	"math/rand/v2"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/rs/zerolog/log"
)

// UpdateAvailableEventName is the event a newer image upstream raises. Like
// UpdateEventName it is not a Docker event: it comes from a daily check of
// every container that is not labelled dev.dozzle.update=off.
const UpdateAvailableEventName = "update_available"

// UpdateChecker checks every container Dozzle can see for a newer image,
// forced past the digest cache.
type UpdateChecker func(ctx context.Context) []container.ImageUpdate

// CheckForUpdates runs check once a day while the listener runs, and raises
// update_available for each new image it finds. Only the process that owns
// the hosts calls it, after ListenToUpdates: an agent's containers are
// checked by its server.
func (l *ContainerEventListener) CheckForUpdates(check UpdateChecker) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.checkUpdates = check
	// A random time of day, so every install does not reach the registries in
	// the same minute.
	l.checkAt = rand.N(24 * time.Hour)
}

// nextUpdateCheck is the first time after now that falls checkAt into a day.
func nextUpdateCheck(now time.Time, checkAt time.Duration) time.Time {
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	next := day.Add(checkAt)
	if !next.After(now) {
		next = day.AddDate(0, 0, 1).Add(checkAt)
	}
	return next
}

func (l *ContainerEventListener) runUpdateChecks(ctx context.Context, check UpdateChecker, checkAt time.Duration) {
	for {
		next := nextUpdateCheck(time.Now(), checkAt)
		log.Debug().Time("at", next).Msg("update alerts: next image check")
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Until(next)):
		}
		l.raiseAvailableUpdates(ctx, check(ctx))
	}
}

// raiseAvailableUpdates raises update_available for each update not raised
// before. alerted outlives Stop, so turning alerts off and on again does not
// repeat one; a restart does. An event dropped on a full channel is not marked,
// so the next check raises it again.
func (l *ContainerEventListener) raiseAvailableUpdates(ctx context.Context, updates []container.ImageUpdate) {
	var hosts map[string]container.Host
	l.alertedMu.Lock()
	defer l.alertedMu.Unlock()
	for _, u := range updates {
		if !u.Result.UpdateAvailable() || u.Result.RemoteDigest == "" {
			continue
		}
		// Keyed on the name, which an update keeps, rather than the id it replaces.
		key := u.Container.Host + "/" + u.Container.Name
		if l.alertedUpdates[key] == u.Result.RemoteDigest {
			continue
		}
		if hosts == nil {
			hosts = l.updateHosts(ctx)
		}
		host, ok := hosts[u.Container.Host]
		if !ok {
			continue
		}

		entry := &ContainerEventEntry{Event: updateAvailableEvent(u), Container: u.Container, Host: host}
		select {
		case l.channel <- entry:
			l.alertedUpdates[key] = u.Result.RemoteDigest
		case <-ctx.Done():
			return
		default:
			log.Warn().Str("containerID", u.Container.ID).Msg("Event channel full, dropping update_available event")
		}
	}
}

// updateHosts maps the id of every host among updateClients to that host, once
// per run rather than once per update.
func (l *ContainerEventListener) updateHosts(ctx context.Context) map[string]container.Host {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	clients := l.updateClients()
	found := make(chan container.Host, len(clients))
	for _, client := range clients {
		go func() {
			if host, err := client.Host(ctx); err == nil {
				found <- host
			} else {
				found <- container.Host{}
			}
		}()
	}
	hosts := make(map[string]container.Host, len(clients))
	for range clients {
		if host := <-found; host.ID != "" {
			hosts[host.ID] = host
		}
	}
	return hosts
}

func updateAvailableEvent(u container.ImageUpdate) container.ContainerEvent {
	return container.ContainerEvent{
		Name:    UpdateAvailableEventName,
		Host:    u.Container.Host,
		ActorID: u.Container.ID,
		ActorAttributes: map[string]string{
			"image":          u.Container.Image,
			"currentVersion": u.Result.CurrentVersion,
			"latestVersion":  u.Result.LatestVersion,
			"currentDigest":  u.Result.LocalDigest,
			"latestDigest":   u.Result.RemoteDigest,
		},
		Time: u.Result.CheckedAt,
	}
}
