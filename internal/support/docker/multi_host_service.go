package docker_support

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/migration"
	"github.com/amir20/dozzle/internal/notification"
	"github.com/amir20/dozzle/internal/notification/dispatcher"
	container_support "github.com/amir20/dozzle/internal/support/container"
	"github.com/amir20/dozzle/types"
	"github.com/rs/zerolog/log"
	lop "github.com/samber/lo/parallel"
)

type HostUnavailableError struct {
	Host container.Host
	Err  error
}

func (h *HostUnavailableError) Error() string {
	return fmt.Sprintf("host %s unavailable: %v", h.Host.ID, h.Err)
}

type ClientManager interface {
	Find(id string) (container_support.ClientService, bool)
	List() []container_support.ClientService
	RetryAndList() ([]container_support.ClientService, []error)
	Subscribe(ctx context.Context, channel chan<- container.Host)
	Hosts(ctx context.Context) []container.Host
	LocalClients() []container.Client
	LocalClientServices() []container_support.ClientService
}

type MultiHostService struct {
	manager             ClientManager
	timeout             time.Duration
	notificationManager *notification.Manager
	cloudConfig         *notification.CloudConfig
	cloudMu             sync.RWMutex
}

func NewMultiHostService(manager ClientManager, timeout time.Duration) *MultiHostService {
	m := &MultiHostService{
		manager: manager,
		timeout: timeout,
	}

	return m
}

func (m *MultiHostService) FindContainer(host string, id string, labels container.ContainerLabels) (*container_support.ContainerService, error) {
	client, ok := m.manager.Find(host)
	if !ok {
		return nil, fmt.Errorf("host %s not found", host)
	}
	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()
	container, err := client.FindContainer(ctx, id, labels)
	if err != nil {
		return nil, err
	}

	return container_support.NewContainerService(client, container), nil
}

func (m *MultiHostService) ListContainersForHost(host string, labels container.ContainerLabels) ([]container.Container, error) {
	client, ok := m.manager.Find(host)
	if !ok {
		return nil, fmt.Errorf("host %s not found", host)
	}
	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()

	return client.ListContainers(ctx, labels)
}

func (m *MultiHostService) ListAllContainers(labels container.ContainerLabels) ([]container.Container, []error) {
	clients, errors := m.manager.RetryAndList()

	type result struct {
		containers []container.Container
		err        error
	}

	results := lop.Map(clients, func(client container_support.ClientService, _ int) result {
		ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
		defer cancel()

		list, err := client.ListContainers(ctx, labels)
		if err != nil {
			host, _ := client.Host(ctx)
			log.Debug().Err(err).Str("host", host.Name).Msg("error listing containers")
			host.Available = false
			return result{nil, &HostUnavailableError{Host: host, Err: err}}
		}

		return result{list, nil}
	})

	containers := make([]container.Container, 0)
	for _, r := range results {
		if r.err != nil {
			errors = append(errors, r.err)
		} else {
			containers = append(containers, r.containers...)
		}
	}

	return containers, errors
}

func (m *MultiHostService) ListAllContainersFiltered(userLabels container.ContainerLabels, filter container_support.ContainerFilter) ([]container.Container, []error) {
	containers, err := m.ListAllContainers(userLabels)
	filtered := make([]container.Container, 0, len(containers))
	for _, container := range containers {
		if filter(&container) {
			filtered = append(filtered, container)
		}
	}
	return filtered, err
}

func (m *MultiHostService) SubscribeEventsAndStats(ctx context.Context, events chan<- container.ContainerEvent, stats chan<- container.ContainerStat) {
	for _, client := range m.manager.List() {
		client.SubscribeEvents(ctx, events)
		client.SubscribeStats(ctx, stats)
	}
}

