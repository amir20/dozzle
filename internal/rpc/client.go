package rpc

import (
	"context"
	"io"
	"time"

	"github.com/amir20/dozzle/internal/docker"
	"github.com/amir20/dozzle/internal/rpc/pb"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type rpcClient struct {
	client pb.StreamServiceClient
}

func NewClient() *rpcClient {
	conn, err := grpc.NewClient("localhost:7007", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to server: %v", err)

	}

	client := pb.NewStreamServiceClient(conn)
	return &rpcClient{client: client}
}

func (c *rpcClient) ContainerLogs(ctx context.Context, containerID string, since *time.Time, std docker.StdType, events chan<- docker.LogEvent) error {
	stream, err := c.client.StreamLogs(ctx, &pb.StreamLogsRequest{ContainerId: containerID})

	if err != nil {
		return err
	}

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			log.Fatalf("cannot receive %v", err)
		}

		// unpack message from any type

		m, err := resp.Event.Message.UnmarshalNew()
		if err != nil {
			log.Fatalf("cannot unpack message %v", err)
		}

		var message any
		switch m := m.(type) {
		case *pb.SimpleMessage:
			message = m.Message
		default:
			log.Fatalf("unknown type %T", m)
		}

		events <- docker.LogEvent{
			Id:          resp.Event.Id,
			ContainerID: resp.Event.ContainerId,
			Message:     message,
			Timestamp:   resp.Event.Timestamp.AsTime().Unix(),
		}
	}
}

func (c *rpcClient) FindContainer(containerID string) (docker.Container, error) {
	response, err := c.client.FindContainer(context.Background(), &pb.FindContainerRequest{ContainerId: containerID})
	if err != nil {
		return docker.Container{}, err
	}

	// TODO: convert response to docker.Container
	return docker.Container{
		ID:      response.Container.Id,
		Name:    response.Container.Name,
		Image:   response.Container.Image,
		Labels:  response.Container.Labels,
		Group:   response.Container.Group,
		ImageID: response.Container.ImageId,
		Created: response.Container.Created.AsTime(),
		State:   response.Container.State,
		Status:  response.Container.Status,
		Health:  response.Container.Health,
		Host:    response.Container.Host,
		Tty:     response.Container.Tty,
		Stats:   nil,
	}, nil
}
