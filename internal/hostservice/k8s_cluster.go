package hostservice

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/container/k8s"
	"github.com/amir20/dozzle/internal/migration"
	"github.com/amir20/dozzle/internal/notification"
	"github.com/amir20/dozzle/internal/notification/dispatcher"
	"github.com/amir20/dozzle/types"
	"github.com/puzpuzpuz/xsync/v4"
	"github.com/rs/zerolog/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/tools/cache"
)

type K8sClusterService struct {
	client              *k8s.Service
	timeout             time.Duration
	hosts               *xsync.Map[string, container.Host] // node name -> host
	hostSubscribers     *xsync.Map[context.Context, chan<- container.Host]
	notificationManager *notification.Manager
	persister           *notification.Persister
}

func NewK8sClusterService(client *k8s.Client, timeout time.Duration, filter container.ContainerLabels) (*K8sClusterService, error) {
	nodes, err := client.Clientset.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	if len(nodes.Items) == 0 {
		return nil, fmt.Errorf("nodes not found")
	}

	m := &K8sClusterService{
		client:          k8s.NewService(client, filter),
		timeout:         timeout,
		hosts:           xsync.NewMap[string, container.Host](),
		hostSubscribers: xsync.NewMap[context.Context, chan<- container.Host](),
	}
	for _, node := range nodes.Items {
		m.hosts.Store(node.Name, nodeToHost(&node))
	}

	go m.watchNodes(context.Background(), client)

	return m, nil
}

func nodeToHost(node *corev1.Node) container.Host {
	available := false
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			available = condition.Status == corev1.ConditionTrue
		}
	}
	return container.Host{
		ID:            node.Name,
		Name:          node.Name,
		MemTotal:      node.Status.Capacity.Memory().Value(),
		NCPU:          int(node.Status.Capacity.Cpu().Value()),
		DockerVersion: node.Status.NodeInfo.ContainerRuntimeVersion,
		Type:          "k8s",
		Available:     available,
	}
}

// watchNodes keeps the host list current. Nodes were listed once at startup, so on
// an autoscaling cluster a pod scheduled onto a node added later belonged to a host
// the UI had never heard of, and a node that went away still looked healthy.
func (m *K8sClusterService) watchNodes(ctx context.Context, client *k8s.Client) {
	informer := coreinformers.NewNodeInformer(client.Clientset, 0, cache.Indexers{})

	upsert := func(obj any) {
		if node, ok := obj.(*corev1.Node); ok {
			m.setHost(nodeToHost(node))
		}
	}
	_, err := informer.AddEventHandlerWithOptions(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj any) { upsert(obj) },
		UpdateFunc: func(_, obj any) { upsert(obj) },
		DeleteFunc: func(obj any) {
			if tombstone, ok := obj.(cache.DeletedFinalStateUnknown); ok {
				obj = tombstone.Obj
			}
			if node, ok := obj.(*corev1.Node); ok {
				host := nodeToHost(node)
				host.Available = false
				m.setHost(host)
			}
		},
	}, cache.HandlerOptions{})
	if err != nil {
		log.Error().Err(err).Msg("could not watch kubernetes nodes")
		return
	}
	informer.RunWithContext(ctx)
}

// setHost records a host and tells subscribers, but only when something they can see
// changed: nodes report status every few seconds with nothing new in it.
func (m *K8sClusterService) setHost(host container.Host) {
	if previous, ok := m.hosts.Load(host.ID); ok && previous == host {
		return
	}
	m.hosts.Store(host.ID, host)
	// Bounded, so one SSE client that is not reading cannot stall the node informer for
	// everyone. A dropped update only leaves that client with a stale host until reload.
	timeout := time.After(250 * time.Millisecond)
	m.hostSubscribers.Range(func(ctx context.Context, ch chan<- container.Host) bool {
		select {
		case ch <- host:
		case <-ctx.Done():
			m.hostSubscribers.Delete(ctx)
		case <-timeout:
			log.Warn().Str("host", host.ID).Msg("subscriber is not reading host updates, dropping update")
		}
		return true
	})
}

func (m *K8sClusterService) FindContainer(host string, id string, labels container.ContainerLabels) (*container.ContainerService, error) {
	c, err := m.client.FindContainer(context.Background(), id, labels)
	if err != nil {
		return nil, err
	}

	return container.NewContainerService(m.client, c), nil
}

func (m *K8sClusterService) ListContainersForHost(host string, labels container.ContainerLabels) ([]container.Container, error) {
	containers, err := m.client.ListContainers(context.Background(), labels)
	if err != nil {
		return nil, err
	}

	filteredContainers := make([]container.Container, 0)
	for _, container := range containers {
		if container.Host == host {
			filteredContainers = append(filteredContainers, container)
		}
	}

	return filteredContainers, nil
}

func (m *K8sClusterService) ListAllContainers(labels container.ContainerLabels) ([]container.Container, []error) {
	containers, err := m.client.ListContainers(context.Background(), labels)
	if err != nil {
		return nil, []error{err}
	}
	return containers, nil
}

