package rpc

import (
	"context"
	"log"
	"net"

	"github.com/amir20/dozzle/internal/docker"
	"github.com/amir20/dozzle/internal/rpc/protos"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type server struct {
	client docker.Client
	protos.UnimplementedStreamServiceServer
}

func NewServer(client docker.Client) protos.StreamServiceServer {
	return &server{client: client}
}

func (s *server) StreamLogs(in *protos.StreamLogsRequest, out protos.StreamService_StreamLogsServer) error {
	s.client.ContainerLogs(out.Context(), in.Id, in.Since.AsTime(), docker.STDALL)
}

func (s *server) StreamEvents(in *protos.StreamEventsRequest, out protos.StreamService_StreamEventsServer) error {
	return nil
}

func (s *server) StreamStats(in *protos.StreamStatsRequest, out protos.StreamService_StreamStatsServer) error {
	return nil
}

func (s *server) FindContainer(ctx context.Context, in *protos.FindContainerRequest) (*protos.FindContainerResponse, error) {
	container, err := s.client.FindContainer(in.ContainerId)
	if err != nil {
		return nil, err
	}

	return &protos.FindContainerResponse{
		Container: &protos.Container{
			Id:      container.ID,
			Name:    container.Name,
			Image:   container.Image,
			ImageId: container.ImageID,
			// Command:   container.Command,
			Created: timestamppb.New(container.Created),
			State:   container.State,
			Status:  container.Status,
			Health:  container.Health,
			Host:    container.Host,
			Tty:     container.Tty,
			Labels:  container.Labels,
			Group:   container.Group,
			Started: timestamppb.New(*container.StartedAt),
		},
	}, nil

}

func (s *server) ListContainers(ctx context.Context, in *protos.ListContainersRequest) (*protos.ListContainersResponse, error) {
	return nil, nil
}

func runServer() {
	// create listiner
	lis, err := net.Listen("tcp", ":50005")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// create grpc server
	s := grpc.NewServer()
	protos.RegisterStreamServiceServer(s, &server{})

	log.Println("start server")
	// and start...
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

}
