package docker

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/goccy/go-json"

	"github.com/amir20/dozzle/internal/utils"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/api/types/system"
	"github.com/docker/docker/client"

	log "github.com/sirupsen/logrus"
)

type StdType int

const (
	UNKNOWN StdType = 1 << iota
	STDOUT
	STDERR
)
const STDALL = STDOUT | STDERR

func (s StdType) String() string {
	switch s {
	case STDOUT:
		return "stdout"
	case STDERR:
		return "stderr"
	case STDALL:
		return "all"
	default:
		return "unknown"
	}
}

type DockerCLI interface {
	ContainerList(context.Context, container.ListOptions) ([]types.Container, error)
	ContainerLogs(context.Context, string, container.LogsOptions) (io.ReadCloser, error)
	Events(context.Context, events.ListOptions) (<-chan events.Message, <-chan error)
	ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error)
	ContainerStats(ctx context.Context, containerID string, stream bool) (container.StatsResponseReader, error)
	Ping(ctx context.Context) (types.Ping, error)
	ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error
	ContainerStop(ctx context.Context, containerID string, options container.StopOptions) error
	ContainerRestart(ctx context.Context, containerID string, options container.StopOptions) error
	Info(ctx context.Context) (system.Info, error)
}

type Client interface {
	ListContainers() ([]Container, error)
	FindContainer(string) (Container, error)
	ContainerLogs(context.Context, string, time.Time, StdType) (io.ReadCloser, error)
	ContainerEvents(context.Context, chan<- ContainerEvent) error
	ContainerLogsBetweenDates(context.Context, string, time.Time, time.Time, StdType) (io.ReadCloser, error)
	ContainerStats(context.Context, string, chan<- ContainerStat) error
	Ping(context.Context) (types.Ping, error)
	Host() Host
	ContainerActions(action ContainerAction, containerID string) error
	IsSwarmMode() bool
	SystemInfo() system.Info
}

type httpClient struct {
	cli     DockerCLI
	filters filters.Args
	host    Host
	info    system.Info
}

func NewClient(cli DockerCLI, filters filters.Args, host Host) Client {
	info, err := cli.Info(context.Background())
	if err != nil {
		log.Errorf("unable to get docker info: %v", err)
	}

	host.NCPU = info.NCPU
	host.MemTotal = info.MemTotal
	host.DockerVersion = info.ServerVersion

	return &httpClient{
		cli:     cli,
		filters: filters,
		host:    host,
		info:    info,
	}
}

// NewClientWithFilters creates a new instance of Client with docker filters
func NewLocalClient(f map[string][]string, hostname string) (Client, error) {
	filterArgs := filters.NewArgs()
	for key, values := range f {
		for _, value := range values {
			filterArgs.Add(key, value)
		}
	}

	log.Debugf("filterArgs = %v", filterArgs)

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())

	if err != nil {
		return nil, err
	}

	info, err := cli.Info(context.Background())
	if err != nil {
		return nil, err
	}

	id := info.ID
	if info.Swarm.NodeID != "" {
		id = info.Swarm.NodeID
	}

	host := Host{
		ID:       id,
		Name:     info.Name,
		MemTotal: info.MemTotal,
		NCPU:     info.NCPU,
		Endpoint: "local",
		Type:     "local",
	}

	if hostname != "" {
		host.Name = hostname
	}

	return NewClient(cli, filterArgs, host), nil
}

func NewRemoteClient(f map[string][]string, host Host) (Client, error) {
	filterArgs := filters.NewArgs()
	for key, values := range f {
		for _, value := range values {
			filterArgs.Add(key, value)
		}
	}

	log.Debugf("filterArgs = %v", filterArgs)

	if host.URL.Scheme != "tcp" {
		log.Fatal("Only tcp scheme is supported")
	}

	opts := []client.Opt{
		client.WithHost(host.URL.String()),
	}

	if host.ValidCerts {
		log.Debugf("Using TLS client config with certs at: %s", filepath.Dir(host.CertPath))
		opts = append(opts, client.WithTLSClientConfig(host.CACertPath, host.CertPath, host.KeyPath))
	} else {
		log.Debugf("No valid certs found, using plain TCP")
	}

	opts = append(opts, client.WithAPIVersionNegotiation())

	cli, err := client.NewClientWithOpts(opts...)

	if err != nil {
		return nil, err
	}

	host.Type = "remote"

	return NewClient(cli, filterArgs, host), nil
}

