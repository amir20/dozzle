package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/notification/dispatcher"
	"github.com/amir20/dozzle/types"
	"github.com/rs/zerolog/log"
)

// processLogEvents processes log events from the listener channel
func (m *Manager) processLogEvents() {
	for {
		select {
		case <-m.ctx.Done():
			return
		case logEvent := <-m.listener.LogChannel():
			if logEvent == nil {
				return
			}
			m.processLogEvent(logEvent)
		}
	}
}

// processLogEvent processes a single log event and sends notifications for matching subscriptions
func (m *Manager) processLogEvent(logEvent *container.LogEvent) {
	// Get container and host from log event's ContainerID
	ctx, cancel := context.WithTimeout(m.ctx, 5*time.Second)
	defer cancel()

	c, host, err := m.listener.FindContainerWithHost(ctx, logEvent.ContainerID, nil)
	if err != nil {
		log.Error().Err(err).Str("containerID", logEvent.ContainerID).Msg("Failed to find container")
		return
	}

	// Skip logs from Dozzle's own containers to avoid feedback loops
	if isDozzleContainer(c) {
		return
	}

	notificationContainer := FromContainerModel(c, host)
	notificationLog := FromLogEvent(*logEvent)

	m.subscriptions.Range(func(_ int, sub *Subscription) bool {
		// Skip disabled or non-log subscriptions
		if !sub.Enabled || !sub.IsLogAlert() {
			return true
		}

		// Check container filter
		if !sub.MatchesContainer(notificationContainer) {
			return true
		}

		// Check log filter
		if !sub.MatchesLog(notificationLog) {
			return true
		}

		// Update stats
		sub.AddTriggeredContainer(notificationContainer.ID)
		sub.TriggerCount.Add(1)
		now := time.Now()
		sub.LastTriggeredAt.Store(&now)

		log.Debug().Str("containerID", notificationContainer.ID).Interface("log", notificationLog.Message).Msg("Matched subscription")

		// Create notification
		notification := types.Notification{
			ID:        fmt.Sprintf("%s-%d", c.ID, time.Now().UnixNano()),
			Type:      types.LogNotification,
			Detail:    formatLogMessage(notificationLog.Message),
			Container: notificationContainer,
			Log:       &notificationLog,
			Subscription: types.SubscriptionConfig{
				ID:                  sub.ID,
				Name:                sub.Name,
				Enabled:             sub.Enabled,
				DispatcherID:        sub.DispatcherID,
				LogExpression:       sub.LogExpression,
				ContainerExpression: sub.ContainerExpression,
			},
			Timestamp: time.Now(),
		}

		// Send to the subscription's dispatcher
		if d, ok := m.getDispatcher(sub.DispatcherID); ok {
			go m.sendNotification(d, notification, sub.DispatcherID)
		}
		return true
	})
}

// processStatEvents processes stat events from the stats listener channel
func (m *Manager) processStatEvents() {
	for {
		select {
		case <-m.ctx.Done():
			return
		case event, ok := <-m.statsListener.Channel():
			if !ok || event == nil {
				return
			}
			m.processStatEvent(event)
		}
	}
}

// processStatEvent processes a single stat event and sends notifications for matching metric subscriptions
func (m *Manager) processStatEvent(event *ContainerStatEvent) {
	// Normalize CPU by core count so alerts report overall load (0-100%),
	// matching the UI. Stat.CPUPercent is per-core (100% = one full core).
	cores := event.Container.CPULimit
	if cores <= 0 {
		cores = float64(event.Host.NCPU)
	}
	if cores <= 0 {
		cores = 1
	}

	notificationStat := types.NotificationStat{
		CPUPercent:    event.Stat.CPUPercent / cores,
		MemoryPercent: event.Stat.MemoryPercent,
		MemoryUsage:   event.Stat.MemoryUsage,
		Mounts:        FromContainerMounts(event.Container),
	}

	notificationContainer := FromContainerModel(event.Container, event.Host)

	m.subscriptions.Range(func(_ int, sub *Subscription) bool {
		// Skip disabled or non-metric subscriptions
		if !sub.Enabled || !sub.IsMetricAlert() {
			return true
		}

		// Check container filter first
		if !sub.MatchesContainer(notificationContainer) {
			return true
		}

		// Evaluate metric expression and record in sample window
		matched := sub.MatchesMetric(notificationStat)
		if !sub.RecordMetricSample(event.Stat.ID, matched) {
			return true
		}

		// Check per-container cooldown
		if sub.IsMetricCooldownActive(event.Stat.ID) {
			return true
		}

		// Set cooldown and update stats
		sub.SetMetricCooldown(event.Stat.ID)
		sub.AddTriggeredContainer(event.Stat.ID)
		sub.TriggerCount.Add(1)
		now := time.Now()
		sub.LastTriggeredAt.Store(&now)

		log.Debug().
			Str("containerID", event.Stat.ID).
			Float64("cpu", notificationStat.CPUPercent).
			Float64("memory", notificationStat.MemoryPercent).
			Str("subscription", sub.Name).
			Msg("Metric alert triggered")

		notification := types.Notification{
			ID:        fmt.Sprintf("%s-metric-%d", event.Stat.ID, time.Now().UnixNano()),
			Type:      types.MetricNotification,
			Detail:    fmt.Sprintf("CPU: %.1f%%, Memory: %.1f%%", notificationStat.CPUPercent, notificationStat.MemoryPercent),
			Container: notificationContainer,
			Stat:      &notificationStat,
			Subscription: types.SubscriptionConfig{
				ID:                  sub.ID,
				Name:                sub.Name,
				Enabled:             sub.Enabled,
				DispatcherID:        sub.DispatcherID,
				MetricExpression:    sub.MetricExpression,
				ContainerExpression: sub.ContainerExpression,
				Cooldown:            sub.Cooldown,
				SampleWindow:        sub.SampleWindow,
			},
			Timestamp: time.Now(),
		}

		if d, ok := m.getDispatcher(sub.DispatcherID); ok {
			go m.sendNotification(d, notification, sub.DispatcherID)
		}
		return true
	})
}