func (m *MultiHostService) SubscribeContainersStarted(ctx context.Context, containers chan<- container.Container, filter container_support.ContainerFilter) {
	newContainers := make(chan container.Container)
	for _, client := range m.manager.List() {
		client.SubscribeContainersStarted(ctx, newContainers)
	}
	go func() {
		<-ctx.Done()
		close(newContainers)
	}()

	go func() {
		for container := range newContainers {
			if filter(&container) {
				select {
				case containers <- container:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
}

func (m *MultiHostService) Hosts() []container.Host {
	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()
	return m.manager.Hosts(ctx)
}

func (m *MultiHostService) LocalHost() (container.Host, error) {
	for _, host := range m.Hosts() {
		if host.Type == "local" {
			return host, nil
		}
	}
	return container.Host{}, fmt.Errorf("local host not found")
}

func (m *MultiHostService) SubscribeAvailableHosts(ctx context.Context, hosts chan<- container.Host) {
	m.manager.Subscribe(ctx, hosts)
}

func (m *MultiHostService) LocalClients() []container.Client {
	return m.manager.LocalClients()
}

func (m *MultiHostService) LocalClientServices() []container_support.ClientService {
	return m.manager.LocalClientServices()
}

func (m *MultiHostService) TotalClients() int {
	return len(m.manager.List())
}

const notificationConfigPath = "./data/notifications.yml"
const cloudConfigPath = "./data/cloud.yml"

// StartNotificationManager initializes and starts the notification manager
func (m *MultiHostService) StartNotificationManager(ctx context.Context) error {
	clients := m.manager.LocalClientServices()
	listener := notification.NewContainerLogListener(ctx, clients)
	statsListener := notification.NewContainerStatsListener(ctx, clients)
	eventListener := notification.NewContainerEventListener(ctx, clients)
	m.notificationManager = notification.NewManager(listener, statsListener, eventListener)

	// Migrate old config format before loading (splits cloud into cloud.yml)
	migration.MigrateCloudConfig(notificationConfigPath, cloudConfigPath)

	// Start first so matcher is available for LoadConfig
	if err := m.notificationManager.Start(); err != nil {
		return err
	}

	// Load notification config
	if file, err := os.Open(notificationConfigPath); err == nil {
		defer file.Close()
		if err := m.notificationManager.LoadConfig(file); err != nil {
			log.Warn().Err(err).Msg("Could not load notification config")
		} else {
			log.Debug().Str("path", notificationConfigPath).Msg("Loaded notification config")
		}
	}

	// Load cloud config
	if file, err := os.Open(cloudConfigPath); err == nil {
		defer file.Close()
		cc, err := notification.LoadCloudConfig(file)
		if err != nil {
			log.Warn().Err(err).Msg("Could not load cloud config")
		} else {
			m.cloudConfig = &cc
			m.setCloudDispatcherFromConfig(&cc)
			log.Debug().Str("path", cloudConfigPath).Msg("Loaded cloud config")
		}
	}

	// Broadcast loaded config to any already-connected agents
	m.broadcastNotificationConfig()

	// Re-broadcast when new agents connect so they receive the current config
	hostCh := make(chan container.Host, 1)
	m.manager.Subscribe(ctx, hostCh)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case host := <-hostCh:
				if host.Available {
					log.Debug().Str("host", host.Name).Msg("New host available, broadcasting notification config")
					m.broadcastNotificationConfig()
				}
			}
		}
	}()

	return nil
}

func (m *MultiHostService) saveNotificationConfig() {
	if err := os.MkdirAll("./data", 0755); err != nil {
		log.Error().Err(err).Msg("Could not create data directory")
		return
	}

	file, err := os.Create(notificationConfigPath)
	if err != nil {
		log.Error().Err(err).Msg("Could not create notification config file")
		return
	}
	defer file.Close()

	if err := m.notificationManager.WriteConfig(file); err != nil {
		log.Error().Err(err).Msg("Could not write notification config")
	}

	// Broadcast to all agents
	m.broadcastNotificationConfig()
}

// saveCloudConfig writes the current cloud config to cloudConfigPath and
// broadcasts the notification config to all agents so they receive the API key.
func (m *MultiHostService) saveCloudConfig() {
	m.cloudMu.RLock()
	cc := m.cloudConfig
	m.cloudMu.RUnlock()

	if cc == nil {
		return
	}

	if err := os.MkdirAll("./data", 0755); err != nil {
		log.Error().Err(err).Msg("Could not create data directory")
		return
	}

	file, err := os.Create(cloudConfigPath)
	if err != nil {
		log.Error().Err(err).Msg("Could not create cloud config file")
		return
	}
	defer file.Close()

	if err := notification.WriteCloudConfig(file, *cc); err != nil {
		log.Error().Err(err).Msg("Could not write cloud config")
	}

	m.broadcastNotificationConfig()
}

// CloudConfig returns the current cloud config, or nil if not set.
func (m *MultiHostService) CloudConfig() *notification.CloudConfig {
	m.cloudMu.RLock()
	defer m.cloudMu.RUnlock()
	return m.cloudConfig
}

// SetCloudConfig sets the cloud config, creates the cloud dispatcher, and persists to disk.
func (m *MultiHostService) SetCloudConfig(cc *notification.CloudConfig) {
	m.cloudMu.Lock()
	m.cloudConfig = cc
	m.cloudMu.Unlock()
	m.setCloudDispatcherFromConfig(cc)
	m.saveCloudConfig()
}

// RemoveCloudConfig clears the cloud config, removes the cloud dispatcher, deletes the file,
// and broadcasts the change to all agents so they stop sending to cloud.
func (m *MultiHostService) RemoveCloudConfig() {
	m.cloudMu.Lock()
	m.cloudConfig = nil
	m.cloudMu.Unlock()
	m.notificationManager.ClearCloudDispatcher()
	if err := os.Remove(cloudConfigPath); err != nil && !os.IsNotExist(err) {
		log.Error().Err(err).Msg("Could not remove cloud config file")
	}
	m.broadcastNotificationConfig()
}

// setCloudDispatcherFromConfig creates a CloudDispatcher from the given config and sets it on the manager.
func (m *MultiHostService) setCloudDispatcherFromConfig(cc *notification.CloudConfig) {
	d, err := dispatcher.NewCloudDispatcher("Dozzle Cloud", cc.APIKey, cc.Prefix, cc.ExpiresAt)
	if err != nil {
		log.Error().Err(err).Msg("Could not create cloud dispatcher from config")
		return
	}
	m.notificationManager.SetCloudDispatcher(d)
}

// NotificationConfigUpdater is an interface for clients that support notification config updates
type NotificationConfigUpdater interface {
	UpdateNotificationConfig(ctx context.Context, subscriptions []types.SubscriptionConfig, dispatchers []types.DispatcherConfig, cloudConfig *types.CloudConfig) error
}

// broadcastNotificationConfig sends current notification config to all agent clients
func (m *MultiHostService) broadcastNotificationConfig() {
	notifSubs := m.notificationManager.Subscriptions()
	notifDispatchers := m.notificationManager.Dispatchers()

	// Convert notification.Subscription to types.SubscriptionConfig
	subscriptions := make([]types.SubscriptionConfig, len(notifSubs))
	for i, sub := range notifSubs {
		subscriptions[i] = types.SubscriptionConfig{
			ID:                  sub.ID,
			Name:                sub.Name,
			Enabled:             sub.Enabled,
			DispatcherID:        sub.DispatcherID,
			LogExpression:       sub.LogExpression,
			ContainerExpression: sub.ContainerExpression,
			MetricExpression:    sub.MetricExpression,
			EventExpression:     sub.EventExpression,
			Cooldown:            sub.Cooldown,
			SampleWindow:        sub.SampleWindow,
		}
	}

	// Convert notification.DispatcherConfig to types.DispatcherConfig
	// Cloud dispatchers are excluded; cloud config is sent separately via CloudConfig.
	dispatchers := make([]types.DispatcherConfig, 0, len(notifDispatchers))
	for _, d := range notifDispatchers {
		if d.Type == "cloud" {
			continue
		}
		dispatchers = append(dispatchers, types.DispatcherConfig{
			ID:       d.ID,
			Name:     d.Name,
			Type:     d.Type,
			URL:      d.URL,
			Template: d.Template,
			Headers:  d.Headers,
		})
	}

	// Get cloud config for broadcasting to agents
	m.cloudMu.RLock()
	ncc := m.cloudConfig
	m.cloudMu.RUnlock()

	var cc *types.CloudConfig
	if ncc != nil {
		cc = &types.CloudConfig{
			APIKey:    ncc.APIKey,
			Prefix:    ncc.Prefix,
			ExpiresAt: ncc.ExpiresAt,
		}
	}

	var wg sync.WaitGroup
	for _, client := range m.manager.List() {
		// Check if client supports notification config updates (agents do, local docker clients don't)
		if updater, ok := client.(NotificationConfigUpdater); ok {
			wg.Go(func() {
				ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
				defer cancel()
				if err := updater.UpdateNotificationConfig(ctx, subscriptions, dispatchers, cc); err != nil {
					log.Error().Err(err).Msg("Failed to broadcast notification config to agent")
				} else {
					log.Debug().Int("subscriptions", len(subscriptions)).Int("dispatchers", len(dispatchers)).Msg("Broadcasted notification config to agent")
				}
			})
		}
	}
	wg.Wait()
}

// NotificationHandler returns the notification manager as an agent.NotificationConfigHandler.
// This is used in swarm mode to pass the handler to the local agent server.
func (m *MultiHostService) NotificationHandler() *notification.Manager {
	return m.notificationManager
}

// AddSubscription adds a subscription to local manager and broadcasts to agents
func (m *MultiHostService) AddSubscription(sub *notification.Subscription) error {
	if err := m.notificationManager.AddSubscription(sub); err != nil {
		return err
	}
	m.saveNotificationConfig()
	return nil
}

// RemoveSubscription removes a subscription from local manager and broadcasts to agents
func (m *MultiHostService) RemoveSubscription(id int) {
	m.notificationManager.RemoveSubscription(id)
	m.saveNotificationConfig()
}

// AddDispatcher adds a dispatcher and returns its auto-generated ID
func (m *MultiHostService) AddDispatcher(d dispatcher.Dispatcher) int {
	id := m.notificationManager.AddDispatcher(d)
	m.saveNotificationConfig()
	return id
}

// UpdateDispatcher updates a dispatcher by ID
func (m *MultiHostService) UpdateDispatcher(id int, d dispatcher.Dispatcher) {
	m.notificationManager.UpdateDispatcher(id, d)
	m.saveNotificationConfig()
}

// RemoveDispatcher removes a dispatcher by ID
func (m *MultiHostService) RemoveDispatcher(id int) {
	m.notificationManager.RemoveDispatcher(id)
	m.saveNotificationConfig()
}

// ReplaceSubscription replaces a subscription with new data
func (m *MultiHostService) ReplaceSubscription(sub *notification.Subscription) error {
	if err := m.notificationManager.ReplaceSubscription(sub); err != nil {
		return err
	}
	m.saveNotificationConfig()
	return nil
}

// UpdateSubscription updates a subscription with the provided fields
func (m *MultiHostService) UpdateSubscription(id int, updates map[string]any) error {
	if err := m.notificationManager.UpdateSubscription(id, updates); err != nil {
		return err
	}
	m.saveNotificationConfig()
	return nil
}

// Subscriptions returns all subscriptions
func (m *MultiHostService) Subscriptions() []*notification.Subscription {
	return m.notificationManager.Subscriptions()
}

// Dispatchers returns all dispatchers
func (m *MultiHostService) Dispatchers() []notification.DispatcherConfig {
	return m.notificationManager.Dispatchers()
}

// NotificationStatsProvider is an interface for clients that can report notification stats
type NotificationStatsProvider interface {
	GetNotificationStats(ctx context.Context) ([]types.SubscriptionStats, error)
}

// FetchAgentNotificationStats fetches and aggregates notification stats from all agent clients
func (m *MultiHostService) FetchAgentNotificationStats() map[int]types.SubscriptionStats {
	// Collect providers
	var providers []NotificationStatsProvider
	for _, client := range m.manager.List() {
		if provider, ok := client.(NotificationStatsProvider); ok {
			providers = append(providers, provider)
		}
	}

	if len(providers) == 0 {
		return nil
	}

	// Fetch stats from all agents in parallel
	allStats := lop.Map(providers, func(provider NotificationStatsProvider, _ int) []types.SubscriptionStats {
		ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
		defer cancel()
		stats, err := provider.GetNotificationStats(ctx)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to fetch notification stats from agent")
			return nil
		}
		return stats
	})

	// Aggregate sequentially
	aggregated := make(map[int]types.SubscriptionStats)
	for _, stats := range allStats {
		for _, s := range stats {
			existing, ok := aggregated[s.SubscriptionID]
			if !ok {
				// Dedup container IDs from this agent
				seen := make(map[string]struct{}, len(s.TriggeredContainerIDs))
				deduped := make([]string, 0, len(s.TriggeredContainerIDs))
				for _, id := range s.TriggeredContainerIDs {
					if _, exists := seen[id]; !exists {
						seen[id] = struct{}{}
						deduped = append(deduped, id)
					}
				}
				s.TriggeredContainerIDs = deduped
				aggregated[s.SubscriptionID] = s
				continue
			}

			existing.TriggerCount += s.TriggerCount

			if s.LastTriggeredAt != nil && (existing.LastTriggeredAt == nil || s.LastTriggeredAt.After(*existing.LastTriggeredAt)) {
				existing.LastTriggeredAt = s.LastTriggeredAt
			}

			// Dedup container IDs across agents
			seen := make(map[string]struct{}, len(existing.TriggeredContainerIDs))
			for _, id := range existing.TriggeredContainerIDs {
				seen[id] = struct{}{}
			}
			for _, id := range s.TriggeredContainerIDs {
				if _, exists := seen[id]; !exists {
					seen[id] = struct{}{}
					existing.TriggeredContainerIDs = append(existing.TriggeredContainerIDs, id)
				}
			}
			aggregated[s.SubscriptionID] = existing
		}
	}

	return aggregated
}
