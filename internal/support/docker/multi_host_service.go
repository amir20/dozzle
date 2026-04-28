package docker_support

import (
	"context"
	"fmt"
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
	persister           *notification.Persister

	cloudNotifyMu sync.RWMutex
	cloudNotifyFn func()
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
	// Swarm mode marks every host as "swarm" so the loop above never matches.
	// Fall back to the local docker client directly — its host ID is stable
	// per node and is what callers (cloud client instance ID, etc.) actually want.
	for _, client := range m.manager.LocalClients() {
		return client.Host(), nil
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

// StartNotificationManager initializes and starts the notification manager
func (m *MultiHostService) StartNotificationManager(ctx context.Context) error {
	clients := m.manager.LocalClientServices()
	listener := notification.NewContainerLogListener(ctx, clients)
	statsListener := notification.NewContainerStatsListener(ctx, clients)
	eventListener := notification.NewContainerEventListener(ctx, clients)
	m.notificationManager = notification.NewManager(listener, statsListener, eventListener)
	m.persister = &notification.Persister{
		Manager:          m.notificationManager,
		NotificationPath: notification.DefaultNotificationConfigPath,
		CloudPath:        notification.DefaultCloudConfigPath,
	}

	// Migrate old config format before loading (splits cloud into cloud.yml)
	migration.MigrateCloudConfig(m.persister.NotificationPath, m.persister.CloudPath)

	// Start first so matcher is available for LoadConfig
	if err := m.notificationManager.Start(); err != nil {
		return err
	}

	m.persister.Load()

	// Broadcast loaded config to any already-connected agents
	m.broadcastNotificationConfig()
	m.broadcastCloudConfig()

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
					log.Debug().Str("host", host.Name).Msg("New host available, broadcasting config")
					m.broadcastNotificationConfig()
					m.broadcastCloudConfig()
				}
			}
		}
	}()

	return nil
}

func (m *MultiHostService) saveNotificationConfig() {
	m.persister.SaveNotifications()
	m.broadcastNotificationConfig()
}

// CloudConfig returns the current cloud config, or nil if not set.
func (m *MultiHostService) CloudConfig() *notification.CloudConfig {
	return m.persister.CloudConfig()
}

// SetCloudConfig sets the cloud config, creates the cloud dispatcher, and persists to disk.
func (m *MultiHostService) SetCloudConfig(cc *notification.CloudConfig) {
	m.persister.SetCloudConfig(cc)
	m.broadcastCloudConfig()
}

// SetCloudStreamLogs updates the bulk-log-streaming privacy flag on the cloud
// config and persists it. Only affects the local cloud client; agents never
// stream logs directly to cloud.
func (m *MultiHostService) SetCloudStreamLogs(enabled bool) {
	m.persister.SetCloudStreamLogs(enabled)
}

// RemoveCloudConfig clears the cloud config, removes the cloud dispatcher, deletes the file,
// and broadcasts the change to all agents so they stop sending to cloud.
func (m *MultiHostService) RemoveCloudConfig() {
	m.persister.RemoveCloudConfig()
	m.broadcastCloudConfig()
}

// NotificationConfigUpdater is an interface for clients that support notification config updates
type NotificationConfigUpdater interface {
	UpdateNotificationConfig(ctx context.Context, subscriptions []types.SubscriptionConfig, dispatchers []types.DispatcherConfig) error
	UpdateCloudConfig(ctx context.Context, cloudConfig *types.CloudConfig) error
}

// broadcastNotificationConfig sends current notification config to all agent clients
func (m *MultiHostService) broadcastNotificationConfig() {
	notifSubs := m.notificationManager.Subscriptions()
	notifDispatchers := m.notificationManager.Dispatchers()

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

	// Cloud dispatchers are excluded; cloud config is broadcast separately.
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

	var wg sync.WaitGroup
	for _, client := range m.manager.List() {
		if updater, ok := client.(NotificationConfigUpdater); ok {
			wg.Go(func() {
				ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
				defer cancel()
				if err := updater.UpdateNotificationConfig(ctx, subscriptions, dispatchers); err != nil {
					log.Error().Err(err).Msg("Failed to broadcast notification config to agent")
				}
			})
		}
	}
	wg.Wait()
}

