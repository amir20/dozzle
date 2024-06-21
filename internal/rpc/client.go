package rpc

import (
	"context"
	"io"
	"time"

	"github.com/amir20/dozzle/internal/docker"
	"github.com/amir20/dozzle/internal/rpc/pb"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/system"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type rpcClient struct {
	client pb.StreamServiceClient
}

func NewClient() docker.Client {
	conn, err := grpc.NewClient("localhost:7007", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to server: %v", err)

	}

	client := pb.NewStreamServiceClient(conn)
	return &rpcClient{client: client}
}

func (c *rpcClient) ContainerLogs(ctx context.Context, containerID string, since *time.Time, std docker.StdType) (io.ReadCloser, error) {
	return nil, nil
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

func (c *rpcClient) ListContainers() ([]docker.Container, error) {
	return nil, nil
}

func (c *rpcClient) Events(ctx context.Context, events chan<- docker.ContainerEvent) error {
	return nil
}

func (c *rpcClient) Host() *docker.Host {
	return nil
}

func (c *rpcClient) ContainerLogsBetweenDates(ctx context.Context, containerID string, since time.Time, until time.Time, std docker.StdType) (io.ReadCloser, error) {
	return nil, nil
}

func (c *rpcClient) ContainerStats(ctx context.Context, containerID string, stats chan<- docker.ContainerStat) error {
	return nil
}

func (c *rpcClient) Ping(ctx context.Context) (types.Ping, error) {
	return types.Ping{}, nil
}

func (c *rpcClient) ContainerActions(action string, containerID string) error {
	return nil
}

func (c *rpcClient) IsSwarmMode() bool {
	return false
}

func (c *rpcClient) SystemInfo() system.Info {
	return system.Info{}
}
