package k8s_support

import (
	"context"
	"io"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/k8s"

	container_support "github.com/amir20/dozzle/internal/support/container"
)

type k8sClientService struct {
	client *k8s.K8sClient
	store  *container.ContainerStore
}

func NewK8sClientService(client *k8s.K8sClient, filter container.ContainerFilter) container_support.ClientService {
	return &k8sClientService{
		client: client,
		store:  container.NewContainerStore(context.Background(), client, filter),
	}
}

func (k *k8sClientService) FindContainer(ctx context.Context, id string, filter container.ContainerFilter) (container.Container, error) {
	return k.store.FindContainer(id, filter)
}

func (k *k8sClientService) ListContainers(ctx context.Context, filter container.ContainerFilter) ([]container.Container, error) {
	return k.store.ListContainers(filter)
}

func (k *k8sClientService) Host(ctx context.Context) (container.Host, error) {
	return k.client.Host(), nil
}

func (k *k8sClientService) ContainerAction(ctx context.Context, container container.Container, action container.ContainerAction) error {
	return k.client.ContainerActions(ctx, action, container.ID)
}

func (k *k8sClientService) LogsBetweenDates(ctx context.Context, c container.Container, from time.Time, to time.Time, stdTypes container.StdType) (<-chan *container.LogEvent, error) {
	reader, err := k.client.ContainerLogsBetweenDates(ctx, c.ID, from, to, stdTypes)
	if err != nil {
		return nil, err
	}

	k8sReader := k8s.NewLogReader(reader)
	g := container.NewEventGenerator(ctx, k8sReader, c)
	return g.Events, nil
}

func (k *k8sClientService) RawLogs(ctx context.Context, container container.Container, from time.Time, to time.Time, stdTypes container.StdType) (io.ReadCloser, error) {
	return k.client.ContainerLogsBetweenDates(ctx, container.ID, from, to, stdTypes)
}

func (k *k8sClientService) StreamLogs(ctx context.Context, c container.Container, from time.Time, stdTypes container.StdType, events chan<- *container.LogEvent) error {
	reader, err := k.client.ContainerLogs(ctx, c.ID, from, stdTypes)
	if err != nil {
		return err
	}

	k8sReader := k8s.NewLogReader(reader)
	g := container.NewEventGenerator(ctx, k8sReader, c)
	for event := range g.Events {
		events <- event
	}

	select {
	case e := <-g.Errors:
		return e
	default:
		return nil
	}
}

func (k *k8sClientService) SubscribeStats(ctx context.Context, stats chan<- container.ContainerStat) {
	k.store.SubscribeStats(ctx, stats)
}

func (k *k8sClientService) SubscribeEvents(ctx context.Context, events chan<- container.ContainerEvent) {
	k.store.SubscribeEvents(ctx, events)
}

func (k *k8sClientService) SubscribeContainersStarted(ctx context.Context, containers chan<- container.Container) {
	k.store.SubscribeNewContainers(ctx, containers)
}
