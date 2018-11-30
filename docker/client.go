package docker

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
	"io"
	"log"
	"sort"
	"strings"
)

type dockerClient struct {
	cli *client.Client
}

// Client is a proxy around the docker client
type Client interface {
	ListContainers() ([]Container, error)
	ContainerLogs(ctx context.Context, id string) (io.ReadCloser, error)
	Events(ctx context.Context) (<-chan events.Message, <-chan error)
}

// NewClient creates a new instance of Client
func NewClient() Client {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Fatal(err)
	}
	return &dockerClient{cli}
}

func (d *dockerClient) ListContainers() ([]Container, error) {
	list, err := d.cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		return nil, err
	}

	var containers []Container
	for _, c := range list {

		container := Container{
			ID:      c.ID[:12],
			Names:   c.Names,
			Name:    strings.TrimPrefix(c.Names[0], "/"),
			Image:   c.Image,
			ImageID: c.ImageID,
			Command: c.Command,
			Created: c.Created,
			State:   c.State,
			Status:  c.Status,
		}
		containers = append(containers, container)
	}

	sort.Slice(containers, func(i, j int) bool {
		return containers[i].Name < containers[j].Name
	})

	return containers, nil
}

func (d *dockerClient) ContainerLogs(ctx context.Context, id string) (io.ReadCloser, error) {
	options := types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true, Follow: true, Tail: "300", Timestamps: true}
	return d.cli.ContainerLogs(ctx, id, options)
}

func (d *dockerClient) Events(ctx context.Context) (<-chan events.Message, <-chan error) {
	return d.cli.Events(ctx, types.EventsOptions{})
}
