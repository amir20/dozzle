package docker

import (
	"context"
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/stretchr/testify/mock"
)

type mockedClient struct {
	mock.Mock
	Client
}

func (m *mockedClient) ListContainers() ([]Container, error) {
	args := m.Called()
	return args.Get(0).([]Container), args.Error(1)
}

func (m *mockedClient) FindContainer(id string) (Container, error) {
	args := m.Called(id)
	return args.Get(0).(Container), args.Error(1)
}

func (m *mockedClient) Events(ctx context.Context, events chan<- ContainerEvent) <-chan error {
	args := m.Called(ctx, events)
	return args.Get(0).(chan error)
}

func (m *mockedClient) ContainerStats(ctx context.Context, id string, stats chan<- ContainerStat) error {
	args := m.Called(ctx, id, stats)
	return args.Error(0)
}

func TestContainerStore_List(t *testing.T) {

	client := new(mockedClient)
	client.On("ListContainers").Return([]Container{
		{
			ID:   "1234",
			Name: "test",
		},
	}, nil)
	client.On("Events", mock.Anything, mock.AnythingOfType("chan<- docker.ContainerEvent")).Return(make(chan error))
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	store := NewContainerStore(ctx, client)
	containers := store.List()

	assert.Equal(t, containers[0].ID, "1234")
}

func TestContainerStore_die(t *testing.T) {
	client := new(mockedClient)
	client.On("ListContainers").Return([]Container{
		{
			ID:    "1234",
			Name:  "test",
			State: "running",
		},
	}, nil)

	client.On("Events", mock.Anything, mock.AnythingOfType("chan<- docker.ContainerEvent")).Return(make(chan error)).
		Run(func(args mock.Arguments) {
			ctx := args.Get(0).(context.Context)
			events := args.Get(1).(chan<- ContainerEvent)
			go func() {
				events <- ContainerEvent{
					Name:    "die",
					ActorID: "1234",
					Host:    "localhost",
				}
				<-ctx.Done()
			}()
		})

	client.On("ContainerStats", mock.Anything, "1234", mock.AnythingOfType("chan<- docker.ContainerStat")).Return(nil)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	store := NewContainerStore(ctx, client)

	// Wait until we get the event
	events := make(chan ContainerEvent)
	store.Subscribe(ctx, events)
	<-events

	containers := store.List()
	assert.Equal(t, containers[0].State, "exited")
}
