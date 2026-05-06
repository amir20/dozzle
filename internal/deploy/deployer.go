package deploy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/netip"
	"sort"
	"strconv"
	"time"

	composetypes "github.com/compose-spec/compose-go/v2/types"
	units "github.com/docker/go-units"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

// Compose resource labels. Used to tag every network/volume/container we
// create so we can filter project-owned resources on teardown.
const (
	labelComposeProject = "com.docker.compose.project"
	labelComposeService = "com.docker.compose.service"
	labelComposeNetwork = "com.docker.compose.network"
	labelComposeVolume  = "com.docker.compose.volume"
)

// StatusUpdate reports progress during deployment.
type StatusUpdate struct {
	Service string
	Action  string
}

func sendStatus(ctx context.Context, ch chan<- StatusUpdate, service, action string) {
	if ch == nil {
		return
	}
	select {
	case ch <- StatusUpdate{Service: service, Action: action}:
	case <-ctx.Done():
	}
}

func shortID(id string) string {
	if len(id) < 12 {
		return id
	}
	return id[:12]
}

func drainPullStream(r io.Reader) error {
	dec := json.NewDecoder(r)
	for {
		var msg struct {
			Error       string `json:"error"`
			ErrorDetail struct {
				Message string `json:"message"`
			} `json:"errorDetail"`
		}
		if err := dec.Decode(&msg); err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("decoding pull stream: %w", err)
		}
		if msg.ErrorDetail.Message != "" {
			return errors.New(msg.ErrorDetail.Message)
		}
		if msg.Error != "" {
			return errors.New(msg.Error)
		}
	}
}

// DockerClient is the subset of the Docker API needed for deployment.
type DockerClient interface {
	ImagePull(ctx context.Context, refStr string, options client.ImagePullOptions) (client.ImagePullResponse, error)
	ContainerCreate(ctx context.Context, options client.ContainerCreateOptions) (client.ContainerCreateResult, error)
	ContainerStart(ctx context.Context, containerID string, options client.ContainerStartOptions) (client.ContainerStartResult, error)
	ContainerList(ctx context.Context, options client.ContainerListOptions) (client.ContainerListResult, error)
	ContainerStop(ctx context.Context, containerID string, options client.ContainerStopOptions) (client.ContainerStopResult, error)
	ContainerRemove(ctx context.Context, containerID string, options client.ContainerRemoveOptions) (client.ContainerRemoveResult, error)
	NetworkList(ctx context.Context, options client.NetworkListOptions) (client.NetworkListResult, error)
	NetworkCreate(ctx context.Context, name string, options client.NetworkCreateOptions) (client.NetworkCreateResult, error)
	NetworkConnect(ctx context.Context, networkID string, options client.NetworkConnectOptions) (client.NetworkConnectResult, error)
	NetworkRemove(ctx context.Context, networkID string, options client.NetworkRemoveOptions) (client.NetworkRemoveResult, error)
	VolumeCreate(ctx context.Context, options client.VolumeCreateOptions) (client.VolumeCreateResult, error)
	VolumeList(ctx context.Context, options client.VolumeListOptions) (client.VolumeListResult, error)
	VolumeRemove(ctx context.Context, volumeID string, options client.VolumeRemoveOptions) (client.VolumeRemoveResult, error)
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

	if err := d.pullImages(ctx, project, status); err != nil {
		return err
	}

	return project.ForEachService(nil, func(name string, svc *composetypes.ServiceConfig) error {
		return d.deployService(ctx, project.Name, name, *svc, networkIDs, status)
	})
}

