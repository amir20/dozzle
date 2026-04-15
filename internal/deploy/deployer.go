package deploy

import (
	"context"
	"fmt"
	"io"
	"maps"
	"strconv"
	"time"

	composetypes "github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/go-connections/nat"
	units "github.com/docker/go-units"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/rs/zerolog/log"
)

// StatusUpdate reports progress during deployment.
type StatusUpdate struct {
	Service string
	Action  string
}

func sendStatus(ch chan<- StatusUpdate, service, action string) {
	if ch != nil {
		ch <- StatusUpdate{Service: service, Action: action}
	}
}

// DockerClient is the subset of the Docker API needed for deployment.
type DockerClient interface {
	ImagePull(ctx context.Context, refStr string, options image.PullOptions) (io.ReadCloser, error)
	ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *ocispec.Platform, containerName string) (container.CreateResponse, error)
	ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error
	ContainerList(ctx context.Context, options container.ListOptions) ([]container.Summary, error)
	ContainerStop(ctx context.Context, containerID string, options container.StopOptions) error
	ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error
	NetworkList(ctx context.Context, options network.ListOptions) ([]network.Summary, error)
	NetworkCreate(ctx context.Context, name string, options network.CreateOptions) (network.CreateResponse, error)
	NetworkConnect(ctx context.Context, networkID, containerID string, config *network.EndpointSettings) error
	VolumeCreate(ctx context.Context, options volume.CreateOptions) (volume.Volume, error)
}

// Deployer deploys a parsed compose project using the Docker API directly.
type Deployer struct {
	cli DockerClient
}

// NewDeployer creates a deployer that uses the given Docker client.
func NewDeployer(cli DockerClient) *Deployer {
	return &Deployer{cli: cli}
}

// Deploy removes existing project containers, ensures networks and volumes
// exist, then pulls images and creates+starts containers in dependency order.
// Status updates are sent to the optional status channel.
func (d *Deployer) Deploy(ctx context.Context, project *composetypes.Project, status chan<- StatusUpdate) error {
	if err := d.removeProjectContainers(ctx, project.Name, status); err != nil {
		return fmt.Errorf("removing existing containers: %w", err)
	}

	networkIDs, err := d.ensureNetworks(ctx, project)
	if err != nil {
		return fmt.Errorf("ensuring networks: %w", err)
	}

	if err := d.createVolumes(ctx, project); err != nil {
		return fmt.Errorf("creating volumes: %w", err)
	}

	return project.ForEachService(nil, func(name string, svc *composetypes.ServiceConfig) error {
		return d.deployService(ctx, project.Name, name, *svc, networkIDs, status)
	})
}

func (d *Deployer) removeProjectContainers(ctx context.Context, projectName string, status chan<- StatusUpdate) error {
	containers, err := d.cli.ContainerList(ctx, container.ListOptions{
		All: true,
		Filters: filters.NewArgs(
			filters.Arg("label", "com.docker.compose.project="+projectName),
		),
	})
	if err != nil {
		return fmt.Errorf("listing containers: %w", err)
	}

	for _, c := range containers {
		service := c.Labels["com.docker.compose.service"]

		sendStatus(status, service, "stopping")
		if err := d.cli.ContainerStop(ctx, c.ID, container.StopOptions{}); err != nil {
			log.Warn().Err(err).Str("container", c.ID[:12]).Msg("Failed to stop container")
		}

		sendStatus(status, service, "removing")
		if err := d.cli.ContainerRemove(ctx, c.ID, container.RemoveOptions{}); err != nil {
			return fmt.Errorf("removing container %s: %w", c.ID[:12], err)
		}
		log.Info().Str("service", service).Str("container", c.ID[:12]).Msg("Removed container")
	}
	return nil
}

