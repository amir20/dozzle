package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
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
	Ping(ctx context.Context) (types.Ping, error)
}

// Client is a proxy around the docker client
type Client interface {
	ListContainers() ([]Container, error)
	FindContainer(string) (Container, error)
	ContainerLogs(context.Context, string, string) (io.ReadCloser, error)
	Events(context.Context) (<-chan ContainerEvent, <-chan error)
	ContainerLogsBetweenDates(context.Context, string, time.Time, time.Time) (io.ReadCloser, error)
	ContainerStats(context.Context, string, chan<- ContainerStat) error
	Ping(context.Context) (types.Ping, error)
}

// NewClientWithFilters creates a new instance of Client with docker filters
func NewClientWithFilters(f map[string][]string) Client {
	filterArgs := filters.NewArgs()
	for key, values := range f {
		for _, value := range values {
			filterArgs.Add(key, value)
		}
	}

	log.Debugf("filterArgs = %v", filterArgs)

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())

	if err != nil {
		log.Fatal(err)
	}

	return &dockerClient{cli, filterArgs}
}

func NewClientWithTlsAndFilter(f map[string][]string, connection string) Client {
	filterArgs := filters.NewArgs()
	for key, values := range f {
		for _, value := range values {
			filterArgs.Add(key, value)
		}
	}

	log.Debugf("filterArgs = %v", filterArgs)

	remoteUrl, err := url.Parse(connection)
	if err != nil {
		log.Fatal(err)
	}

	if remoteUrl.Scheme != "tcp" {
		log.Fatal("Only tcp scheme is supported")
	}

	host := remoteUrl.Hostname()
	basePath := "/certs"

	if _, err := os.Stat(filepath.Join(basePath, host)); !os.IsNotExist(err) {
		basePath = filepath.Join(basePath, host)
	}

	cacertPath := filepath.Join(basePath, "ca.pem")
	certPath := filepath.Join(basePath, "cert.pem")
	keyPath := filepath.Join(basePath, "key.pem")

	opts := []client.Opt{
		client.WithHost(connection),
	}

	if _, err := os.Stat(cacertPath); os.IsNotExist(err) {
		log.Debugf("%s does not exist, using plain HTTP", cacertPath)
	} else {
		log.Debugf("Using TLS client config with certs at: %s", basePath)
		opts = append(opts, client.WithTLSClientConfig(cacertPath, certPath, keyPath))
	}

	opts = append(opts, client.WithAPIVersionNegotiation())

	cli, err := client.NewClientWithOpts(opts...)

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
	if !found {
		return container, fmt.Errorf("unable to find container with id: %s", id)
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
			Health:  findBetweenParentheses(c.Status),
		}
		containers = append(containers, container)
	}

	sort.Slice(containers, func(i, j int) bool {
		return strings.ToLower(containers[i].Name) < strings.ToLower(containers[j].Name)
	})

	return containers, nil
}

func (d *dockerClient) ContainerStats(ctx context.Context, id string, stats chan<- ContainerStat) error {
	response, err := d.cli.ContainerStats(ctx, id, true)

	if err != nil {
		return err
	}

	go func() {
		log.Debugf("starting to stream stats for: %s", id)
		defer response.Body.Close()
		decoder := json.NewDecoder(response.Body)
		var v *types.StatsJSON
		for {
			if err := decoder.Decode(&v); err != nil {
				if err == context.Canceled || err == io.EOF {
					log.Debugf("stopping stats streaming for container %s", id)
					return
				}
				log.Errorf("decoder for stats api returned an unknown error %v", err)
			}

			ncpus := uint8(v.CPUStats.OnlineCPUs)
			if ncpus == 0 {
				ncpus = uint8(len(v.CPUStats.CPUUsage.PercpuUsage))
			}

			var (
				cpuDelta    = float64(v.CPUStats.CPUUsage.TotalUsage) - float64(v.PreCPUStats.CPUUsage.TotalUsage)
				systemDelta = float64(v.CPUStats.SystemUsage) - float64(v.PreCPUStats.SystemUsage)
				cpuPercent  = int64((cpuDelta / systemDelta) * float64(ncpus) * 100)
				memUsage    = int64(calculateMemUsageUnixNoCache(v.MemoryStats))
				memPercent  = int64(float64(memUsage) / float64(v.MemoryStats.Limit) * 100)
			)

			if cpuPercent > 0 || memUsage > 0 {
				select {
				case <-ctx.Done():
					return
				case stats <- ContainerStat{
					ID:            id,
					CPUPercent:    cpuPercent,
					MemoryPercent: memPercent,
					MemoryUsage:   memUsage,
				}:
				}
			}
		}
	}()

	return nil
}

func (d *dockerClient) ContainerLogs(ctx context.Context, id string, since string) (io.ReadCloser, error) {
	log.WithField("id", id).WithField("since", since).Debug("streaming logs for container")

	if since != "" {
		if millis, err := strconv.ParseInt(since, 10, 64); err == nil {
			since = time.UnixMicro(millis).Add(time.Millisecond).Format(time.RFC3339Nano)
		} else {
			log.WithError(err).Debug("unable to parse since")
		}
	}

	options := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Tail:       "300",
		Timestamps: true,
		Since:      since,
	}

	log.Debugf("streaming logs from Docker with option: %+v", options)
	reader, err := d.cli.ContainerLogs(ctx, id, options)
	if err != nil {
		return nil, err
	}

	containerJSON, err := d.cli.ContainerInspect(ctx, id)
	if err != nil {
		return nil, err
	}

	return newLogReader(reader, containerJSON.Config.Tty), nil
}

func (d *dockerClient) Events(ctx context.Context) (<-chan ContainerEvent, <-chan error) {
	dockerMessages, errors := d.cli.Events(ctx, types.EventsOptions{})
	messages := make(chan ContainerEvent)

	go func() {
		defer close(messages)

		for {
			select {
			case <-ctx.Done():
				return
			case message, ok := <-dockerMessages:
				if !ok {
					return
				}

				if message.Type == "container" && len(message.Actor.ID) > 0 {
					messages <- ContainerEvent{
						ActorID: message.Actor.ID[:12],
						Name:    message.Action,
					}
				}
			}
		}
	}()

	return messages, errors
}

func (d *dockerClient) ContainerLogsBetweenDates(ctx context.Context, id string, from time.Time, to time.Time) (io.ReadCloser, error) {
	options := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: true,
		Since:      from.Format(time.RFC3339),
		Until:      to.Format(time.RFC3339),
	}

	log.Debugf("fetching logs from Docker with option: %+v", options)

	reader, err := d.cli.ContainerLogs(ctx, id, options)

	if err != nil {
		return nil, err
	}

	containerJSON, err := d.cli.ContainerInspect(ctx, id)
	if err != nil {
		return nil, err
	}

	return newLogReader(reader, containerJSON.Config.Tty), nil
}

func (d *dockerClient) Ping(ctx context.Context) (types.Ping, error) {
	return d.cli.Ping(ctx)
}

var PARENTHESIS_RE = regexp.MustCompile(`\(([^)]+)\)`)

func findBetweenParentheses(s string) string {
	if results := PARENTHESIS_RE.FindStringSubmatch(s); results != nil {
		return results[1]
	}
	return ""
}