// Finds a container by id, skipping the filters
func (d *httpClient) FindContainer(id string) (Container, error) {
	log.Debugf("finding container with id: %s", id)
	if json, err := d.cli.ContainerInspect(context.Background(), id); err == nil {
		return newContainerFromJSON(json, d.host.ID), nil
	} else {
		return Container{}, err
	}

}

func (d *httpClient) ContainerActions(action ContainerAction, containerID string) error {
	switch action {
	case Start:
		return d.cli.ContainerStart(context.Background(), containerID, container.StartOptions{})
	case Stop:
		return d.cli.ContainerStop(context.Background(), containerID, container.StopOptions{})
	case Restart:
		return d.cli.ContainerRestart(context.Background(), containerID, container.StopOptions{})
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}

func (d *httpClient) ListContainers() ([]Container, error) {
	log.Debugf("listing containers with filters: %v", d.filters)
	containerListOptions := container.ListOptions{
		Filters: d.filters,
		All:     true,
	}
	list, err := d.cli.ContainerList(context.Background(), containerListOptions)
	if err != nil {
		return nil, err
	}

	var containers = make([]Container, 0, len(list))
	for _, c := range list {
		containers = append(containers, newContainer(c, d.host.ID))
	}

	sort.Slice(containers, func(i, j int) bool {
		return strings.ToLower(containers[i].Name) < strings.ToLower(containers[j].Name)
	})

	return containers, nil
}

func (d *httpClient) ContainerStats(ctx context.Context, id string, stats chan<- ContainerStat) error {
	response, err := d.cli.ContainerStats(ctx, id, true)

	if err != nil {
		return err
	}

	defer response.Body.Close()
	decoder := json.NewDecoder(response.Body)
	var v *container.StatsResponse
	for {
		if err := decoder.Decode(&v); err != nil {
			return err
		}

		var (
			memPercent, cpuPercent float64
			mem, memLimit          float64
			previousCPU            uint64
			previousSystem         uint64
		)
		daemonOSType := response.OSType

		if daemonOSType != "windows" {
			previousCPU = v.PreCPUStats.CPUUsage.TotalUsage
			previousSystem = v.PreCPUStats.SystemUsage
			cpuPercent = calculateCPUPercentUnix(previousCPU, previousSystem, v)
			mem = calculateMemUsageUnixNoCache(v.MemoryStats)
			memLimit = float64(v.MemoryStats.Limit)
			memPercent = calculateMemPercentUnixNoCache(memLimit, mem)
		} else {
			cpuPercent = calculateCPUPercentWindows(v)
			mem = float64(v.MemoryStats.PrivateWorkingSet)
		}

		if cpuPercent > 0 || mem > 0 {
			select {
			case <-ctx.Done():
				return nil
			case stats <- ContainerStat{
				ID:            id,
				CPUPercent:    cpuPercent,
				MemoryPercent: memPercent,
				MemoryUsage:   mem,
			}:
			}
		}
	}
}

func (d *httpClient) ContainerLogs(ctx context.Context, id string, since time.Time, stdType StdType) (io.ReadCloser, error) {
	log.WithField("id", id).WithField("since", since).WithField("stdType", stdType).Debug("streaming logs for container")

	sinceQuery := since.Add(-50 * time.Millisecond).Format(time.RFC3339Nano)
	options := container.LogsOptions{
		ShowStdout: stdType&STDOUT != 0,
		ShowStderr: stdType&STDERR != 0,
		Follow:     true,
		Tail:       strconv.Itoa(100),
		Timestamps: true,
		Since:      sinceQuery,
	}

	reader, err := d.cli.ContainerLogs(ctx, id, options)
	if err != nil {
		return nil, err
	}

	return reader, nil
}

func (d *httpClient) ContainerEvents(ctx context.Context, messages chan<- ContainerEvent) error {
	dockerMessages, err := d.cli.Events(ctx, events.ListOptions{})

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-err:
			return err

		case message := <-dockerMessages:
			if message.Type == events.ContainerEventType && len(message.Actor.ID) > 0 {
				messages <- ContainerEvent{
					ActorID: message.Actor.ID[:12],
					Name:    string(message.Action),
					Host:    d.host.ID,
				}
			}
		}
	}
}

