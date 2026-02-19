package notification

import (
	"context"
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
			Detail:    fmt.Sprintf("%v", notificationLog.Message),
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
		if d, ok := m.dispatchers.Load(sub.DispatcherID); ok {
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
	notificationStat := types.NotificationStat{
		CPUPercent:    event.Stat.CPUPercent,
		MemoryPercent: event.Stat.MemoryPercent,
		MemoryUsage:   event.Stat.MemoryUsage,
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

		// Check metric expression
		if !sub.MatchesMetric(notificationStat) {
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
			Float64("cpu", event.Stat.CPUPercent).
			Float64("memory", event.Stat.MemoryPercent).
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
			},
			Timestamp: time.Now(),
		}

		if d, ok := m.dispatchers.Load(sub.DispatcherID); ok {
			go m.sendNotification(d, notification, sub.DispatcherID)
		}
		return true
	})
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
