package agent

import (
	"context"
	"errors"
	"fmt"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/container/agent/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrNoUpdateHistory is returned by an agent that keeps no update events: one
// older than the RecentUpdates and StreamUpdates RPCs.
var ErrNoUpdateHistory = errors.New("agent keeps no update history")

func updatesErr(err error) error {
	if status.Code(err) == codes.Unimplemented {
		return fmt.Errorf("%w: %v", ErrNoUpdateHistory, err)
	}
	return rpcErrToErr(err)
}

// RecentUpdates is the agent's kept update events, oldest first.
func (c *Client) RecentUpdates(ctx context.Context) ([]container.ContainerUpdateEvent, error) {
	resp, err := c.client.RecentUpdates(ctx, &pb.RecentUpdatesRequest{})
	if err != nil {
		return nil, updatesErr(err)
	}
	events := make([]container.ContainerUpdateEvent, 0, len(resp.Events))
	for _, e := range resp.Events {
		events = append(events, updateEventFromProto(e))
	}
	return events, nil
}

// StreamUpdates sends the agent's update events to ch as they happen, until
// ctx ends or the stream breaks. It blocks.
func (c *Client) StreamUpdates(ctx context.Context, ch chan<- container.ContainerUpdateEvent) error {
	stream, err := c.client.StreamUpdates(ctx, &pb.StreamUpdatesRequest{})
	if err != nil {
		return updatesErr(err)
	}
	for {
		resp, err := stream.Recv()
		if err != nil {
			return updatesErr(err)
		}
		select {
		case ch <- updateEventFromProto(resp.Event):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
