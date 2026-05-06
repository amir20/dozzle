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
	docker "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/events"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/api/types/system"
	"github.com/moby/moby/client"

	"github.com/rs/zerolog/log"
)

type DockerCLI interface {
	ContainerList(context.Context, client.ContainerListOptions) (client.ContainerListResult, error)
	ContainerLogs(context.Context, string, client.ContainerLogsOptions) (client.ContainerLogsResult, error)
	Events(context.Context, client.EventsListOptions) client.EventsResult
	ContainerInspect(ctx context.Context, containerID string, options client.ContainerInspectOptions) (client.ContainerInspectResult, error)
	ContainerStats(ctx context.Context, containerID string, options client.ContainerStatsOptions) (client.ContainerStatsResult, error)
	Ping(ctx context.Context, options client.PingOptions) (client.PingResult, error)
	ContainerStart(ctx context.Context, containerID string, options client.ContainerStartOptions) (client.ContainerStartResult, error)
	ContainerStop(ctx context.Context, containerID string, options client.ContainerStopOptions) (client.ContainerStopResult, error)
	ContainerRestart(ctx context.Context, containerID string, options client.ContainerRestartOptions) (client.ContainerRestartResult, error)
	ContainerAttach(ctx context.Context, containerID string, options client.ContainerAttachOptions) (client.ContainerAttachResult, error)
	ExecCreate(ctx context.Context, containerID string, options client.ExecCreateOptions) (client.ExecCreateResult, error)
	ExecAttach(ctx context.Context, execID string, config client.ExecAttachOptions) (client.ExecAttachResult, error)
	ExecResize(ctx context.Context, execID string, options client.ExecResizeOptions) (client.ExecResizeResult, error)
	Info(ctx context.Context, options client.InfoOptions) (client.SystemInfoResult, error)
	ImagePull(ctx context.Context, refStr string, options client.ImagePullOptions) (client.ImagePullResponse, error)
	ContainerRemove(ctx context.Context, containerID string, options client.ContainerRemoveOptions) (client.ContainerRemoveResult, error)
	ContainerCreate(ctx context.Context, options client.ContainerCreateOptions) (client.ContainerCreateResult, error)
	ServiceInspect(ctx context.Context, serviceID string, opts client.ServiceInspectOptions) (client.ServiceInspectResult, error)
	ServiceUpdate(ctx context.Context, serviceID string, options client.ServiceUpdateOptions) (client.ServiceUpdateResult, error)
}

type DockerClient struct {
	cli  DockerCLI
	host container.Host
	info system.Info
}

func NewClient(cli DockerCLI, host container.Host) *DockerClient {
	infoResult, err := cli.Info(context.Background(), client.InfoOptions{})
	if err != nil {
		log.Error().Err(err).Msg("Failed to get docker info")
	}
	info := infoResult.Info

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
	cli, err := client.New(client.FromEnv, client.WithUserAgent("Docker-Client/Dozzle"))

	if err != nil {
		return nil, err
	}

	infoResult, err := cli.Info(context.Background(), client.InfoOptions{})
	if err != nil {
		return nil, err
	}
	info := infoResult.Info

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

	opts = append(opts, client.WithUserAgent("Docker-Client/Dozzle"))

	cli, err := client.New(opts...)

	if err != nil {
		return nil, err
	}

	host.Type = "remote"

	return NewClient(cli, host), nil
}

// Finds a container by id, skipping the filters
func (d *DockerClient) FindContainer(ctx context.Context, id string) (container.Container, error) {
	log.Debug().Str("id", id).Msg("Finding container")
	if result, err := d.cli.ContainerInspect(ctx, id, client.ContainerInspectOptions{}); err == nil {
		return newContainerFromJSON(result.Container, d.host.ID), nil
	} else {
		return container.Container{}, err
	}

}

