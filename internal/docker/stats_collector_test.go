package docker

import (
	"context"
	"testing"

	"github.com/amir20/dozzle/internal/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockedClient struct {
	mock.Mock
	container.Client
}

func (m *mockedClient) ListContainers(ctx context.Context, filter container.ContainerLabels) ([]container.Container, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]container.Container), args.Error(1)
}

func (m *mockedClient) FindContainer(ctx context.Context, id string) (container.Container, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(container.Container), args.Error(1)
}

func (m *mockedClient) ContainerEvents(ctx context.Context, events chan<- container.ContainerEvent) error {
	args := m.Called(ctx, events)
	return args.Error(0)
}

func (m *mockedClient) ContainerStats(ctx context.Context, id string, stats chan<- container.ContainerStat) error {
	args := m.Called(ctx, id, stats)
	return args.Error(0)
}

func (m *mockedClient) Host() container.Host {
	args := m.Called()
	return args.Get(0).(container.Host)
}

func startedCollector(ctx context.Context) *DockerStatsCollector {
	client := new(mockedClient)
	client.On("ListContainers", mock.Anything, mock.Anything).Return([]container.Container{
		{
			ID:    "1234",
			Name:  "test",
			State: "running",
		},
	}, nil)
	client.On("ContainerEvents", mock.Anything, mock.AnythingOfType("chan<- container.ContainerEvent")).
		Return(nil).
		Run(func(args mock.Arguments) {
			ctx := args.Get(0).(context.Context)
			<-ctx.Done()
		})
	client.On("ContainerStats", mock.Anything, mock.Anything, mock.AnythingOfType("chan<- container.ContainerStat")).
		Return(nil).
		Run(func(args mock.Arguments) {
			stats := args.Get(2).(chan<- container.ContainerStat)
			stats <- container.ContainerStat{
				ID: "1234",
			}
		})
	client.On("Host").Return(container.Host{
		ID: "localhost",
	})

	collector := NewDockerStatsCollector(client, container.ContainerLabels{})
	stats := make(chan container.ContainerStat)

	collector.Subscribe(ctx, stats)

	go collector.Start(ctx)

	<-stats

	return collector
}

func TestCancelers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	collector := startedCollector(ctx)

	_, ok := collector.cancelers.Load("1234")
	assert.True(t, ok, "canceler should be stored")

	assert.False(t, collector.Start(ctx), "second start should return false")
	assert.Equal(t, int32(2), collector.totalStarted.Load(), "total started should be 2")

	collector.Stop()

	assert.Equal(t, int32(1), collector.totalStarted.Load(), "total started should be 1")
}

func TestSecondStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	collector := startedCollector(ctx)

	assert.False(t, collector.Start(ctx), "second start should return false")
	assert.Equal(t, int32(2), collector.totalStarted.Load(), "total started should be 2")

	collector.Stop()
	assert.Equal(t, int32(1), collector.totalStarted.Load(), "total started should be 1")
}

func TestStop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	collector := startedCollector(ctx)
	collector.Stop()
	assert.Equal(t, int32(0), collector.totalStarted.Load(), "total started should be 1")
}
