package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/amir20/dozzle/internal/container"
	pb "github.com/amir20/dozzle/proto/cloud"
)

type inspectContainerArgs struct {
	ContainerID string `json:"container_id"`
	Host        string `json:"host_id"`
}

type findContainersArgs struct {
	Name   string `json:"name"`
	Image  string `json:"image"`
	State  string `json:"state"`
	Health string `json:"health"`
}

func executeListHosts(deps ToolDeps) (*pb.CallToolResponse, error) {
	hosts := deps.scoped().Hosts()
	result := make([]*pb.HostInfo, len(hosts))
	for i, h := range hosts {
		result[i] = &pb.HostInfo{
			Id:            h.ID,
			Name:          h.Name,
			NCpu:          int32(h.NCPU),
			MemTotal:      h.MemTotal,
			DockerVersion: h.DockerVersion,
			AgentVersion:  h.AgentVersion,
			Type:          h.Type,
			Available:     h.Available,
		}
	}
	return &pb.CallToolResponse{
		Success: true,
		Result:  &pb.CallToolResponse_ListHosts{ListHosts: &pb.ListHostsResult{Hosts: result}},
	}, nil
}

func executeFindContainers(argsJSON string, deps ToolDeps) (*pb.CallToolResponse, error) {
	var args findContainersArgs
	if argsJSON != "" {
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return nil, fmt.Errorf("failed to parse arguments: %w", err)
		}
	}

	containers, errs := deps.scoped().ListAllContainers()
	logHostErrors(errs)
	hostNames := buildHostNameMap(deps.HostService)

	result := make([]*pb.ContainerInfo, 0, len(containers))
	for _, c := range containers {
		if args.Name != "" && !containsIgnoreCase(c.Name, args.Name) {
			continue
		}
		if args.Image != "" && !containsIgnoreCase(c.Image, args.Image) {
			continue
		}
		if args.State != "" && !strings.EqualFold(c.State, args.State) {
			continue
		}
		if args.Health != "" && !strings.EqualFold(c.Health, args.Health) {
			continue
		}
		result = append(result, containerToProto(c, hostNames))
	}
	return &pb.CallToolResponse{
		Success: true,
		Result:  &pb.CallToolResponse_ListContainers{ListContainers: &pb.ListContainersResult{Containers: result}},
	}, nil
}

func executeListRunningContainers(deps ToolDeps) (*pb.CallToolResponse, error) {
	containers, errs := deps.scoped().ListAllContainers()
	logHostErrors(errs)
	hostNames := buildHostNameMap(deps.HostService)

	result := make([]*pb.ContainerInfo, 0, len(containers))
	for _, c := range containers {
		if c.State != "running" {
			continue
		}
		result = append(result, containerToProto(c, hostNames))
	}
	return &pb.CallToolResponse{
		Success: true,
		Result:  &pb.CallToolResponse_ListContainers{ListContainers: &pb.ListContainersResult{Containers: result}},
	}, nil
}

func executeListAllContainers(deps ToolDeps) (*pb.CallToolResponse, error) {
	containers, errs := deps.scoped().ListAllContainers()
	logHostErrors(errs)
	hostNames := buildHostNameMap(deps.HostService)

	result := make([]*pb.ContainerInfo, 0, len(containers))
	for _, c := range containers {
		result = append(result, containerToProto(c, hostNames))
	}
	return &pb.CallToolResponse{
		Success: true,
		Result:  &pb.CallToolResponse_ListContainers{ListContainers: &pb.ListContainersResult{Containers: result}},
	}, nil
}

func executeGetRunningContainerStats(deps ToolDeps) (*pb.CallToolResponse, error) {
	containers, errs := deps.scoped().ListAllContainers()
	logHostErrors(errs)
	hostNames := buildHostNameMap(deps.HostService)

	result := make([]*pb.ContainerStatEntry, 0, len(containers))
	for _, c := range containers {
		if c.State != "running" {
			continue
		}
		if c.Stats == nil || c.Stats.Len() == 0 {
			continue
		}

		stats := c.Stats.Data()
		latest := stats[len(stats)-1]
		first := stats[0]

		var maxCPU, maxMem float64
		for _, s := range stats {
			maxCPU = max(maxCPU, s.CPUPercent)
			maxMem = max(maxMem, s.MemoryPercent)
		}

		var rxDelta, txDelta uint64
		if latest.NetworkRxTotal >= first.NetworkRxTotal {
			rxDelta = latest.NetworkRxTotal - first.NetworkRxTotal
		}
		if latest.NetworkTxTotal >= first.NetworkTxTotal {
			txDelta = latest.NetworkTxTotal - first.NetworkTxTotal
		}

		result = append(result, &pb.ContainerStatEntry{
			Id:             c.ID,
			Name:           c.Name,
			Host:           resolveHostName(c.Host, hostNames),
			HostId:         c.Host,
			CpuPercent:     latest.CPUPercent,
			MemoryPercent:  latest.MemoryPercent,
			MemoryUsage:    latest.MemoryUsage,
			MaxCpu_5Min:    maxCPU,
			MaxMemory_5Min: maxMem,
			NetworkRxTotal: latest.NetworkRxTotal,
			NetworkTxTotal: latest.NetworkTxTotal,
			NetworkRx_5Min: rxDelta,
			NetworkTx_5Min: txDelta,
		})
	}
	return &pb.CallToolResponse{
		Success: true,
		Result:  &pb.CallToolResponse_ContainerStats{ContainerStats: &pb.ContainerStatsResult{Stats: result}},
	}, nil
}

