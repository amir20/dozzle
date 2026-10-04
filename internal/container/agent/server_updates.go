package agent

import (
	"context"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/container/agent/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// updateHistory is the agent's own host's update record, or an Unimplemented
// error for a service that keeps none, which the server reads the same way as
// an agent too old to have the RPC.
func (s *server) updateHistory() (container.UpdateHistory, error) {
	if h, ok := s.service.(container.UpdateHistory); ok {
		return h, nil
	}
	return nil, status.Error(codes.Unimplemented, "this agent keeps no update history")
}

func (s *server) RecentUpdates(_ context.Context, _ *pb.RecentUpdatesRequest) (*pb.RecentUpdatesResponse, error) {
	history, err := s.updateHistory()
	if err != nil {
		return nil, err
	}
	events := history.RecentUpdates()
	resp := &pb.RecentUpdatesResponse{Events: make([]*pb.ContainerUpdateEvent, 0, len(events))}
	for _, e := range events {
		resp.Events = append(resp.Events, updateEventToProto(e))
	}
	return resp, nil
}

func (s *server) StreamUpdates(_ *pb.StreamUpdatesRequest, out pb.AgentService_StreamUpdatesServer) error {
	history, err := s.updateHistory()
	if err != nil {
		return err
	}
	// Buffered: the store drops a send that would block, and Send waits on
	// gRPC flow control.
	events := make(chan container.ContainerUpdateEvent, 16)
	history.SubscribeUpdates(out.Context(), events)
	for {
		select {
		case e := <-events:
			if err := out.Send(&pb.StreamUpdatesResponse{Event: updateEventToProto(e)}); err != nil {
				return err
			}
		case <-out.Context().Done():
			return nil
		}
	}
}
