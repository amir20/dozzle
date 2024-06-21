package rpc

import (
	"context"
	"net"
	"time"

	"github.com/amir20/dozzle/internal/docker"
	"github.com/amir20/dozzle/internal/rpc/pb"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type server struct {
	client docker.Client
	pb.UnimplementedStreamServiceServer
}

func NewServer(client docker.Client) pb.StreamServiceServer {
	return &server{client: client}
}

func (s *server) StreamLogs(in *pb.StreamLogsRequest, out pb.StreamService_StreamLogsServer) error {
	var since *time.Time
	if in.Since != nil {
		time := in.Since.AsTime()
		since = &time
	}

	reader, err := s.client.ContainerLogs(out.Context(), in.ContainerId, since, docker.STDALL)
	if err != nil {
		return err
	}

	container, err := s.client.FindContainer(in.ContainerId)
	if err != nil {
		return err
	}

	g := docker.NewEventGenerator(reader, container)

	for {
		select {
		case event := <-g.Events:
			var message *anypb.Any
			switch event.Message.(type) {
			case string:
				message, err =
					anypb.New(&pb.SimpleMessage{
						Message: event.Message.(string),
					})
				if err != nil {
					log.Errorf("failed to create anypb: %v", err)
					continue
				}
			default:
				log.Errorf("unknown message type: %T", event.Message)
			}

			out.Send(&pb.StreamLogsResponse{
				Event: &pb.LogEvent{
					Message:     message,
					Timestamp:   timestamppb.New(time.Unix(event.Timestamp, 0)),
					Id:          event.Id,
					ContainerId: event.ContainerID,
				},
			})
		case <-out.Context().Done():
			return nil
		}
	}

}

func (s *server) StreamEvents(in *pb.StreamEventsRequest, out pb.StreamService_StreamEventsServer) error {
	return nil
}

func (s *server) StreamStats(in *pb.StreamStatsRequest, out pb.StreamService_StreamStatsServer) error {
	return nil
}

func (s *server) FindContainer(ctx context.Context, in *pb.FindContainerRequest) (*pb.FindContainerResponse, error) {
	container, err := s.client.FindContainer(in.ContainerId)
	if err != nil {
		return nil, err
	}

	return &pb.FindContainerResponse{
		Container: &pb.Container{
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

func (s *server) ListContainers(ctx context.Context, in *pb.ListContainersRequest) (*pb.ListContainersResponse, error) {
	return nil, nil
}

func RunAgentServer(client docker.Client) {
	lis, err := net.Listen("tcp", "localhost:7007")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterStreamServiceServer(s, &server{client: client})

	log.Infof("server listening on %s", lis.Addr().String())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