func (d *Deployer) ensureNetworks(ctx context.Context, project *composetypes.Project) (map[string]string, error) {
	ids := make(map[string]string)

	existing, err := d.cli.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing networks: %w", err)
	}
	existingByName := make(map[string]string)
	for _, n := range existing {
		existingByName[n.Name] = n.ID
	}

	for name, netCfg := range project.Networks {
		if bool(netCfg.External) {
			extName := netCfg.Name
			if extName == "" {
				extName = name
			}
			if id, ok := existingByName[extName]; ok {
				ids[name] = id
			} else {
				return nil, fmt.Errorf("external network %q not found", extName)
			}
			continue
		}

		fullName := netCfg.Name
		if fullName == "" {
			fullName = project.Name + "_" + name
		}

		if id, ok := existingByName[fullName]; ok {
			ids[name] = id
			log.Info().Str("network", fullName).Msg("Network already exists, skipping")
			continue
		}

		driver := netCfg.Driver
		if driver == "" {
			driver = "bridge"
		}

		labels := make(map[string]string)
		maps.Copy(labels, netCfg.Labels)
		labels["com.docker.compose.project"] = project.Name
		labels["com.docker.compose.network"] = name

		resp, err := d.cli.NetworkCreate(ctx, fullName, network.CreateOptions{
			Driver:     driver,
			Internal:   netCfg.Internal,
			Attachable: netCfg.Attachable,
			Labels:     labels,
			Options:    netCfg.DriverOpts,
		})
		if err != nil {
			return nil, fmt.Errorf("creating network %q: %w", fullName, err)
		}
		ids[name] = resp.ID
		log.Info().Str("network", fullName).Str("id", resp.ID[:12]).Msg("Created network")
	}

	return ids, nil
}

func (d *Deployer) createVolumes(ctx context.Context, project *composetypes.Project) error {
	for name, volCfg := range project.Volumes {
		if bool(volCfg.External) {
			continue
		}

		fullName := volCfg.Name
		if fullName == "" {
			fullName = project.Name + "_" + name
		}
		driver := volCfg.Driver
		if driver == "" {
			driver = "local"
		}

		labels := make(map[string]string)
		maps.Copy(labels, volCfg.Labels)
		labels["com.docker.compose.project"] = project.Name
		labels["com.docker.compose.volume"] = name

		_, err := d.cli.VolumeCreate(ctx, volume.CreateOptions{
			Name:       fullName,
			Driver:     driver,
			DriverOpts: volCfg.DriverOpts,
			Labels:     labels,
		})
		if err != nil {
			return fmt.Errorf("creating volume %q: %w", fullName, err)
		}
		log.Info().Str("volume", fullName).Msg("Ensured volume")
	}
	return nil
}

func (d *Deployer) deployService(ctx context.Context, projectName, name string, svc composetypes.ServiceConfig, networkIDs map[string]string, status chan<- StatusUpdate) error {
	if svc.Image == "" {
		return fmt.Errorf("service %q has no image (build is not supported)", name)
	}

	sendStatus(status, name, "pulling")
	log.Info().Str("service", name).Str("image", svc.Image).Msg("Pulling image")
	reader, err := d.cli.ImagePull(ctx, svc.Image, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("pulling image %q: %w", svc.Image, err)
	}
	io.Copy(io.Discard, reader)
	reader.Close()

	config, hostConfig, networkingConfig := buildContainerConfig(projectName, name, svc, networkIDs)

	containerName := svc.ContainerName
	if containerName == "" {
		containerName = projectName + "-" + name + "-1"
	}

	sendStatus(status, name, "creating")
	resp, err := d.cli.ContainerCreate(ctx, config, hostConfig, networkingConfig, nil, containerName)
	if err != nil {
		return fmt.Errorf("creating container %q: %w", containerName, err)
	}
	log.Info().Str("service", name).Str("container", containerName).Str("id", resp.ID[:12]).Msg("Created container")

	// Connect to additional networks beyond the first.
	first := true
	for netName := range svc.Networks {
		if first {
			first = false
			continue
		}
		netID, ok := networkIDs[netName]
		if !ok {
			continue
		}
		var aliases []string
		if netCfg := svc.Networks[netName]; netCfg != nil {
			aliases = netCfg.Aliases
		}
		aliases = append(aliases, name)
		if err := d.cli.NetworkConnect(ctx, netID, resp.ID, &network.EndpointSettings{
			Aliases: aliases,
		}); err != nil {
			return fmt.Errorf("connecting container to network %q: %w", netName, err)
		}
	}

	sendStatus(status, name, "starting")
	if err := d.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("starting container %q: %w", containerName, err)
	}
	log.Info().Str("service", name).Str("container", containerName).Msg("Started container")

	return nil
}

