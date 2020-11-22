package docker

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
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
	ContainerStats(ctx context.Context, containerID string, stream bool) (types.ContainerStats, error)
}

// Client is a proxy around the docker client
type Client interface {
	ListContainers() ([]Container, error)
	FindContainer(string) (Container, error)
	ContainerLogs(context.Context, string, int, string) (<-chan string, <-chan error)
	Events(context.Context) (<-chan events.Message, <-chan error)
	ContainerLogsBetweenDates(context.Context, string, time.Time, time.Time) ([]string, error)
	ContainerStats(context.Context, string, chan<- ContainerStat) error
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

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())

	if err != nil {
		log.Fatal(err)
	}

	return &dockerClient{cli, filterArgs}
}

func (d *dockerClient) FindContainer(id string) (Container, error) {
	var container Container
	containers, err := d.ListContainers()
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

func (d *dockerClient) ListContainers() ([]Container, error) {
	containerListOptions := types.ContainerListOptions{
		Filters: d.filters,
		All:     true,
	}
	list, err := d.cli.ContainerList(context.Background(), containerListOptions)
	if err != nil {
		return nil, err
	}

	var containers = make([]Container, 0, len(list))
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
		return strings.ToLower(containers[i].Name) < strings.ToLower(containers[j].Name)
	})

	return containers, nil
}

func logReader(reader io.ReadCloser, tty bool) func() (string, error) {
	if tty {
		scanner := bufio.NewScanner(reader)
		return func() (string, error) {
			if scanner.Scan() {
				return scanner.Text(), nil
			}

			return "", io.EOF
		}
	}
	hdr := make([]byte, 8)
	var buffer bytes.Buffer
	return func() (string, error) {
		buffer.Reset()
		_, err := reader.Read(hdr)
		if err != nil {
			return "", err
		}
		count := binary.BigEndian.Uint32(hdr[4:])
		_, err = io.CopyN(&buffer, reader, int64(count))
		if err != nil {
			return "", err
		}
		return buffer.String(), nil
	}
}

func (d *dockerClient) ContainerStats(ctx context.Context, id string, stats chan<- ContainerStat) error {
	response, err := d.cli.ContainerStats(ctx, id, true)

	if err != nil {
		return err
	}

	go func() {
		defer response.Body.Close()
		decoder := json.NewDecoder(response.Body)
		var v *types.StatsJSON
		for {
			if err := decoder.Decode(&v); err != nil {
				if err == context.Canceled || err == io.EOF {
					log.Debugf("stopping stats streaming for container %s", id)
					break
				}
				log.Errorf("decoder for stats api returned an unknown error %v", err)
			}

			var (
				cpuDelta    = float64(v.CPUStats.CPUUsage.TotalUsage) - float64(v.PreCPUStats.CPUUsage.TotalUsage)
				systemDelta = float64(v.CPUStats.SystemUsage) - float64(v.PreCPUStats.SystemUsage)
				cpuPercent  = int64((cpuDelta / systemDelta) * float64(len(v.CPUStats.CPUUsage.PercpuUsage)) * 100)
				memUsage    = int64(v.MemoryStats.Usage - v.MemoryStats.Stats["cache"])
				memPercent  = int64(float64(memUsage) / float64(v.MemoryStats.Limit) * 100)
			)

			if cpuPercent > 0 || memUsage > 0 {
				stats <- ContainerStat{
					ID:            id,
					CPUPercent:    cpuPercent,
					MemoryPercent: memPercent,
					MemoryUsage:   memUsage,
				}
			}
		}
	}()

	return nil
}

func (d *dockerClient) ContainerLogs(ctx context.Context, id string, tailSize int, since string) (<-chan string, <-chan error) {
	log.WithField("id", id).WithField("since", since).Debug("Streaming logs for container")

	options := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Tail:       strconv.Itoa(tailSize),
		Timestamps: true,
		Since:      since,
	}
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

	go func() {
		defer close(messages)
		defer close(errChannel)
		defer reader.Close()
		nextEntry := logReader(reader, containerJSON.Config.Tty)
		for {
			line, err := nextEntry()
			if err != nil {
				errChannel <- err
				break
			}
			select {
			case messages <- line:
			case <-ctx.Done():
			}
		}
	}()

	return messages, errChannel
}

func (d *dockerClient) Events(ctx context.Context) (<-chan events.Message, <-chan error) {
	return d.cli.Events(ctx, types.EventsOptions{})
}

func (d *dockerClient) ContainerLogsBetweenDates(ctx context.Context, id string, from time.Time, to time.Time) ([]string, error) {
	options := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: true,
		Since:      strconv.FormatInt(from.Unix(), 10),
		Until:      strconv.FormatInt(to.Unix(), 10),
	}
	reader, _ := d.cli.ContainerLogs(ctx, id, options)
	defer reader.Close()

	containerJSON, _ := d.cli.ContainerInspect(ctx, id)

	nextEntry := logReader(reader, containerJSON.Config.Tty)

	var messages []string
	for {
		line, err := nextEntry()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return nil, err
			}
		}
		messages = append(messages, line)
	}

	return messages, nil
}
