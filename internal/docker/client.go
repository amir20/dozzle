package docker

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"encoding/json"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/utils"
	"github.com/docker/docker/api/types"
	docker "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/system"
	"github.com/docker/docker/client"

	"github.com/rs/zerolog/log"
)

type DockerCLI interface {
	ContainerList(context.Context, docker.ListOptions) ([]docker.Summary, error)
	ContainerLogs(context.Context, string, docker.LogsOptions) (io.ReadCloser, error)
	Events(context.Context, events.ListOptions) (<-chan events.Message, <-chan error)
	ContainerInspect(ctx context.Context, containerID string) (docker.InspectResponse, error)
	ContainerStats(ctx context.Context, containerID string, stream bool) (docker.StatsResponseReader, error)
	Ping(ctx context.Context) (types.Ping, error)
	ContainerStart(ctx context.Context, containerID string, options docker.StartOptions) error
	ContainerStop(ctx context.Context, containerID string, options docker.StopOptions) error
	ContainerRestart(ctx context.Context, containerID string, options docker.StopOptions) error
	ContainerAttach(ctx context.Context, containerID string, options docker.AttachOptions) (types.HijackedResponse, error)
	ContainerExecCreate(ctx context.Context, containerID string, options docker.ExecOptions) (docker.ExecCreateResponse, error)
	ContainerExecAttach(ctx context.Context, execID string, config docker.ExecAttachOptions) (types.HijackedResponse, error)
	ContainerExecResize(ctx context.Context, execID string, options docker.ResizeOptions) error
	Info(ctx context.Context) (system.Info, error)
}

type DockerClient struct {
	cli  DockerCLI
	host container.Host
	info system.Info
}

func NewClient(cli DockerCLI, host container.Host) *DockerClient {
	info, err := cli.Info(context.Background())
	if err != nil {
		log.Error().Err(err).Msg("Failed to get docker info")
	}

	id := info.ID
	if info.Swarm.NodeID != "" {
		id = info.Swarm.NodeID
	}

	host.ID = id
	host.NCPU = info.NCPU
	host.MemTotal = info.MemTotal
	host.DockerVersion = info.ServerVersion
	host.Swarm = info.Swarm.NodeID != ""

	return &DockerClient{
		cli:  cli,
		host: host,
		info: info,
	}
}

// NewLocalClient creates a new instance of Client with docker filters
func NewLocalClient(hostname string) (*DockerClient, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithUserAgent("Docker-Client/Dozzle"))

	if err != nil {
		return nil, err
	}

	info, err := cli.Info(context.Background())
	if err != nil {
		return nil, err
	}

	host := container.Host{
		Name:     info.Name,
		Endpoint: "local",
		Type:     "local",
	}

	if hostname != "" {
		host.Name = hostname
	}

	return NewClient(cli, host), nil
}

func NewRemoteClient(host container.Host) (*DockerClient, error) {
	if host.URL.Scheme != "tcp" {
		return nil, fmt.Errorf("invalid scheme: %s", host.URL.Scheme)
	}

	opts := []client.Opt{
		client.WithHost(host.URL.String()),
	}

	if host.ValidCerts {
		log.Debug().Str("caCertPath", host.CACertPath).Str("certPath", host.CertPath).Str("keyPath", host.KeyPath).Msg("Using TLS for remote client")
		opts = append(opts, client.WithTLSClientConfig(host.CACertPath, host.CertPath, host.KeyPath))
	} else {
		log.Debug().Msg("Not using TLS for remote client")
	}

	opts = append(opts, client.WithAPIVersionNegotiation(), client.WithUserAgent("Docker-Client/Dozzle"))

	cli, err := client.NewClientWithOpts(opts...)

	if err != nil {
		return nil, err
	}

	host.Type = "remote"

	return NewClient(cli, host), nil
}

// Finds a container by id, skipping the filters
func (d *DockerClient) FindContainer(ctx context.Context, id string) (container.Container, error) {
	log.Debug().Str("id", id).Msg("Finding container")
	if json, err := d.cli.ContainerInspect(ctx, id); err == nil {
		return newContainerFromJSON(json, d.host.ID), nil
	} else {
		return container.Container{}, err
	}

}