func buildContainerConfig(projectName, name string, svc composetypes.ServiceConfig, networkIDs map[string]string) (*container.Config, *container.HostConfig, *network.NetworkingConfig) {
	// Environment
	env := make([]string, 0, len(svc.Environment))
	for k, v := range svc.Environment {
		if v != nil {
			env = append(env, k+"="+*v)
		} else {
			env = append(env, k)
		}
	}

	// Labels
	labels := make(map[string]string)
	maps.Copy(labels, svc.Labels)
	labels["com.docker.compose.project"] = projectName
	labels["com.docker.compose.service"] = name

	// Ports
	exposedPorts := nat.PortSet{}
	portBindings := nat.PortMap{}
	for _, p := range svc.Ports {
		natPort := nat.Port(strconv.FormatUint(uint64(p.Target), 10) + "/" + p.Protocol)
		exposedPorts[natPort] = struct{}{}
		portBindings[natPort] = append(portBindings[natPort], nat.PortBinding{
			HostIP:   p.HostIP,
			HostPort: p.Published,
		})
	}

	// Mounts
	mounts := make([]mount.Mount, 0, len(svc.Volumes))
	for _, v := range svc.Volumes {
		m := mount.Mount{
			Target:   v.Target,
			ReadOnly: v.ReadOnly,
		}
		switch v.Type {
		case "bind":
			m.Type = mount.TypeBind
			m.Source = v.Source
		case "volume":
			m.Type = mount.TypeVolume
			m.Source = v.Source
		case "tmpfs":
			m.Type = mount.TypeTmpfs
		default:
			m.Type = mount.TypeVolume
			m.Source = v.Source
		}
		mounts = append(mounts, m)
	}

	// Tmpfs
	tmpfs := make(map[string]string)
	for _, t := range svc.Tmpfs {
		tmpfs[t] = ""
	}

	// Restart policy
	restartPolicy := container.RestartPolicy{}
	switch svc.Restart {
	case "always":
		restartPolicy.Name = container.RestartPolicyAlways
	case "on-failure":
		restartPolicy.Name = container.RestartPolicyOnFailure
	case "unless-stopped":
		restartPolicy.Name = container.RestartPolicyUnlessStopped
	default:
		restartPolicy.Name = container.RestartPolicyDisabled
	}

	// Network mode
	networkMode := container.NetworkMode(svc.NetworkMode)

	// First network for NetworkingConfig
	var networkingConfig *network.NetworkingConfig
	if networkMode == "" && len(svc.Networks) > 0 {
		endpointsConfig := make(map[string]*network.EndpointSettings)
		for netName, netCfg := range svc.Networks {
			netID, ok := networkIDs[netName]
			if !ok {
				continue
			}
			var aliases []string
			if netCfg != nil {
				aliases = netCfg.Aliases
			}
			aliases = append(aliases, name)
			endpointsConfig[netID] = &network.EndpointSettings{
				Aliases: aliases,
			}
			break // only first network at create time
		}
		if len(endpointsConfig) > 0 {
			networkingConfig = &network.NetworkingConfig{EndpointsConfig: endpointsConfig}
		}
	}

	// Extra hosts
	extraHosts := svc.ExtraHosts.AsList(":")

	// Healthcheck
	var healthcheck *dockerspec.HealthcheckConfig
	if svc.HealthCheck != nil {
		if svc.HealthCheck.Disable {
			healthcheck = &dockerspec.HealthcheckConfig{
				Test: []string{"NONE"},
			}
		} else {
			healthcheck = &dockerspec.HealthcheckConfig{
				Test: []string(svc.HealthCheck.Test),
			}
			if svc.HealthCheck.Interval != nil {
				healthcheck.Interval = time.Duration(*svc.HealthCheck.Interval)
			}
			if svc.HealthCheck.Timeout != nil {
				healthcheck.Timeout = time.Duration(*svc.HealthCheck.Timeout)
			}
			if svc.HealthCheck.StartPeriod != nil {
				healthcheck.StartPeriod = time.Duration(*svc.HealthCheck.StartPeriod)
			}
			if svc.HealthCheck.StartInterval != nil {
				healthcheck.StartInterval = time.Duration(*svc.HealthCheck.StartInterval)
			}
			if svc.HealthCheck.Retries != nil {
				healthcheck.Retries = int(*svc.HealthCheck.Retries)
			}
		}
	}

	// Logging
	var logConfig container.LogConfig
	if svc.Logging != nil {
		logConfig.Type = svc.Logging.Driver
		logConfig.Config = make(map[string]string)
		for k, v := range svc.Logging.Options {
			logConfig.Config[k] = v
		}
	}

	// Devices
	devices := make([]container.DeviceMapping, 0, len(svc.Devices))
	for _, dev := range svc.Devices {
		devices = append(devices, container.DeviceMapping{
			PathOnHost:        dev.Source,
			PathInContainer:   dev.Target,
			CgroupPermissions: dev.Permissions,
		})
	}

	// Ulimits
	var ulimits []*units.Ulimit
	for name, ul := range svc.Ulimits {
		if ul.Single != 0 {
			ulimits = append(ulimits, &units.Ulimit{
				Name: name,
				Hard: int64(ul.Single),
				Soft: int64(ul.Single),
			})
		} else {
			ulimits = append(ulimits, &units.Ulimit{
				Name: name,
				Soft: int64(ul.Soft),
				Hard: int64(ul.Hard),
			})
		}
	}

	config := &container.Config{
		Image:        svc.Image,
		Env:          env,
		Labels:       labels,
		ExposedPorts: exposedPorts,
		Hostname:     svc.Hostname,
		WorkingDir:   svc.WorkingDir,
		User:         svc.User,
		Tty:          svc.Tty,
		StdinOnce:    svc.StdinOpen,
		StopSignal:   svc.StopSignal,
		Healthcheck:  healthcheck,
	}
	if svc.StopGracePeriod != nil {
		timeout := int(time.Duration(*svc.StopGracePeriod).Seconds())
		config.StopTimeout = &timeout
	}
	if len(svc.Command) > 0 {
		config.Cmd = []string(svc.Command)
	}
	if len(svc.Entrypoint) > 0 {
		config.Entrypoint = []string(svc.Entrypoint)
	}

	hostConfig := &container.HostConfig{
		PortBindings:   portBindings,
		RestartPolicy:  restartPolicy,
		Mounts:         mounts,
		Privileged:     svc.Privileged,
		ReadonlyRootfs: svc.ReadOnly,
		ExtraHosts:     extraHosts,
		DNS:            svc.DNS,
		CapAdd:         svc.CapAdd,
		CapDrop:        svc.CapDrop,
		NetworkMode:    networkMode,
		PidMode:        container.PidMode(svc.Pid),
		IpcMode:        container.IpcMode(svc.Ipc),
		Tmpfs:          tmpfs,
		ShmSize:        int64(svc.ShmSize),
		Init:           svc.Init,
		SecurityOpt:    svc.SecurityOpt,
		Sysctls:        svc.Sysctls,
		LogConfig:      logConfig,
		Runtime:        svc.Runtime,
		Resources: container.Resources{
			NanoCPUs: int64(svc.CPUS * 1e9),
			Memory:   int64(svc.MemLimit),
			Ulimits:  ulimits,
			Devices:  devices,
		},
	}

	return config, hostConfig, networkingConfig
}
