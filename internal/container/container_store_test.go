package container

import (
	"context"
	"testing"

	"github.com/amir20/dozzle/internal/utils"
	"github.com/magiconair/properties/assert"
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

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	collector := &fakeStatsCollector{}
	store := NewContainerStore(ctx, client, collector, ContainerLabels{})
	containers, _ := store.ListContainers(ContainerLabels{})

	assert.Equal(t, containers[0].ID, "1234")
}

type fakeStatsCollector struct{}

func (f *fakeStatsCollector) Subscribe(_ context.Context, _ chan<- ContainerStat) {}
func (f *fakeStatsCollector) Start(_ context.Context) bool                        { return true }
func (f *fakeStatsCollector) Stop()                                               {}