func (d *DockerClient) ContainerActions(ctx context.Context, action container.ContainerAction, containerID string) error {
	switch action {
	case container.Start:
		return d.cli.ContainerStart(ctx, containerID, docker.StartOptions{})
	case container.Stop:
		return d.cli.ContainerStop(ctx, containerID, docker.StopOptions{})
	case container.Restart:
		return d.cli.ContainerRestart(ctx, containerID, docker.StopOptions{})
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}

func (d *DockerClient) ListContainers(ctx context.Context, labels container.ContainerLabels) ([]container.Container, error) {
	log.Debug().Interface("labels", labels).Str("host", d.host.Name).Msg("Listing containers")
	filterArgs := filters.NewArgs()
	for key, values := range labels {
		for _, value := range values {
			filterArgs.Add(key, value)
		}
	}
	containerListOptions := docker.ListOptions{
		Filters: filterArgs,
		All:     true,
	}
	list, err := d.cli.ContainerList(ctx, containerListOptions)
	if err != nil {
		return nil, err
	}

	var containers = make([]container.Container, 0, len(list))
	for _, c := range list {
		containers = append(containers, newContainer(c, d.host.ID))
	}

	sort.Slice(containers, func(i, j int) bool {
		return strings.ToLower(containers[i].Name) < strings.ToLower(containers[j].Name)
	})

	return containers, nil
}

func (d *DockerClient) ContainerStats(ctx context.Context, id string, stats chan<- container.ContainerStat) error {
	response, err := d.cli.ContainerStats(ctx, id, true)

	if err != nil {
		return err
	}

	defer response.Body.Close()
	decoder := json.NewDecoder(response.Body)
	var v *docker.StatsResponse
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
			case stats <- container.ContainerStat{
				ID:            id,
				CPUPercent:    cpuPercent,
				MemoryPercent: memPercent,
				MemoryUsage:   mem,
			}:
			}
		}
	}
}

func (d *DockerClient) ContainerLogs(ctx context.Context, id string, since time.Time, stdType container.StdType) (io.ReadCloser, error) {
	log.Debug().Str("id", id).Time("since", since).Stringer("stdType", stdType).Str("host", d.host.Name).Msg("Streaming logs for container")

	sinceQuery := since.Add(-50 * time.Millisecond).Format(time.RFC3339Nano)
	options := docker.LogsOptions{
		ShowStdout: stdType&container.STDOUT != 0,
		ShowStderr: stdType&container.STDERR != 0,
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

func (d *DockerClient) ContainerEvents(ctx context.Context, messages chan<- container.ContainerEvent) error {
	dockerMessages, err := d.cli.Events(ctx, events.ListOptions{})

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-err:
			return err

		case message := <-dockerMessages:
			if message.Type == events.ContainerEventType && len(message.Actor.ID) > 0 {
				messages <- container.ContainerEvent{
					ActorID:         message.Actor.ID[:12],
					Name:            string(message.Action),
					Host:            d.host.ID,
					ActorAttributes: message.Actor.Attributes,
					Time:            time.Now(),
				}
			}
		}
	}
}

func (d *DockerClient) ContainerLogsBetweenDates(ctx context.Context, id string, from time.Time, to time.Time, stdType container.StdType) (io.ReadCloser, error) {
	log.Debug().Str("id", id).Time("from", from).Time("to", to).Stringer("stdType", stdType).Str("host", d.host.Name).Msg("Fetching logs between dates for container")
	options := docker.LogsOptions{
		ShowStdout: stdType&container.STDOUT != 0,
		ShowStderr: stdType&container.STDERR != 0,
		Timestamps: true,
		Since:      from.Format(time.RFC3339Nano),
		Until:      to.Format(time.RFC3339Nano),
	}

	reader, err := d.cli.ContainerLogs(ctx, id, options)
	if err != nil {
		return nil, err
	}

	return reader, nil
}

func (d *DockerClient) Ping(ctx context.Context) error {
	_, err := d.cli.Ping(ctx)
	return err
}

