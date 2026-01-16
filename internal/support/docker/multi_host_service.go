package docker_support

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/notification"
	"github.com/amir20/dozzle/internal/notification/dispatcher"
	container_support "github.com/amir20/dozzle/internal/support/container"
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

// StartNotificationManager initializes and starts the notification manager
func (m *MultiHostService) StartNotificationManager() error {
	clients := m.manager.LocalClientServices()
	listener := notification.NewContainerLogListener(clients)
	m.notificationManager = notification.NewManager(listener)

	// Start first so matcher is available for LoadConfig
	if err := m.notificationManager.Start(); err != nil {
		return err
	}

	// Load config if exists
	if file, err := os.Open(notificationConfigPath); err == nil {
		defer file.Close()
		if err := m.notificationManager.LoadConfig(file); err != nil {
			log.Warn().Err(err).Msg("Could not load notification config")
		} else {
			log.Debug().Str("path", notificationConfigPath).Msg("Loaded notification config")
		}
	}

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
}

// AddSubscription adds a subscription to local manager and broadcasts to agents
func (m *MultiHostService) AddSubscription(sub *notification.Subscription) error {
	// Add to local manager
	if err := m.notificationManager.AddSubscription(sub); err != nil {
		return err
	}

	// TODO: Broadcast to agents via gRPC when agent notification support is added

	m.saveNotificationConfig()
	return nil
}

// RemoveSubscription removes a subscription from local manager and broadcasts to agents
func (m *MultiHostService) RemoveSubscription(id int) {
	// Remove from local manager
	m.notificationManager.RemoveSubscription(id)

	// TODO: Broadcast to agents via gRPC when agent notification support is added

	m.saveNotificationConfig()
}

// AddDispatcher adds a dispatcher and returns its auto-generated ID
func (m *MultiHostService) AddDispatcher(d dispatcher.Dispatcher) int {
	id := m.notificationManager.AddDispatcher(d)
	m.saveNotificationConfig()
	return id
}

// RemoveDispatcher removes a dispatcher by ID
func (m *MultiHostService) RemoveDispatcher(id int) {
	m.notificationManager.RemoveDispatcher(id)
	m.saveNotificationConfig()
}

// Subscriptions returns all subscriptions
func (m *MultiHostService) Subscriptions() []notification.Subscription {
	return m.notificationManager.Subscriptions()
}

// Dispatchers returns all dispatchers
func (m *MultiHostService) Dispatchers() []notification.DispatcherConfig {
	return m.notificationManager.Dispatchers()
}