// broadcastCloudConfig sends current cloud config to all agent clients
func (m *MultiHostService) broadcastCloudConfig() {
	ncc := m.persister.CloudConfig()

	var cc *types.CloudConfig
	if ncc != nil {
		cc = &types.CloudConfig{
			APIKey:    ncc.APIKey,
			Prefix:    ncc.Prefix,
			ExpiresAt: ncc.ExpiresAt,
		}
	}

	var count int
	var wg sync.WaitGroup
	for _, client := range m.manager.List() {
		if updater, ok := client.(NotificationConfigUpdater); ok {
			count++
			wg.Go(func() {
				ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
				defer cancel()
				if err := updater.UpdateCloudConfig(ctx, cc); err != nil {
					log.Error().Err(err).Msg("Failed to broadcast cloud config to agent")
				}
			})
		}
	}
	wg.Wait()
	log.Debug().Int("agents", count).Bool("hasCloud", cc != nil).Msg("Broadcasted cloud config")
}

// NotificationHandler returns the notification manager as an agent.NotificationConfigHandler.
// This is used in swarm mode to pass the handler to the local agent server.
func (m *MultiHostService) NotificationHandler() *notification.Manager {
	return m.notificationManager
}

// SetCloudNotifyFunc registers a callback the agent server invokes after a
// peer broadcast so the local cloud client reconnects with the new API key.
func (m *MultiHostService) SetCloudNotifyFunc(fn func()) {
	m.cloudNotifyMu.Lock()
	m.cloudNotifyFn = fn
	m.cloudNotifyMu.Unlock()
}

func (m *MultiHostService) cloudNotify() {
	m.cloudNotifyMu.RLock()
	fn := m.cloudNotifyFn
	m.cloudNotifyMu.RUnlock()
	if fn != nil {
		fn()
	}
}

// SwarmNotificationHandler returns the agent-server handler for swarm replicas.
// Broadcasts persist to disk and update this replica's persister + cloud client.
func (m *MultiHostService) SwarmNotificationHandler() *swarmNotificationHandler {
	return &swarmNotificationHandler{
		Manager:   m.notificationManager,
		persister: m.persister,
		notify:    m.cloudNotify,
	}
}

type swarmNotificationHandler struct {
	*notification.Manager
	persister *notification.Persister
	notify    func()
}

func (h *swarmNotificationHandler) HandleNotificationConfig(subscriptions []types.SubscriptionConfig, dispatchers []types.DispatcherConfig) error {
	if err := h.Manager.HandleNotificationConfig(subscriptions, dispatchers); err != nil {
		return err
	}
	h.persister.SaveNotifications()
	return nil
}

// persister.SetCloudConfig calls applyCloudDispatcher → Manager.SetCloudDispatcher,
// and persister.RemoveCloudConfig calls Manager.ClearCloudDispatcher; we route
// through the persister so disk + manager stay in lockstep on every replica.
func (h *swarmNotificationHandler) SetCloudDispatcher(d dispatcher.Dispatcher) {
	cd, ok := d.(*dispatcher.CloudDispatcher)
	if !ok {
		log.Warn().Str("type", fmt.Sprintf("%T", d)).Msg("Cloud dispatcher type assertion failed in swarm handler, falling back to in-memory only")
		h.Manager.SetCloudDispatcher(d)
		return
	}
	cc := &notification.CloudConfig{
		APIKey:    cd.APIKey,
		Prefix:    cd.Prefix,
		ExpiresAt: cd.ExpiresAt,
	}
	h.persister.SetCloudConfig(cc)
	h.notify()
}

func (h *swarmNotificationHandler) ClearCloudDispatcher() {
	h.persister.RemoveCloudConfig()
	h.notify()
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
