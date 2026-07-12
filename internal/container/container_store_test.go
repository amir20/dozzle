package container

import (
	"context"
	"testing"

	"github.com/amir20/dozzle/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockedClient struct {
	mock.Mock
	Client
}

func (m *mockedClient) ListContainers(ctx context.Context, filter ContainerLabels) ([]Container, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]Container), args.Error(1)
}

func (m *mockedClient) FindContainer(ctx context.Context, id string) (Container, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(Container), args.Error(1)
}

func (m *mockedClient) ContainerEvents(ctx context.Context, events chan<- ContainerEvent) error {
	args := m.Called(ctx, events)
	return args.Error(0)
}

func (m *mockedClient) ContainerStats(ctx context.Context, id string, stats chan<- ContainerStat) error {
	args := m.Called(ctx, id, stats)
	return args.Error(0)
}

func (m *mockedClient) Host() Host {
	args := m.Called()
	return args.Get(0).(Host)
}

func TestContainerStore_List(t *testing.T) {

	client := new(mockedClient)
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{
		{
			ID:   "1234",
			Name: "test",
		},
	}, nil)
	client.On("ContainerEvents", mock.Anything, mock.AnythingOfType("chan<- container.ContainerEvent")).Return(nil).Run(func(args mock.Arguments) {
		ctx := args.Get(0).(context.Context)
		<-ctx.Done()
	})
	client.On("Host").Return(Host{
		ID: "localhost",
	})

	client.On("FindContainer", mock.Anything, "1234").Return(Container{
		ID:    "1234",
		Name:  "test",
		Image: "test",
		Stats: utils.NewRingBuffer[ContainerStat](300),
	}, nil)

	collector := &fakeStatsCollector{}
	store := NewContainerStore(t.Context(), client, collector, ContainerLabels{})
	containers, _ := store.ListContainers(ContainerLabels{})

	assert.Equal(t, containers[0].ID, "1234")
}

func TestContainerStore_die(t *testing.T) {
	client := new(mockedClient)
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{
		{
			ID:    "1234",
			Name:  "test",
			State: "running",
			Stats: utils.NewRingBuffer[ContainerStat](300),
		},
	}, nil)

	ready := make(chan struct{})
	client.On("ContainerEvents", mock.Anything, mock.AnythingOfType("chan<- container.ContainerEvent")).Return(nil).
		Run(func(args mock.Arguments) {
			ctx := args.Get(0).(context.Context)
			events := args.Get(1).(chan<- ContainerEvent)
			<-ready
			events <- ContainerEvent{
				Name:    "die",
				ActorID: "1234",
				Host:    "localhost",
			}
			<-ctx.Done()
		})
	client.On("Host").Return(Host{
		ID: "localhost",
	})

	client.On("ContainerStats", mock.Anything, "1234", mock.AnythingOfType("chan<- container.ContainerStat")).Return(nil)

	client.On("FindContainer", mock.Anything, "1234").Return(Container{
		ID:    "1234",
		Name:  "test",
		Image: "test",
		Stats: utils.NewRingBuffer[ContainerStat](300),
	}, nil)

	store := NewContainerStore(t.Context(), client, &fakeStatsCollector{}, ContainerLabels{})

	// Wait until we get the event
	events := make(chan ContainerEvent)
	store.SubscribeEvents(t.Context(), events)
	close(ready)
	<-events

	containers, _ := store.ListContainers(ContainerLabels{})
	assert.Equal(t, containers[0].State, "exited")
}

func TestContainerStore_rename(t *testing.T) {
	run := func(t *testing.T, initial Container, attributes map[string]string) Container {
		client := new(mockedClient)
		client.On("ListContainers", mock.Anything, mock.Anything).Return([]Container{initial}, nil)
		client.On("FindContainer", mock.Anything, initial.ID).Return(initial, nil)
		client.On("Host").Return(Host{ID: "localhost"})

		ready := make(chan struct{})
		client.On("ContainerEvents", mock.Anything, mock.AnythingOfType("chan<- container.ContainerEvent")).Return(nil).
			Run(func(args mock.Arguments) {
				ctx := args.Get(0).(context.Context)
				events := args.Get(1).(chan<- ContainerEvent)
				<-ready
				events <- ContainerEvent{
					Name:            "rename",
					ActorID:         initial.ID,
					Host:            "localhost",
					ActorAttributes: attributes,
				}
				<-ctx.Done()
			})

		store := NewContainerStore(t.Context(), client, &fakeStatsCollector{}, ContainerLabels{})

		events := make(chan ContainerEvent)
		store.SubscribeEvents(t.Context(), events)
		close(ready)
		<-events

		containers, err := store.ListContainers(ContainerLabels{})
		assert.NoError(t, err)
		assert.Len(t, containers, 1)
		return containers[0]
	}

	t.Run("keeps custom name from dev.dozzle.name label", func(t *testing.T) {
		initial := Container{
			ID:          "1234",
			Name:        "custom-name",
			State:       "running",
			FullyLoaded: true,
			Labels:      map[string]string{"dev.dozzle.name": "custom-name"},
			Stats:       utils.NewRingBuffer[ContainerStat](300),
		}
		result := run(t, initial, map[string]string{"name": "new-docker-name"})
		assert.Equal(t, "custom-name", result.Name)
	})

	t.Run("keeps custom name from coolify.serviceName label", func(t *testing.T) {
		initial := Container{
			ID:          "1234",
			Name:        "coolify-name",
			State:       "running",
			FullyLoaded: true,
			Labels:      map[string]string{"coolify.serviceName": "coolify-name"},
			Stats:       utils.NewRingBuffer[ContainerStat](300),
		}
		result := run(t, initial, map[string]string{"name": "new-docker-name"})
		assert.Equal(t, "coolify-name", result.Name)
	})

	t.Run("follows rename when name comes from docker", func(t *testing.T) {
		initial := Container{
			ID:          "1234",
			Name:        "old-docker-name",
			State:       "running",
			FullyLoaded: true,
			Stats:       utils.NewRingBuffer[ContainerStat](300),
		}
		result := run(t, initial, map[string]string{"name": "new-docker-name"})
		assert.Equal(t, "new-docker-name", result.Name)
	})
}

type fakeStatsCollector struct{}

func (f *fakeStatsCollector) Subscribe(_ context.Context, _ chan<- ContainerStat) {}
func (f *fakeStatsCollector) Start(_ context.Context) bool                        { return true }
func (f *fakeStatsCollector) Stop()                                               {}