func (d *httpClient) ContainerLogsBetweenDates(ctx context.Context, id string, from time.Time, to time.Time, stdType StdType) (io.ReadCloser, error) {
	options := container.LogsOptions{
		ShowStdout: stdType&STDOUT != 0,
		ShowStderr: stdType&STDERR != 0,
		Timestamps: true,
		Since:      from.Format(time.RFC3339Nano),
		Until:      to.Format(time.RFC3339Nano),
	}

	log.Debugf("fetching logs from Docker with option: %+v", options)

	reader, err := d.cli.ContainerLogs(ctx, id, options)
	if err != nil {
		return nil, err
	}

	return reader, nil
}

func (d *httpClient) Ping(ctx context.Context) (types.Ping, error) {
	return d.cli.Ping(ctx)
}

func (d *httpClient) Host() Host {
	return d.host
}

func (d *httpClient) IsSwarmMode() bool {
	return d.info.Swarm.LocalNodeState != swarm.LocalNodeStateInactive
}

func (d *httpClient) SystemInfo() system.Info {
	return d.info
}

func newContainer(c types.Container, host string) Container {
	name := "no name"
	if len(c.Names) > 0 {
		name = strings.TrimPrefix(c.Names[0], "/")
	}

	group := ""
	if c.Labels["dev.dozzle.group"] != "" {
		group = c.Labels["dev.dozzle.group"]
	}
	return Container{
		ID:      c.ID[:12],
		Name:    name,
		Image:   c.Image,
		ImageID: c.ImageID,
		Command: c.Command,
		Created: time.Unix(c.Created, 0),
		State:   c.State,
		Host:    host,
		Labels:  c.Labels,
		Stats:   utils.NewRingBuffer[ContainerStat](300), // 300 seconds of stats
		Group:   group,
	}
}

func newContainerFromJSON(c types.ContainerJSON, host string) Container {
	name := "no name"
	if len(c.Name) > 0 {
		name = strings.TrimPrefix(c.Name, "/")
	}

	group := ""
	if c.Config.Labels["dev.dozzle.group"] != "" {
		group = c.Config.Labels["dev.dozzle.group"]
	}

	container := Container{
		ID:      c.ID[:12],
		Name:    name,
		Image:   c.Image,
		ImageID: c.Image,
		Command: strings.Join(c.Config.Entrypoint, " ") + " " + strings.Join(c.Config.Cmd, " "),
		State:   c.State.Status,
		Host:    host,
		Labels:  c.Config.Labels,
		Stats:   utils.NewRingBuffer[ContainerStat](300), // 300 seconds of stats
		Group:   group,
		Tty:     c.Config.Tty,
	}

	if startedAt, err := time.Parse(time.RFC3339Nano, c.State.StartedAt); err == nil {
		container.StartedAt = startedAt.UTC()
	}

	if createdAt, err := time.Parse(time.RFC3339Nano, c.Created); err == nil {
		container.Created = createdAt.UTC()
	}

	if c.State.Health != nil {
		container.Health = strings.ToLower(c.State.Health.Status)
	}

	return container
}