func (d *DockerClient) ContainerActions(ctx context.Context, action container.ContainerAction, containerID string) error {
	switch action {
	case container.Start:
		_, err := d.cli.ContainerStart(ctx, containerID, client.ContainerStartOptions{})
		return err
	case container.Stop:
		_, err := d.cli.ContainerStop(ctx, containerID, client.ContainerStopOptions{})
		return err
	case container.Restart:
		_, err := d.cli.ContainerRestart(ctx, containerID, client.ContainerRestartOptions{})
		return err
	case container.Remove:
		_, err := d.cli.ContainerRemove(ctx, containerID, client.ContainerRemoveOptions{})
		return err
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}

func (d *DockerClient) ImagePull(ctx context.Context, imageName string) (io.ReadCloser, error) {
	return d.cli.ImagePull(ctx, imageName, client.ImagePullOptions{})
}

func (d *DockerClient) ContainerInspect(ctx context.Context, containerID string) (docker.InspectResponse, error) {
	result, err := d.cli.ContainerInspect(ctx, containerID, client.ContainerInspectOptions{})
	return result.Container, err
}

func (d *DockerClient) ContainerRemove(ctx context.Context, containerID string) error {
	_, err := d.cli.ContainerRemove(ctx, containerID, client.ContainerRemoveOptions{})
	return err
}

func (d *DockerClient) ContainerCreate(ctx context.Context, inspectResp docker.InspectResponse, name string) (string, error) {
	// Clear hostname when using network modes that don't support it (host, container:*)
	// Docker always populates Hostname in inspect responses, but rejects it on create
	// for these network modes.
	if inspectResp.HostConfig != nil {
		mode := string(inspectResp.HostConfig.NetworkMode)
		if mode == "host" || strings.HasPrefix(mode, "container:") {
			inspectResp.Config.Hostname = ""
		}
	}

	// Build clean EndpointsConfig with only network names and aliases,
	// stripping runtime state (IPs, gateways, MAC addresses) that can
	// cause conflicts when recreating.
	var networkingConfig *network.NetworkingConfig
	if inspectResp.NetworkSettings != nil && len(inspectResp.NetworkSettings.Networks) > 0 {
		endpointsConfig := make(map[string]*network.EndpointSettings, len(inspectResp.NetworkSettings.Networks))
		for netName, ep := range inspectResp.NetworkSettings.Networks {
			endpointsConfig[netName] = &network.EndpointSettings{
				Aliases: ep.Aliases,
			}
		}
		networkingConfig = &network.NetworkingConfig{EndpointsConfig: endpointsConfig}
	}

	resp, err := d.cli.ContainerCreate(ctx, client.ContainerCreateOptions{
		Config:           inspectResp.Config,
		HostConfig:       inspectResp.HostConfig,
		NetworkingConfig: networkingConfig,
		Platform:         nil,
		Name:             name,
	})
	if err != nil {
		return "", err
	}
	return resp.ID, nil
}

func (d *DockerClient) ServiceUpdate(ctx context.Context, serviceID string, imageName string) error {
	inspectResult, err := d.cli.ServiceInspect(ctx, serviceID, client.ServiceInspectOptions{})
	if err != nil {
		return err
	}
	svc := inspectResult.Service
	svc.Spec.TaskTemplate.ContainerSpec.Image = imageName
	svc.Spec.TaskTemplate.ForceUpdate++
	_, err = d.cli.ServiceUpdate(ctx, serviceID, client.ServiceUpdateOptions{Version: svc.Version, Spec: svc.Spec})
	return err
}

func (d *DockerClient) ListContainers(ctx context.Context, labels container.ContainerLabels) ([]container.Container, error) {
	log.Debug().Interface("labels", labels).Str("host", d.host.Name).Msg("Listing containers")
	filterArgs := make(client.Filters)
	for key, values := range labels {
		filterArgs.Add(key, values...)
	}
	containerListOptions := client.ContainerListOptions{
		Filters: filterArgs,
		All:     true,
	}
	list, err := d.cli.ContainerList(ctx, containerListOptions)
	if err != nil {
		return nil, err
	}

	var containers = make([]container.Container, 0, len(list.Items))
	for _, c := range list.Items {
		containers = append(containers, newContainer(c, d.host.ID))
	}

	sort.Slice(containers, func(i, j int) bool {
		return strings.ToLower(containers[i].Name) < strings.ToLower(containers[j].Name)
	})

	return containers, nil
}

func (d *DockerClient) ContainerStats(ctx context.Context, id string, stats chan<- container.ContainerStat) error {
	response, err := d.cli.ContainerStats(ctx, id, client.ContainerStatsOptions{Stream: true})

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
			networkRx, networkTx   uint64
		)
		daemonOSType := v.OSType

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

		// Calculate total network bytes across all interfaces
		for _, netStats := range v.Networks {
			networkRx += netStats.RxBytes
			networkTx += netStats.TxBytes
		}

		select {
		case <-ctx.Done():
			return nil
		case stats <- container.ContainerStat{
			ID:             id,
			CPUPercent:     cpuPercent,
			MemoryPercent:  memPercent,
			MemoryUsage:    mem,
			NetworkRxTotal: networkRx,
			NetworkTxTotal: networkTx,
		}:
		}
	}
}