// pullImages pulls every service image concurrently (bounded). Pulls are
// independent, so fanning them out collapses wall time on multi-service
// projects where the ordered create phase is gated on the slowest image.
func (d *Deployer) pullImages(ctx context.Context, project *composetypes.Project, status chan<- StatusUpdate) error {
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(5)
	for svcName, svc := range project.Services {
		if svc.Image == "" {
			continue
		}
		imageRef := svc.Image
		g.Go(func() error {
			sendStatus(gctx, status, svcName, "pulling")
			log.Info().Str("service", svcName).Str("image", imageRef).Msg("Pulling image")
			reader, err := d.cli.ImagePull(gctx, imageRef, client.ImagePullOptions{})
			if err != nil {
				return fmt.Errorf("pulling image %q: %w", imageRef, err)
			}
			pullErr := drainPullStream(reader)
			reader.Close()
			if pullErr != nil {
				return fmt.Errorf("pulling image %q: %w", imageRef, pullErr)
			}
			return nil
		})
	}
	return g.Wait()
}

// Remove stops and removes all containers and project-labeled networks for
// the given project. Volumes are preserved unless removeVolumes is true —
// volumes typically hold user data, so they opt in explicitly.
func (d *Deployer) Remove(ctx context.Context, projectName string, removeVolumes bool, status chan<- StatusUpdate) error {
	if err := d.removeProjectContainers(ctx, projectName, status); err != nil {
		return fmt.Errorf("removing containers: %w", err)
	}
	if err := d.removeProjectNetworks(ctx, projectName); err != nil {
		return fmt.Errorf("removing networks: %w", err)
	}
	if removeVolumes {
		if err := d.removeProjectVolumes(ctx, projectName); err != nil {
			return fmt.Errorf("removing volumes: %w", err)
		}
	}
	return nil
}

func (d *Deployer) removeProjectNetworks(ctx context.Context, projectName string) error {
	networks, err := d.cli.NetworkList(ctx, client.NetworkListOptions{
		Filters: make(client.Filters).Add("label", labelComposeProject+"="+projectName),
	})
	if err != nil {
		return fmt.Errorf("listing networks: %w", err)
	}
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(5)
	for _, n := range networks.Items {
		g.Go(func() error {
			if _, err := d.cli.NetworkRemove(gctx, n.ID, client.NetworkRemoveOptions{}); err != nil {
				log.Warn().Err(err).Str("network", n.Name).Msg("Failed to remove network")
				return err
			}
			log.Info().Str("network", n.Name).Msg("Removed network")
			return nil
		})
	}
	return g.Wait()
}

func (d *Deployer) removeProjectVolumes(ctx context.Context, projectName string) error {
	vols, err := d.cli.VolumeList(ctx, client.VolumeListOptions{
		Filters: make(client.Filters).Add("label", labelComposeProject+"="+projectName),
	})
	if err != nil {
		return fmt.Errorf("listing volumes: %w", err)
	}
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(5)
	for _, v := range vols.Items {
		g.Go(func() error {
			if _, err := d.cli.VolumeRemove(gctx, v.Name, client.VolumeRemoveOptions{Force: false}); err != nil {
				log.Warn().Err(err).Str("volume", v.Name).Msg("Failed to remove volume")
				return err
			}
			log.Info().Str("volume", v.Name).Msg("Removed volume")
			return nil
		})
	}
	return g.Wait()
}

func (d *Deployer) removeProjectContainers(ctx context.Context, projectName string, status chan<- StatusUpdate) error {
	containers, err := d.cli.ContainerList(ctx, client.ContainerListOptions{
		All:     true,
		Filters: make(client.Filters).Add("label", labelComposeProject+"="+projectName),
	})
	if err != nil {
		return fmt.Errorf("listing containers: %w", err)
	}

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(5)
	for _, c := range containers.Items {
		service := c.Labels[labelComposeService]
		g.Go(func() error {
			sendStatus(gctx, status, service, "removing")
			if _, err := d.cli.ContainerRemove(gctx, c.ID, client.ContainerRemoveOptions{Force: true}); err != nil {
				return fmt.Errorf("removing container %s: %w", shortID(c.ID), err)
			}
			log.Info().Str("service", service).Str("container", shortID(c.ID)).Msg("Removed container")
			return nil
		})
	}
	return g.Wait()
}

