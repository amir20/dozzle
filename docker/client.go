package docker

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
)

type dockerClient struct {
	cli     dockerProxy
	filters filters.Args
}

type dockerProxy interface {
	ContainerList(context.Context, types.ContainerListOptions) ([]types.Container, error)
	ContainerLogs(context.Context, string, types.ContainerLogsOptions) (io.ReadCloser, error)
	Events(context.Context, types.EventsOptions) (<-chan events.Message, <-chan error)
	ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error)
}

// Client is a proxy around the docker client
type Client interface {
	ListContainers(showAll bool) ([]Container, error)
	FindContainer(string) (Container, error)
	ContainerLogs(context.Context, string, int) (<-chan string, <-chan error)
	Events(context.Context) (<-chan events.Message, <-chan error)
	ContainerLogsBetweenDates(context.Context, string, time.Time, time.Time) (io.ReadCloser, error)
}

// NewClient creates a new instance of Client
func NewClient() Client {
	return NewClientWithFilters(map[string]string{})
}

// NewClientWithFilters creates a new instance of Client with docker filters
func NewClientWithFilters(f map[string]string) Client {
	filterArgs := filters.NewArgs()
	for k, v := range f {
		filterArgs.Add(k, v)
	}

	log.Debugf("filterArgs = %v", filterArgs)

	cli, err := client.NewClientWithOpts(client.FromEnv)

	if err != nil {
		log.Fatal(err)
	}

	return &dockerClient{cli, filterArgs}
}

func (d *dockerClient) FindContainer(id string) (Container, error) {
	var container Container
	containers, err := d.ListContainers(true)
	if err != nil {
		return container, err
	}

	found := false
	for _, c := range containers {
		if c.ID == id {
			container = c
			found = true
			break
		}
	}
	if found == false {
		return container, fmt.Errorf("Unable to find container with id: %s", id)
	}

	return container, nil
}

func (d *dockerClient) ListContainers(showAll bool) ([]Container, error) {
	containerListOptions := types.ContainerListOptions{
		Filters: d.filters,
		All:     showAll,
	}
	list, err := d.cli.ContainerList(context.Background(), containerListOptions)
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

	if containers == nil {
		containers = []Container{}
	}

	return containers, nil
}

func (d *dockerClient) ContainerLogs(ctx context.Context, id string, tailSize int) (<-chan string, <-chan error) {
	options := types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true, Follow: true, Tail: strconv.Itoa(tailSize), Timestamps: true}
	reader, err := d.cli.ContainerLogs(ctx, id, options)
	errChannel := make(chan error, 1)

	if err != nil {
		errChannel <- err
		close(errChannel)
		return nil, errChannel
	}

	messages := make(chan string)
	go func() {
		<-ctx.Done()
		reader.Close()
	}()

	containerJSON, _ := d.cli.ContainerInspect(ctx, id)

	if containerJSON.Config.Tty {
		go func() {
			defer close(messages)
			defer close(errChannel)
			defer reader.Close()
			scanner := bufio.NewScanner(reader)
			for scanner.Scan() {
				line := scanner.Text()
				select {
				case messages <- line:
				case <-ctx.Done():
				}
			}
		}()
	} else {
		go func() {
			defer close(messages)
			defer close(errChannel)
			defer reader.Close()

			hdr := make([]byte, 8)
			var buffer bytes.Buffer
			for {
				_, err := reader.Read(hdr)
				if err != nil {
					errChannel <- err
					break
				}
				count := binary.BigEndian.Uint32(hdr[4:])
				_, err = io.CopyN(&buffer, reader, int64(count))

				if err != nil {
					errChannel <- err
					break
				}
				select {
				case messages <- buffer.String():
				case <-ctx.Done():
				}
				buffer.Reset()
			}
		}()
	}

	return messages, errChannel
}

func (d *dockerClient) Events(ctx context.Context) (<-chan events.Message, <-chan error) {
	return d.cli.Events(ctx, types.EventsOptions{})
}

func (d *dockerClient) ContainerLogsBetweenDates(ctx context.Context, id string, from time.Time, to time.Time) (io.ReadCloser, error) {
	options := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: true,
		Since:      strconv.FormatInt(from.Unix(), 10),
		Until:      strconv.FormatInt(to.Unix(), 10),
	}
	reader, err := d.cli.ContainerLogs(ctx, id, options)

	if err != nil {
		return nil, err
	}

	return reader, nil
}