// processDockerEvents processes Docker events from the event listener channel
func (m *Manager) processDockerEvents() {
	for {
		select {
		case <-m.ctx.Done():
			return
		case event, ok := <-m.eventListener.Channel():
			if !ok || event == nil {
				return
			}
			m.processDockerEvent(event)
		}
	}
}

// processDockerEvent processes a single Docker event and sends notifications for matching event subscriptions
func (m *Manager) processDockerEvent(event *ContainerEventEntry) {
	notificationContainer := FromContainerModel(event.Container, event.Host)
	notificationEvent := types.NotificationEvent{
		Name:       event.Event.Name,
		ActorID:    event.Event.ActorID,
		Attributes: event.Event.ActorAttributes,
		Timestamp:  event.Event.Time,
	}

	m.subscriptions.Range(func(_ int, sub *Subscription) bool {
		if !sub.Enabled || !sub.IsEventAlert() {
			return true
		}

		if !sub.MatchesContainer(notificationContainer) {
			return true
		}

		if !sub.MatchesEvent(notificationEvent) {
			return true
		}

		if sub.IsEventCooldownActive(event.Event.ActorID) {
			return true
		}

		if sub.Cooldown > 0 {
			sub.SetEventCooldown(event.Event.ActorID)
		}

		sub.AddTriggeredContainer(event.Event.ActorID)
		sub.TriggerCount.Add(1)
		now := time.Now()
		sub.LastTriggeredAt.Store(&now)

		log.Debug().
			Str("containerID", event.Event.ActorID).
			Str("event", event.Event.Name).
			Str("subscription", sub.Name).
			Msg("Event alert triggered")

		detail := fmt.Sprintf("Container event: %s", event.Event.Name)
		if exitCode, ok := event.Event.ActorAttributes["exitCode"]; ok && event.Event.Name == "die" {
			detail = fmt.Sprintf("Container event: %s (exit code %s)", event.Event.Name, describeExitCode(exitCode))
		}

		notification := types.Notification{
			ID:        fmt.Sprintf("%s-event-%d", event.Event.ActorID, time.Now().UnixNano()),
			Type:      types.EventNotification,
			Detail:    detail,
			Container: notificationContainer,
			Event:     &notificationEvent,
			Subscription: types.SubscriptionConfig{
				ID:                  sub.ID,
				Name:                sub.Name,
				Enabled:             sub.Enabled,
				DispatcherID:        sub.DispatcherID,
				EventExpression:     sub.EventExpression,
				ContainerExpression: sub.ContainerExpression,
				Cooldown:            sub.Cooldown,
			},
			Timestamp: time.Now(),
		}

		if d, ok := m.getDispatcher(sub.DispatcherID); ok {
			go m.sendNotification(d, notification, sub.DispatcherID)
		}
		return true
	})
}

// signalExitCodes maps the 128+N exit codes a container gets when a signal kills
// it to that signal's name.
//
// A bare "exit code 137" is routinely misread as an out-of-memory kill, both by
// people and by Dozzle Cloud's triage model, which was shipping headlines like
// "exited with code 137 — likely OOM" for ordinary Watchtower update cycles.
// 137 is SIGKILL: it is what `docker stop`, a host reboot, a redeploy, or an
// image updater produces whenever a container does not exit on SIGTERM within
// the grace period. A genuine out-of-memory kill arrives as its own "oom" event,
// so naming the signal costs nothing and removes the ambiguity at the source.
//
// Only codes that mean the same signal everywhere Docker runs are listed; the
// rest are returned unchanged rather than risking a wrong name.
var signalExitCodes = map[string]string{
	"129": "SIGHUP",
	"130": "SIGINT",
	"131": "SIGQUIT",
	"134": "SIGABRT",
	"137": "SIGKILL",
	"139": "SIGSEGV",
	"143": "SIGTERM",
}

// describeExitCode annotates a container exit code with its signal name when it
// has one, and otherwise returns it unchanged.
func describeExitCode(exitCode string) string {
	if signal, ok := signalExitCodes[exitCode]; ok {
		return exitCode + ", " + signal
	}
	return exitCode
}

func formatLogMessage(message any) string {
	switch v := message.(type) {
	case string:
		return v
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(b)
	}
}

// sendNotification sends a notification using the dispatcher
func (m *Manager) sendNotification(d dispatcher.Dispatcher, notification types.Notification, id int) {
	acquireCtx, acquireCancel := context.WithTimeout(m.ctx, time.Minute)
	defer acquireCancel()
	if err := m.sendSem.Acquire(acquireCtx, 1); err != nil {
		log.Warn().Err(err).Int("subscription", id).Msg("Notification dropped: too many pending")
		return
	}
	defer m.sendSem.Release(1)

	ctx, cancel := context.WithTimeout(m.ctx, 30*time.Second)
	defer cancel()

	if err := d.Send(ctx, notification); err != nil {
		log.Error().Err(err).Int("subscription", id).Msg("Failed to send notification")
	}
}
