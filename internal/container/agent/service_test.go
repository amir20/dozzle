package agent

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/container/agent/pb"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

// droppingAgent refuses every stream, like an agent that is restarting.
type droppingAgent struct {
	pb.AgentServiceClient
	events, stats, started atomic.Int32
}

var errAgentDown = errors.New("agent down")

func (d *droppingAgent) StreamEvents(context.Context, *pb.StreamEventsRequest, ...grpc.CallOption) (grpc.ServerStreamingClient[pb.StreamEventsResponse], error) {
	d.events.Add(1)
	return nil, errAgentDown
}

func (d *droppingAgent) StreamStats(context.Context, *pb.StreamStatsRequest, ...grpc.CallOption) (grpc.ServerStreamingClient[pb.StreamStatsResponse], error) {
	d.stats.Add(1)
	return nil, errAgentDown
}

func (d *droppingAgent) StreamContainerStarted(context.Context, *pb.StreamContainerStartedRequest, ...grpc.CallOption) (grpc.ServerStreamingClient[pb.StreamContainerStartedResponse], error) {
	d.started.Add(1)
	return nil, errAgentDown
}

// An agent that restarts under the same id is never republished, so the service
// has to reopen its own streams or that host goes dark until Dozzle restarts.
func TestSubscribeReopensStreamsAfterTheyEnd(t *testing.T) {
	agent := &droppingAgent{}
	svc := &service{client: &Client{client: agent}, retryMin: time.Millisecond, retryMax: time.Millisecond}

	ctx, cancel := context.WithCancel(t.Context())
	svc.SubscribeEvents(ctx, make(chan container.ContainerEvent))
	svc.SubscribeStats(ctx, make(chan container.ContainerStat))
	svc.SubscribeContainersStarted(ctx, make(chan container.Container))

	assert.Eventually(t, func() bool {
		return agent.events.Load() >= 3 && agent.stats.Load() >= 3 && agent.started.Load() >= 3
	}, 5*time.Second, time.Millisecond)

	cancel()
	time.Sleep(20 * time.Millisecond)
	events := agent.events.Load()
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, events, agent.events.Load(), "streams must stop once the subscriber is gone")
}