func (m *K8sClusterService) ListAllContainersFiltered(userLabels container.ContainerLabels, filter container.ContainerFilter) ([]container.Container, []error) {
	containers, err := m.ListAllContainers(userLabels)
	filtered := make([]container.Container, 0, len(containers))
	for _, container := range containers {
		if filter(&container) {
			filtered = append(filtered, container)
		}
	}
	return filtered, err
}

func (m *K8sClusterService) SubscribeEventsAndStats(ctx context.Context, events chan<- container.ContainerEvent, stats chan<- container.ContainerStat) {
	m.client.SubscribeEvents(ctx, events)
	m.client.SubscribeStats(ctx, stats)
}

func (m *K8sClusterService) SubscribeContainersStarted(ctx context.Context, containers chan<- container.Container, filter container.ContainerFilter) {
	newContainers := make(chan container.Container)
	m.client.SubscribeContainersStarted(ctx, newContainers)
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

func (m *K8sClusterService) Hosts() []container.Host {
	hosts := make([]container.Host, 0, m.hosts.Size())
	m.hosts.Range(func(_ string, host container.Host) bool {
		hosts = append(hosts, host)
		return true
	})
	slices.SortFunc(hosts, func(a, b container.Host) int { return strings.Compare(a.Name, b.Name) })
	return hosts
}

func (m *K8sClusterService) LocalHost() (container.Host, error) {
	return m.client.Client().Host(), nil
}

func (m *K8sClusterService) SubscribeAvailableHosts(ctx context.Context, hosts chan<- container.Host) {
	m.hostSubscribers.Store(ctx, hosts)
	go func() {
		<-ctx.Done()
		m.hostSubscribers.Delete(ctx)
	}()
}

func (m *K8sClusterService) LocalClients() []container.Client {
	return []container.Client{m.client.Client()}
}

func (m *K8sClusterService) LocalClientServices() []container.ClientService {
	return []container.ClientService{m.client}
}

// StartNotificationManager initializes and starts the notification manager for k8s mode
func (m *K8sClusterService) StartNotificationManager(ctx context.Context) error {
	clients := m.LocalClientServices()
	listener := notification.NewContainerLogListener(ctx, clients)
	statsListener := notification.NewContainerStatsListener(ctx, clients)
	eventListener := notification.NewContainerEventListener(ctx, clients)
	m.notificationManager = notification.NewManager(listener, statsListener, eventListener)
	m.persister = &notification.Persister{
		Manager:          m.notificationManager,
		NotificationPath: notification.DefaultNotificationConfigPath,
		CloudPath:        notification.DefaultCloudConfigPath,
	}

	migration.MigrateCloudConfig(m.persister.NotificationPath, m.persister.CloudPath)

	// Start first so matcher is available for LoadConfig
	if err := m.notificationManager.Start(); err != nil {
		return err
	}

	m.persister.Load()
	return nil
}

func (m *K8sClusterService) AddSubscription(sub *notification.Subscription) error {
	if err := m.notificationManager.AddSubscription(sub); err != nil {
		return err
	}
	m.persister.SaveNotifications()
	return nil
}

func (m *K8sClusterService) RemoveSubscription(id int) {
	m.notificationManager.RemoveSubscription(id)
	m.persister.SaveNotifications()
}

func (m *K8sClusterService) ReplaceSubscription(sub *notification.Subscription) error {
	if err := m.notificationManager.ReplaceSubscription(sub); err != nil {
		return err
	}
	m.persister.SaveNotifications()
	return nil
}

func (m *K8sClusterService) UpdateSubscription(id int, updates map[string]any) error {
	if err := m.notificationManager.UpdateSubscription(id, updates); err != nil {
		return err
	}
	m.persister.SaveNotifications()
	return nil
}

func (m *K8sClusterService) Subscriptions() []*notification.Subscription {
	return m.notificationManager.Subscriptions()
}

func (m *K8sClusterService) AddDispatcher(d dispatcher.Dispatcher) int {
	id := m.notificationManager.AddDispatcher(d)
	m.persister.SaveNotifications()
	return id
}

func (m *K8sClusterService) UpdateDispatcher(id int, d dispatcher.Dispatcher) {
	m.notificationManager.UpdateDispatcher(id, d)
	m.persister.SaveNotifications()
}

func (m *K8sClusterService) RemoveDispatcher(id int) {
	m.notificationManager.RemoveDispatcher(id)
	m.persister.SaveNotifications()
}

func (m *K8sClusterService) Dispatchers() []notification.DispatcherConfig {
	return m.notificationManager.Dispatchers()
}

func (m *K8sClusterService) FetchAgentNotificationStats() map[int]types.SubscriptionStats {
	return nil
}

func (m *K8sClusterService) CloudConfig() *notification.CloudConfig {
	return m.persister.CloudConfig()
}

func (m *K8sClusterService) SetCloudConfig(cc *notification.CloudConfig) {
	m.persister.SetCloudConfig(cc)
}

func (m *K8sClusterService) SetCloudStreamLogs(enabled bool) {
	m.persister.SetCloudStreamLogs(enabled)
}

func (m *K8sClusterService) RemoveCloudConfig() {
	m.persister.RemoveCloudConfig()
}

func (m *K8sClusterService) ResetCloudDispatcherBreaker() {
	m.notificationManager.ResetCloudDispatcherBreaker()
}
