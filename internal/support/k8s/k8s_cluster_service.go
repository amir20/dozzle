package k8s_support

import (
	"context"
	"fmt"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/k8s"
	container_support "github.com/amir20/dozzle/internal/support/container"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type K8sClusterService struct {
	client  *K8sClientService
	timeout time.Duration
	hosts   []container.Host
}

func NewK8sClusterService(client *k8s.K8sClient, timeout time.Duration) (*K8sClusterService, error) {
	hosts := make([]container.Host, 0)
	nodes, err := client.Clientset.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	if len(nodes.Items) == 0 {
		return nil, fmt.Errorf("nodes not found")
	}

	for _, node := range nodes.Items {
		hosts = append(hosts, container.Host{
			ID:            node.Name,
			Name:          node.Name,
			MemTotal:      node.Status.Capacity.Memory().Value(),
			NCPU:          int(node.Status.Capacity.Cpu().Value()),
			DockerVersion: node.Status.NodeInfo.ContainerRuntimeVersion,
			Type:          "k8s",
			Available:     true,
		})
	}

	return &K8sClusterService{
		client:  NewK8sClientService(client, container.ContainerLabels{}),
		timeout: timeout,
		hosts:   hosts,
	}, nil
}

func (m *K8sClusterService) FindContainer(host string, id string, labels container.ContainerLabels) (*container_support.ContainerService, error) {
	container, err := m.client.FindContainer(context.Background(), id, labels)
	if err != nil {
		return nil, err
	}

	return container_support.NewContainerService(m.client, container), nil
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

func (m *K8sClusterService) ListAllContainersFiltered(userLabels container.ContainerLabels, filter container_support.ContainerFilter) ([]container.Container, []error) {
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

func (m *K8sClusterService) SubscribeContainersStarted(ctx context.Context, containers chan<- container.Container, filter container_support.ContainerFilter) {
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
	return m.hosts
}

func (m *K8sClusterService) LocalHost() (container.Host, error) {
	return m.client.client.Host(), nil
}

func (m *K8sClusterService) SubscribeAvailableHosts(ctx context.Context, hosts chan<- container.Host) {
}

func (m *K8sClusterService) LocalClients() []container.Client {
	return []container.Client{m.client.client}
}