func executeInspectContainer(argsJSON string, deps ToolDeps) (*pb.CallToolResponse, error) {
	var args inspectContainerArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}
	// Read-only: resolve an ambiguous name in one shot rather than erroring and
	// forcing a find_containers round-trip. The note is discarded here because
	// inspect already returns the concrete id/name/state, which is itself the
	// disambiguation — no need to mangle those structured fields with prose.
	hostID, containerID, _, err := resolveContainerRefRead(args.ContainerID, args.Host, deps)
	if err != nil {
		return nil, err
	}

	cs, err := deps.scoped().FindContainer(hostID, containerID)
	if err != nil {
		return nil, fmt.Errorf("container not found: %w", err)
	}

	c := cs.Container
	return &pb.CallToolResponse{
		Success: true,
		Result: &pb.CallToolResponse_InspectContainer{InspectContainer: &pb.InspectContainerResult{
			Id:            c.ID,
			Name:          c.Name,
			Image:         c.Image,
			Command:       c.Command,
			Created:       c.Created.UTC().Format(time.RFC3339),
			StartedAt:     c.StartedAt.UTC().Format(time.RFC3339),
			FinishedAt:    formatTimeOrEmpty(c.FinishedAt),
			State:         c.State,
			Health:        c.Health,
			HostName:      resolveHostName(c.Host, buildHostNameMap(deps.HostService)),
			HostId:        c.Host,
			Labels:        c.Labels,
			MemoryLimit:   c.MemoryLimit,
			CpuLimit:      c.CPULimit,
			Ports:         c.Ports,
			Mounts:        mountStrings(c.Mounts),
			RestartPolicy: c.RestartPolicy,
			NetworkMode:   c.NetworkMode,
			RestartCount:  int32(c.RestartCount),
			OomKilled:     c.OOMKilled,
			ExitCode:      int32(c.ExitCode),
		}},
	}, nil
}

type checkImageUpdatesArgs struct {
	Name    string `json:"name"`
	Image   string `json:"image"`
	Refresh bool   `json:"refresh"`
}

// executeCheckImageUpdates answers with every checked container, current ones
// included, so the cloud can say how many are up to date and why some could
// not be checked rather than only listing what is outdated.
func executeCheckImageUpdates(ctx context.Context, argsJSON string, deps ToolDeps) (*pb.CallToolResponse, error) {
	if !deps.ImageCheckMode.Allows(true) {
		return nil, fmt.Errorf("image update checks are off on this Dozzle instance (--image-check-mode)")
	}

	var args checkImageUpdatesArgs
	if argsJSON != "" {
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return nil, fmt.Errorf("failed to parse arguments: %w", err)
		}
	}

	containers, errs := deps.scoped().ListAllContainers()
	logHostErrors(errs)
	var matched []container.Container
	for _, c := range containers {
		if (args.Name == "" || containsIgnoreCase(c.Name, args.Name)) &&
			(args.Image == "" || containsIgnoreCase(c.Image, args.Image)) {
			matched = append(matched, c)
		}
	}
	// The listing is scoped, so each lookup after it can skip the labels
	// rather than re-list a host per container.
	updates := container.CheckImageUpdates(ctx, deps.HostService, matched, args.Refresh)
	hostNames := buildHostNameMap(deps.HostService)

	result := make([]*pb.ContainerInfo, len(updates))
	for i, u := range updates {
		info := containerToProto(u.Container, hostNames)
		info.ImageUpdate = &pb.ImageUpdate{
			Status:       string(u.Result.Status),
			LocalDigest:  u.Result.LocalDigest,
			RemoteDigest: u.Result.RemoteDigest,
			Reason:       u.Result.Reason,
			CheckedAt:    u.Result.CheckedAt.Unix(),
		}
		result[i] = info
	}
	return &pb.CallToolResponse{
		Success: true,
		Result:  &pb.CallToolResponse_ListContainers{ListContainers: &pb.ListContainersResult{Containers: result}},
	}, nil
}
