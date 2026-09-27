package hostservice

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/migration"
	"github.com/amir20/dozzle/internal/notification"
	"github.com/amir20/dozzle/internal/notification/dispatcher"
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
	Find(id string) (container.ClientService, bool)
	List() []container.ClientService
	RetryAndList() ([]container.ClientService, []error)
	Subscribe(ctx context.Context, channel chan<- container.Host)
	Hosts(ctx context.Context) []container.Host
	LocalClients() []container.Client
	LocalClientServices() []container.ClientService
}

type MultiHostService struct {
	manager             ClientManager
	timeout             time.Duration
	notificationManager *notification.Manager
	persister           *notification.Persister
	cloudNotifyFn       atomic.Pointer[func()]
	// agents is the manager when it can take agents while running, else nil.
	agents agentAdder
	// configMu orders config pushes against RemoveAgent, so a broadcast that
	// listed an agent before it was removed cannot land after its config was
	// cleared and hand the removed agent the rules and cloud key back.
	configMu sync.Mutex
}

func NewMultiHostService(manager ClientManager, timeout time.Duration) *MultiHostService {
	m := &MultiHostService{
		manager: manager,
		timeout: timeout,
	}
	m.agents, _ = manager.(agentAdder)

	return m
}

func (m *MultiHostService) FindContainer(host string, id string, labels container.ContainerLabels) (*container.ContainerService, error) {
	client, ok := m.manager.Find(host)
	if !ok {
		return nil, fmt.Errorf("host %s not found", host)
	}
	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()
	c, err := client.FindContainer(ctx, id, labels)
	if err != nil {
		return nil, err
	}

	return container.NewContainerService(client, c), nil
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

	results := lop.Map(clients, func(client container.ClientService, _ int) result {
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

func (m *MultiHostService) ListAllContainersFiltered(userLabels container.ContainerLabels, filter container.ContainerFilter) ([]container.Container, []error) {
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
	m.followClients(ctx, func(client container.ClientService, _ bool) {
		client.SubscribeEvents(ctx, events)
		client.SubscribeStats(ctx, stats)
	})
}

func (m *MultiHostService) SubscribeContainersStarted(ctx context.Context, containers chan<- container.Container, filter container.ContainerFilter) {
	newContainers := make(chan container.Container)
	m.followClients(ctx, func(client container.ClientService, late bool) {
		client.SubscribeContainersStarted(ctx, newContainers)
		if !late {
			return
		}
		// A host that joins later is new to this subscriber along with everything
		// already running on it, so those count as started too.
		go func() {
			running, err := m.listLate(ctx, client)
			if err != nil {
				log.Debug().Err(err).Msg("could not list containers of a newly added host")
				return
			}
			for _, c := range running {
				if c.State != "running" {
					continue
				}
				select {
				case newContainers <- c:
				case <-ctx.Done():
					return
				}
			}
		}()
	})
	// newContainers is never closed: the stores sending into it drop their
	// subscription only after ctx ends, so a close would race their sends and panic.
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case container := <-newContainers:
				if filter(&container) {
					select {
					case containers <- container:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()
}

// listLate lists a newly joined host for one subscriber, bounded by the timeout
// and by that subscriber's lifetime.
func (m *MultiHostService) listLate(ctx context.Context, client container.ClientService) ([]container.Container, error) {
	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()
	return client.ListContainers(ctx, nil)
}

// followClients calls subscribe for every client now, then again for each client
// that becomes available later (an agent added from the UI, or one that was down
// at startup), with late set. Without it, a subscriber only ever sees the hosts
// that existed when it subscribed. Each client is handed over once, except after
// a re-key (see below).
func (m *MultiHostService) followClients(ctx context.Context, subscribe func(client container.ClientService, late bool)) {
	// Subscribe before listing, so a host added in between is not missed.
	hosts := make(chan container.Host, 8)
	m.manager.Subscribe(ctx, hosts)

	seen := map[container.ClientService]bool{}
	for _, client := range m.manager.List() {
		seen[client] = true
		subscribe(client, false)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case host := <-hosts:
				if !host.Available || host.Removed {
					continue
				}
				client, ok := m.manager.Find(host.ID)
				if !ok {
					continue
				}
				if seen[client] {
					// A re-key means the agent process restarted, which ended every
					// stream this subscriber had open to it, so subscribe again. Not
					// as late: its running containers are ones the subscriber has.
					if host.ReplacesID != "" {
						subscribe(client, false)
					}
					continue
				}
				seen[client] = true
				subscribe(client, true)
			}
		}
	}()
}

// agentAdder is a ClientManager that can take agents while running. Only the
// server-mode manager is one; swarm discovers its nodes itself.
type agentAdder interface {
	AddAgent(ctx context.Context, endpoint string, cert *tls.Certificate) (container.Host, error)
	RemoveAgent(endpoint string) error
	AgentHostID(endpoint string) string
}

var ErrAgentsUnsupported = errors.New("agents cannot be added in this mode")

// CanAddAgents reports whether AddAgent works in this mode.
func (m *MultiHostService) CanAddAgents() bool {
	return m.agents != nil
}

func (m *MultiHostService) AddAgent(ctx context.Context, endpoint string, cert *tls.Certificate) (container.Host, error) {
	if m.agents == nil {
		return container.Host{}, ErrAgentsUnsupported
	}
	return m.agents.AddAgent(ctx, endpoint, cert)
}

func (m *MultiHostService) RemoveAgent(endpoint string) error {
	if m.agents == nil {
		return ErrAgentsUnsupported
	}
	m.configMu.Lock()
	defer m.configMu.Unlock()
	m.clearAgentConfig(endpoint)
	return m.agents.RemoveAgent(endpoint)
}

// clearAgentConfig takes back the notification and cloud config the hub pushed
// to an agent, so it stops alerting and streaming to cloud once it is no longer
// served. Best effort: an agent that is down keeps what it had until restarted.
func (m *MultiHostService) clearAgentConfig(endpoint string) {
	id := m.agents.AgentHostID(endpoint)
	if id == "" {
		return
	}
	client, ok := m.manager.Find(id)
	if !ok {
		return
	}
	updater, ok := client.(NotificationConfigUpdater)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), min(m.timeout, 3*time.Second))
	defer cancel()
	if err := updater.UpdateNotificationConfig(ctx, nil, nil); err != nil {
		log.Debug().Err(err).Str("endpoint", endpoint).Msg("could not clear notification config on removed agent")
	}
	if err := updater.UpdateCloudConfig(ctx, nil); err != nil {
		log.Debug().Err(err).Str("endpoint", endpoint).Msg("could not clear cloud config on removed agent")
	}
}

func (m *MultiHostService) AgentHostID(endpoint string) string {
	if m.agents == nil {
		return ""
	}
	return m.agents.AgentHostID(endpoint)
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

func (m *MultiHostService) LocalClientServices() []container.ClientService {
	return m.manager.LocalClientServices()
}

// ClientServices returns every client this Dozzle serves — the local docker
// daemon, --remote-host daemons and --remote-agent agents alike. Unlike
// LocalClientServices it does not filter by client type, so a caller that needs
// the whole fleet (the cloud client) sees agents too.
//
// retry re-dials agents that were unreachable the last time they were tried, so
// one that came up since joins the returned set. It costs a connection attempt
// per still-unreachable agent, up to the configured timeout each — pass it on
// the periodic fan-out calls, not on per-container lookups.
func (m *MultiHostService) ClientServices(retry bool) []container.ClientService {
	if !retry {
		return m.manager.List()
	}
	services, _ := m.manager.RetryAndList()
	return services
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
// The persister only exists once StartNotificationManager has run. The shell reads
// this on every page render, so it is reachable before that and in modes that never
// start a notification manager at all; no persister means nothing is linked.
func (m *MultiHostService) CloudConfig() *notification.CloudConfig {
	if m.persister == nil {
		return nil
	}
	return m.persister.CloudConfig()
}

// SetCloudConfig sets the cloud config, creates the cloud dispatcher, and persists to disk.
func (m *MultiHostService) SetCloudConfig(cc *notification.CloudConfig) {
	m.persister.SetCloudConfig(cc)
	m.broadcastCloudConfig()
}

// SetCloudStreamLogs updates the bulk-log-streaming privacy flag on the cloud
// config, persists it, and broadcasts it.
//
// Agents connect to cloud themselves and stream their own logs, so the setting
// has to reach them: without the broadcast a user who turned streaming off saw
// it stop on the hub while every agent kept sending.
func (m *MultiHostService) SetCloudStreamLogs(enabled bool) {
	m.persister.SetCloudStreamLogs(enabled)
	m.broadcastCloudConfig()
}

// ResetCloudDispatcherBreaker clears the cloud dispatcher's auth circuit breaker
// so notifications resume immediately once the key is known good again.
func (m *MultiHostService) ResetCloudDispatcherBreaker() {
	m.notificationManager.ResetCloudDispatcherBreaker()
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
	// Held across read and send, so an older snapshot can never go out last.
	m.configMu.Lock()
	defer m.configMu.Unlock()
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
	m.configMu.Lock()
	defer m.configMu.Unlock()
	ncc := m.persister.CloudConfig()

	var cc *types.CloudConfig
	if ncc != nil {
		cc = &types.CloudConfig{
			APIKey:     ncc.APIKey,
			Prefix:     ncc.Prefix,
			ExpiresAt:  ncc.ExpiresAt,
			StreamLogs: ncc.StreamLogs,
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
	m.cloudNotifyFn.Store(&fn)
}

func (m *MultiHostService) cloudNotify() {
	if fn := m.cloudNotifyFn.Load(); fn != nil {
		(*fn)()
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

// SetCloudStreamLogs applies a peer replica's log-streaming choice. Routed
// through the persister like everything else here so disk and manager stay in
// lockstep across replicas.
func (h *swarmNotificationHandler) SetCloudStreamLogs(enabled *bool) {
	if enabled == nil {
		return
	}
	h.persister.SetCloudStreamLogs(*enabled)
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
	if m.notificationManager == nil {
		return nil
	}
	return m.notificationManager.Subscriptions()
}

// Dispatchers returns all dispatchers
func (m *MultiHostService) Dispatchers() []notification.DispatcherConfig {
	if m.notificationManager == nil {
		return nil
	}
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