func (d *DockerClient) ContainerLogs(ctx context.Context, id string, since time.Time, stdType container.StdType) (io.ReadCloser, error) {
	log.Debug().Str("id", id).Time("since", since).Stringer("stdType", stdType).Str("host", d.host.Name).Msg("Streaming logs for container")

	sinceQuery := since.Add(-50 * time.Millisecond).Format(time.RFC3339Nano)
	options := client.ContainerLogsOptions{
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
	eventsResult := d.cli.Events(ctx, client.EventsListOptions{})
	dockerMessages := eventsResult.Messages
	err := eventsResult.Err

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
	options := client.ContainerLogsOptions{
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
	_, err := d.cli.Ping(ctx, client.PingOptions{})
	return err
}

func (d *DockerClient) Host() container.Host {
	return d.host
}

// RawClient returns the underlying *client.Client if the DockerCLI is one.
// Needed for operations like network/volume management that aren't part of
// the DockerCLI interface.
func (d *DockerClient) RawClient() *client.Client {
	if c, ok := d.cli.(*client.Client); ok {
		return c
	}
	return nil
}

func (d *DockerClient) ContainerAttach(ctx context.Context, id string) (*container.ExecSession, error) {
	log.Debug().Str("id", id).Str("host", d.host.Name).Msg("Attaching to container")
	options := client.ContainerAttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
		Stderr: true,
	}

	waiter, err := d.cli.ContainerAttach(ctx, id, options)

	if err != nil {
		return nil, err
	}

	// Docker attach doesn't support resize - it's not an exec session
	// Return a no-op resize function
	resizeFn := func(width uint, height uint) error {
		log.Debug().Uint("width", width).Uint("height", height).Msg("resize not supported for attach")
		return nil
	}

	return &container.ExecSession{
		Writer: waiter.Conn,
		Reader: waiter.Reader,
		Resize: resizeFn,
	}, nil
}

func (d *DockerClient) ContainerExec(ctx context.Context, id string, cmd []string) (*container.ExecSession, error) {
	log.Debug().Str("id", id).Str("host", d.host.Name).Msg("Executing command in container")
	options := client.ExecCreateOptions{
		AttachStdout: true,
		AttachStderr: true,
		AttachStdin:  true,
		Cmd:          cmd,
		TTY:          true,
	}

	execID, err := d.cli.ExecCreate(ctx, id, options)
	if err != nil {
		return nil, err
	}

	waiter, err := d.cli.ExecAttach(ctx, execID.ID, client.ExecAttachOptions{TTY: true})
	if err != nil {
		return nil, err
	}

	// Initial resize
	if _, err = d.cli.ExecResize(ctx, execID.ID, client.ExecResizeOptions{
		Width:  100,
		Height: 40,
	}); err != nil {
		return nil, err
	}

	// Create resize closure that captures execID and context
	resizeFn := func(width uint, height uint) error {
		_, err := d.cli.ExecResize(ctx, execID.ID, client.ExecResizeOptions{
			Width:  width,
			Height: height,
		})
		return err
	}

	return &container.ExecSession{
		Writer: waiter.Conn,
		Reader: waiter.Reader,
		Resize: resizeFn,
	}, nil
}

func newContainer(c docker.Summary, host string) container.Container {
	name := "no name"
	if c.Labels["dev.dozzle.name"] != "" {
		name = c.Labels["dev.dozzle.name"]
	} else if c.Labels["coolify.serviceName"] != "" {
		name = c.Labels["coolify.serviceName"]
	} else if len(c.Names) > 0 {
		name = strings.TrimPrefix(c.Names[0], "/")
	}

	group := ""
	if c.Labels["dev.dozzle.group"] != "" {
		group = c.Labels["dev.dozzle.group"]
	} else if c.Labels["coolify.projectName"] != "" {
		group = c.Labels["coolify.projectName"]
	}
	return container.Container{
		ID:      c.ID[:12],
		Name:    name,
		Image:   c.Image,
		Command: c.Command,
		Created: time.Unix(c.Created, 0),
		State:   string(c.State),
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
	} else if c.Config.Labels["coolify.serviceName"] != "" {
		name = c.Config.Labels["coolify.serviceName"]
	} else if len(c.Name) > 0 {
		name = strings.TrimPrefix(c.Name, "/")
	}

	group := ""
	if c.Config.Labels["dev.dozzle.group"] != "" {
		group = c.Config.Labels["dev.dozzle.group"]
	} else if c.Config.Labels["coolify.projectName"] != "" {
		group = c.Config.Labels["coolify.projectName"]
	}

	// Format port bindings as readable strings
	var ports []string
	for port, bindings := range c.HostConfig.PortBindings {
		for _, b := range bindings {
			if b.HostPort != "" {
				ports = append(ports, fmt.Sprintf("%s:%s->%s", b.HostIP, b.HostPort, port))
			} else {
				ports = append(ports, port.String())
			}
		}
	}

	// Format mounts as readable strings
	var mounts []string
	for _, m := range c.Mounts {
		mounts = append(mounts, fmt.Sprintf("%s:%s (%s)", m.Source, m.Destination, m.Type))
	}

	restartPolicy := ""
	if c.HostConfig.RestartPolicy.Name != "" {
		restartPolicy = string(c.HostConfig.RestartPolicy.Name)
	}

	container := container.Container{
		ID:            c.ID[:12],
		Name:          name,
		Image:         c.Config.Image,
		Command:       strings.Join(c.Config.Entrypoint, " ") + " " + strings.Join(c.Config.Cmd, " "),
		State:         string(c.State.Status),
		Host:          host,
		Labels:        c.Config.Labels,
		Stats:         utils.NewRingBuffer[container.ContainerStat](300), // 300 seconds of stats
		Group:         group,
		Tty:           c.Config.Tty,
		MemoryLimit:   uint64(c.HostConfig.Memory),
		CPULimit:      float64(c.HostConfig.NanoCPUs) / 1e9,
		Env:           c.Config.Env,
		Ports:         ports,
		Mounts:        mounts,
		RestartPolicy: restartPolicy,
		NetworkMode:   string(c.HostConfig.NetworkMode),
		FullyLoaded:   true,
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
		container.Health = strings.ToLower(string(c.State.Health.Status))
	}

	return container
}