func (d *Deployer) ensureNetworks(ctx context.Context, project *composetypes.Project) (map[string]string, error) {
	ids := make(map[string]string)

	// Two listings: project-owned (for idempotent reuse) and external references.
	// Project filter keeps the response small on hosts with many stacks.
	ownedList, err := d.cli.NetworkList(ctx, client.NetworkListOptions{
		Filters: make(client.Filters).Add("label", labelComposeProject+"="+project.Name),
	})
	if err != nil {
		return nil, fmt.Errorf("listing networks: %w", err)
	}
	ownedByName := make(map[string]string, len(ownedList.Items))
	for _, n := range ownedList.Items {
		ownedByName[n.Name] = n.ID
	}

	// External networks are looked up unfiltered, on demand (they are not project-labeled).
	var externalByName map[string]string
	lookupExternalNet := func() error {
		if externalByName != nil {
			return nil
		}
		all, err := d.cli.NetworkList(ctx, client.NetworkListOptions{})
		if err != nil {
			return fmt.Errorf("listing external networks: %w", err)
		}
		externalByName = make(map[string]string, len(all.Items))
		for _, n := range all.Items {
			externalByName[n.Name] = n.ID
		}
		return nil
	}

	for name, netCfg := range project.Networks {
		if bool(netCfg.External) {
			extName := netCfg.Name
			if extName == "" {
				extName = name
			}
			if err := lookupExternalNet(); err != nil {
				return nil, err
			}
			id, ok := externalByName[extName]
			if !ok {
				return nil, fmt.Errorf("external network %q not found", extName)
			}
			ids[name] = id
			continue
		}

		fullName := netCfg.Name
		if fullName == "" {
			fullName = project.Name + "_" + name
		}

		if id, ok := ownedByName[fullName]; ok {
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
		labels[labelComposeProject] = project.Name
		labels[labelComposeNetwork] = name

		resp, err := d.cli.NetworkCreate(ctx, fullName, client.NetworkCreateOptions{
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
		log.Info().Str("network", fullName).Str("id", shortID(resp.ID)).Msg("Created network")
	}

	return ids, nil
}

func (d *Deployer) createVolumes(ctx context.Context, project *composetypes.Project) error {
	// Project-filtered listing keeps the response small on hosts with many stacks.
	ownedList, err := d.cli.VolumeList(ctx, client.VolumeListOptions{
		Filters: make(client.Filters).Add("label", labelComposeProject+"="+project.Name),
	})
	if err != nil {
		return fmt.Errorf("listing volumes: %w", err)
	}
	ownedByName := make(map[string]struct{}, len(ownedList.Items))
	for _, v := range ownedList.Items {
		ownedByName[v.Name] = struct{}{}
	}

	// External volumes are looked up unfiltered, on demand (they are not project-labeled).
	var externalByName map[string]struct{}
	lookupExternal := func() error {
		if externalByName != nil {
			return nil
		}
		all, err := d.cli.VolumeList(ctx, client.VolumeListOptions{})
		if err != nil {
			return fmt.Errorf("listing external volumes: %w", err)
		}
		externalByName = make(map[string]struct{}, len(all.Items))
		for _, v := range all.Items {
			externalByName[v.Name] = struct{}{}
		}
		return nil
	}

	for name, volCfg := range project.Volumes {
		if bool(volCfg.External) {
			extName := volCfg.Name
			if extName == "" {
				extName = name
			}
			if err := lookupExternal(); err != nil {
				return err
			}
			if _, ok := externalByName[extName]; !ok {
				return fmt.Errorf("external volume %q not found", extName)
			}
			continue
		}

		fullName := volCfg.Name
		if fullName == "" {
			fullName = project.Name + "_" + name
		}

		if _, ok := ownedByName[fullName]; ok {
			log.Info().Str("volume", fullName).Msg("Volume already exists, skipping")
			continue
		}

		driver := volCfg.Driver
		if driver == "" {
			driver = "local"
		}

		labels := make(map[string]string)
		maps.Copy(labels, volCfg.Labels)
		labels[labelComposeProject] = project.Name
		labels[labelComposeVolume] = name

		if _, err := d.cli.VolumeCreate(ctx, client.VolumeCreateOptions{
			Name:       fullName,
			Driver:     driver,
			DriverOpts: volCfg.DriverOpts,
			Labels:     labels,
		}); err != nil {
			return fmt.Errorf("creating volume %q: %w", fullName, err)
		}
		log.Info().Str("volume", fullName).Msg("Created volume")
	}
	return nil
}

func (d *Deployer) deployService(ctx context.Context, projectName, name string, svc composetypes.ServiceConfig, networkIDs map[string]string, status chan<- StatusUpdate) error {
	if svc.Image == "" {
		return fmt.Errorf("service %q has no image (build is not supported)", name)
	}

	// Pick the primary network (attached at container-create time) deterministically.
	// sortedNetworks is alphabetical so create-time and later NetworkConnect agree.
	sortedNetworks := sortedNetworkNames(svc.Networks)
	primaryNetName := ""
	for _, n := range sortedNetworks {
		if _, ok := networkIDs[n]; ok {
			primaryNetName = n
			break
		}
	}

	config, hostConfig, networkingConfig := buildContainerConfig(projectName, name, svc, networkIDs, primaryNetName)

	containerName := svc.ContainerName
	if containerName == "" {
		containerName = projectName + "-" + name + "-1"
	}

	sendStatus(ctx, status, name, "creating")
	resp, err := d.cli.ContainerCreate(ctx, client.ContainerCreateOptions{
		Config:           config,
		HostConfig:       hostConfig,
		NetworkingConfig: networkingConfig,
		Name:             containerName,
	})
	if err != nil {
		return fmt.Errorf("creating container %q: %w", containerName, err)
	}
	log.Info().Str("service", name).Str("container", containerName).Str("id", shortID(resp.ID)).Msg("Created container")

	for _, netName := range sortedNetworks {
		if netName == primaryNetName {
			continue
		}
		netID, ok := networkIDs[netName]
		if !ok {
			continue
		}
		netCfg := svc.Networks[netName]
		aliases := copyAliases(netCfg, name)
		if _, err := d.cli.NetworkConnect(ctx, netID, client.NetworkConnectOptions{Container: resp.ID, EndpointConfig: &network.EndpointSettings{
			Aliases: aliases,
		}}); err != nil {
			// Partial attachment would leave an orphan. Remove and bail.
			if _, rmErr := d.cli.ContainerRemove(ctx, resp.ID, client.ContainerRemoveOptions{Force: true}); rmErr != nil {
				log.Warn().Err(rmErr).Str("container", shortID(resp.ID)).Msg("Failed to clean up after NetworkConnect error")
			}
			return fmt.Errorf("connecting container to network %q: %w", netName, err)
		}
	}

	sendStatus(ctx, status, name, "starting")
	if _, err := d.cli.ContainerStart(ctx, resp.ID, client.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("starting container %q: %w", containerName, err)
	}
	log.Info().Str("service", name).Str("container", containerName).Msg("Started container")

	return nil
}

// copyAliases returns a fresh aliases slice with the service name appended.
// Important: compose-go's ServiceNetworkConfig.Aliases is shared across
// deploys — appending directly would mutate the project model if cap allows.
func copyAliases(netCfg *composetypes.ServiceNetworkConfig, serviceName string) []string {
	var src []string
	if netCfg != nil {
		src = netCfg.Aliases
	}
	out := make([]string, 0, len(src)+1)
	out = append(out, src...)
	out = append(out, serviceName)
	return out
}

func sortedNetworkNames(nets map[string]*composetypes.ServiceNetworkConfig) []string {
	names := make([]string, 0, len(nets))
	for k := range nets {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

func buildContainerConfig(projectName, name string, svc composetypes.ServiceConfig, networkIDs map[string]string, primaryNetName string) (*container.Config, *container.HostConfig, *network.NetworkingConfig) {
	env := make([]string, 0, len(svc.Environment))
	for k, v := range svc.Environment {
		if v != nil {
			env = append(env, k+"="+*v)
		} else {
			env = append(env, k)
		}
	}

	labels := make(map[string]string)
	maps.Copy(labels, svc.Labels)
	labels[labelComposeProject] = projectName
	labels[labelComposeService] = name

	exposedPorts := network.PortSet{}
	portBindings := network.PortMap{}
	for _, p := range svc.Ports {
		proto := p.Protocol
		if proto == "" {
			proto = "tcp"
		}
		port, err := network.ParsePort(strconv.FormatUint(uint64(p.Target), 10) + "/" + proto)
		if err != nil {
			log.Warn().Err(err).Uint32("port", p.Target).Str("protocol", proto).Msg("Skipping invalid port")
			continue
		}
		exposedPorts[port] = struct{}{}
		var hostIP netip.Addr
		if p.HostIP != "" {
			var err error
			hostIP, err = netip.ParseAddr(p.HostIP)
			if err != nil {
				log.Warn().Err(err).Str("hostIP", p.HostIP).Msg("Failed to parse host IP, using zero value")
			}
		}
		portBindings[port] = append(portBindings[port], network.PortBinding{
			HostIP:   hostIP,
			HostPort: p.Published,
		})
	}

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

	tmpfs := make(map[string]string)
	for _, t := range svc.Tmpfs {
		tmpfs[t] = ""
	}

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

	networkMode := container.NetworkMode(svc.NetworkMode)

	var networkingConfig *network.NetworkingConfig
	if networkMode == "" && primaryNetName != "" {
		if netID, ok := networkIDs[primaryNetName]; ok {
			netCfg := svc.Networks[primaryNetName]
			aliases := copyAliases(netCfg, name)
			networkingConfig = &network.NetworkingConfig{
				EndpointsConfig: map[string]*network.EndpointSettings{
					netID: {Aliases: aliases},
				},
			}
		}
	}

	extraHosts := svc.ExtraHosts.AsList(":")

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

	var logConfig container.LogConfig
	if svc.Logging != nil && svc.Logging.Driver != "" {
		logConfig.Type = svc.Logging.Driver
		logConfig.Config = maps.Clone(svc.Logging.Options)
	}

	devices := make([]container.DeviceMapping, 0, len(svc.Devices))
	for _, dev := range svc.Devices {
		devices = append(devices, container.DeviceMapping{
			PathOnHost:        dev.Source,
			PathInContainer:   dev.Target,
			CgroupPermissions: dev.Permissions,
		})
	}

	ulimits := make([]*units.Ulimit, 0, len(svc.Ulimits))
	for ulName, ul := range svc.Ulimits {
		if ul.Single != 0 {
			ulimits = append(ulimits, &units.Ulimit{
				Name: ulName,
				Hard: int64(ul.Single),
				Soft: int64(ul.Single),
			})
		} else {
			ulimits = append(ulimits, &units.Ulimit{
				Name: ulName,
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

	dns := make([]netip.Addr, 0, len(svc.DNS))
	for _, d := range svc.DNS {
		addr, err := netip.ParseAddr(d)
		if err != nil {
			log.Warn().Err(err).Str("dns", d).Msg("Skipping invalid DNS address")
			continue
		}
		dns = append(dns, addr)
	}

	hostConfig := &container.HostConfig{
		PortBindings:   portBindings,
		RestartPolicy:  restartPolicy,
		Mounts:         mounts,
		Privileged:     svc.Privileged,
		ReadonlyRootfs: svc.ReadOnly,
		ExtraHosts:     extraHosts,
		DNS:            dns,
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