func (d *DockerClient) Host() container.Host {
	log.Debug().Str("host", d.host.Name).Msg("Fetching host")
	return d.host
}

func (d *DockerClient) ContainerAttach(ctx context.Context, id string) (io.WriteCloser, io.Reader, error) {
	log.Debug().Str("id", id).Str("host", d.host.Name).Msg("Attaching to container")
	options := docker.AttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
		Stderr: true,
	}

	waiter, err := d.cli.ContainerAttach(ctx, id, options)

	if err != nil {
		return nil, nil, err
	}

	return waiter.Conn, waiter.Reader, nil
}

func (d *DockerClient) ContainerExec(ctx context.Context, id string, cmd []string) (io.WriteCloser, io.Reader, error) {
	log.Debug().Str("id", id).Str("host", d.host.Name).Msg("Executing command in container")
	options := docker.ExecOptions{
		AttachStdout: true,
		AttachStderr: true,
		AttachStdin:  true,
		Cmd:          cmd,
		Tty:          true,
	}

	execID, err := d.cli.ContainerExecCreate(ctx, id, options)
	if err != nil {
		return nil, nil, err
	}

	waiter, err := d.cli.ContainerExecAttach(ctx, execID.ID, docker.ExecAttachOptions{})
	if err != nil {
		return nil, nil, err
	}

	if err = d.cli.ContainerExecResize(ctx, execID.ID, docker.ResizeOptions{
		Width:  100,
		Height: 40,
	}); err != nil {
		return nil, nil, err
	}

	return waiter.Conn, waiter.Reader, nil
}

func newContainer(c docker.Summary, host string) container.Container {
	name := "no name"
	if c.Labels["dev.dozzle.name"] != "" {
		name = c.Labels["dev.dozzle.name"]
	} else if len(c.Names) > 0 {
		name = strings.TrimPrefix(c.Names[0], "/")
	}

	group := ""
	if c.Labels["dev.dozzle.group"] != "" {
		group = c.Labels["dev.dozzle.group"]
	}
	return container.Container{
		ID:      c.ID[:12],
		Name:    name,
		Image:   c.Image,
		Command: c.Command,
		Created: time.Unix(c.Created, 0),
		State:   c.State,
		Host:    host,
		Labels:  c.Labels,
		Stats:   utils.NewRingBuffer[container.ContainerStat](300), // 300 seconds of stats
		Group:   group,
	}
}

func newContainerFromJSON(c docker.InspectResponse, host string) container.Container {
	name := "no name"
	if c.Config.Labels["dev.dozzle.name"] != "" {
		name = c.Config.Labels["dev.dozzle.name"]
	} else if len(c.Name) > 0 {
		name = strings.TrimPrefix(c.Name, "/")
	}

	group := ""
	if c.Config.Labels["dev.dozzle.group"] != "" {
		group = c.Config.Labels["dev.dozzle.group"]
	}

	container := container.Container{
		ID:          c.ID[:12],
		Name:        name,
		Image:       c.Config.Image,
		Command:     strings.Join(c.Config.Entrypoint, " ") + " " + strings.Join(c.Config.Cmd, " "),
		State:       c.State.Status,
		Host:        host,
		Labels:      c.Config.Labels,
		Stats:       utils.NewRingBuffer[container.ContainerStat](300), // 300 seconds of stats
		Group:       group,
		Tty:         c.Config.Tty,
		MemoryLimit: uint64(c.HostConfig.Memory),
		CPULimit:    float64(c.HostConfig.NanoCPUs) / 1e9,
		FullyLoaded: true,
	}

	if createdAt, err := time.Parse(time.RFC3339Nano, c.Created); err == nil {
		container.Created = createdAt.UTC()
	}

	if startedAt, err := time.Parse(time.RFC3339Nano, c.State.StartedAt); err == nil {
		container.StartedAt = startedAt.UTC()
	}

	if stoppedAt, err := time.Parse(time.RFC3339Nano, c.State.FinishedAt); err == nil {
		container.FinishedAt = stoppedAt.UTC()
	}

	if c.State.Health != nil {
		container.Health = strings.ToLower(c.State.Health.Status)
	}

	return container
}
