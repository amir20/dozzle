package deploy

import (
	"context"
	"fmt"
	"io"
	"maps"
	"strconv"

	composetypes "github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/rs/zerolog/log"
)

// Deployer deploys a parsed compose project using the Docker API directly.
type Deployer struct {
	cli *client.Client
}

// NewDeployer creates a deployer that uses the given Docker client.
func NewDeployer(cli *client.Client) *Deployer {
	return &Deployer{cli: cli}
}

// Deploy creates all resources defined in the project: networks, volumes,
// then pulls images and creates+starts containers in dependency order.
func (d *Deployer) Deploy(ctx context.Context, project *composetypes.Project) error {
	networkIDs, err := d.createNetworks(ctx, project)
	if err != nil {
		return fmt.Errorf("creating networks: %w", err)
	}

	if err := d.createVolumes(ctx, project); err != nil {
		return fmt.Errorf("creating volumes: %w", err)
	}

	// ForEachService walks services in dependency order.
	return project.ForEachService(nil, func(name string, svc *composetypes.ServiceConfig) error {
		return d.deployService(ctx, project.Name, name, *svc, networkIDs)
	})
}

func (d *Deployer) createNetworks(ctx context.Context, project *composetypes.Project) (map[string]string, error) {
	ids := make(map[string]string)

	for name, netCfg := range project.Networks {
		if bool(netCfg.External) {
			list, err := d.cli.NetworkList(ctx, network.ListOptions{})
			if err != nil {
				return nil, fmt.Errorf("listing networks: %w", err)
			}
			extName := netCfg.Name
			if extName == "" {
				extName = name
			}
			found := false
			for _, n := range list {
				if n.Name == extName {
					ids[name] = n.ID
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("external network %q not found", extName)
			}
			continue
		}

		fullName := netCfg.Name
		if fullName == "" {
			fullName = project.Name + "_" + name
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
		log.Info().Str("volume", fullName).Msg("Created volume")
	}
	return nil
}

func (d *Deployer) deployService(ctx context.Context, projectName, name string, svc composetypes.ServiceConfig, networkIDs map[string]string) error {
	if svc.Image == "" {
		return fmt.Errorf("service %q has no image (build is not supported)", name)
	}

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
	}
	if len(svc.Command) > 0 {
		config.Cmd = []string(svc.Command)
	}
	if len(svc.Entrypoint) > 0 {
		config.Entrypoint = []string(svc.Entrypoint)
	}

	hostConfig := &container.HostConfig{
		PortBindings:  portBindings,
		RestartPolicy: restartPolicy,
		Mounts:        mounts,
		Privileged:    svc.Privileged,
		ExtraHosts:    extraHosts,
		DNS:           svc.DNS,
		CapAdd:        svc.CapAdd,
		CapDrop:       svc.CapDrop,
		NetworkMode:   networkMode,
		Tmpfs:         tmpfs,
		ShmSize:       int64(svc.ShmSize),
		Init:          svc.Init,
		SecurityOpt:   svc.SecurityOpt,
		Sysctls:       svc.Sysctls,
		Resources: container.Resources{
			NanoCPUs: int64(svc.CPUS * 1e9),
			Memory:   int64(svc.MemLimit),
		},
	}

	return config, hostConfig, networkingConfig
}
