package docker

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func startedCollector(ctx context.Context) *StatsCollector {
	client := new(mockedClient)
	client.On("ListContainers").Return([]Container{
		{
			ID:    "1234",
			Name:  "test",
			State: "running",
		},
	}, nil)
	client.On("Events", mock.Anything, mock.AnythingOfType("chan<- docker.ContainerEvent")).Return(make(chan error))
	client.On("ContainerStats", mock.Anything, mock.Anything, mock.AnythingOfType("chan<- docker.ContainerStat")).
		Return(nil).
		Run(func(args mock.Arguments) {
			stats := args.Get(2).(chan<- ContainerStat)
			stats <- ContainerStat{
				ID: "1234",
			}
		})

	collector := NewStatsCollector(client)
	stats := make(chan ContainerStat)

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
