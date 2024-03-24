package docker

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSubscribe(t *testing.T) {
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

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	collector := NewStatsCollector(client)
	stats := make(chan ContainerStat)

	collector.Subscribe(ctx, stats)

	_, ok := collector.subscribers.Load(ctx)
	assert.True(t, ok)

	go collector.Start(ctx)

	<-stats

	_, ok = collector.cancelers.Load("1234")
	assert.True(t, ok, "canceler should be stored")

	assert.False(t, collector.Start(ctx), "second start should return false")
}
