package cloud

import (
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

func executeListHosts(hostService ToolHostService) (*pb.CallToolResponse, error) {
	hosts := hostService.Hosts()
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

func executeFindContainers(argsJSON string, hostService ToolHostService, labels container.ContainerLabels) (*pb.CallToolResponse, error) {
	var args findContainersArgs
	if argsJSON != "" {
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return nil, fmt.Errorf("failed to parse arguments: %w", err)
		}
	}

	containers, errs := hostService.ListAllContainers(labels)
	logHostErrors(errs)
	hostNames := buildHostNameMap(hostService)

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

func executeListRunningContainers(hostService ToolHostService, labels container.ContainerLabels) (*pb.CallToolResponse, error) {
	containers, errs := hostService.ListAllContainers(labels)
	logHostErrors(errs)
	hostNames := buildHostNameMap(hostService)

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

func executeListAllContainers(hostService ToolHostService, labels container.ContainerLabels) (*pb.CallToolResponse, error) {
	containers, errs := hostService.ListAllContainers(labels)
	logHostErrors(errs)
	hostNames := buildHostNameMap(hostService)

	result := make([]*pb.ContainerInfo, 0, len(containers))
	for _, c := range containers {
		result = append(result, containerToProto(c, hostNames))
	}
	return &pb.CallToolResponse{
		Success: true,
		Result:  &pb.CallToolResponse_ListContainers{ListContainers: &pb.ListContainersResult{Containers: result}},
	}, nil
}

func executeGetRunningContainerStats(hostService ToolHostService, labels container.ContainerLabels) (*pb.CallToolResponse, error) {
	containers, errs := hostService.ListAllContainers(labels)
	logHostErrors(errs)
	hostNames := buildHostNameMap(hostService)

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

		var maxCPU, maxMem float64
		for _, s := range stats {
			maxCPU = max(maxCPU, s.CPUPercent)
			maxMem = max(maxMem, s.MemoryPercent)
		}

		result = append(result, &pb.ContainerStatEntry{
			Id:             c.ID,
			Name:           c.Name,
			Host:           resolveHostName(c.Host, hostNames),
			CpuPercent:     latest.CPUPercent,
			MemoryPercent:  latest.MemoryPercent,
			MemoryUsage:    latest.MemoryUsage,
			MaxCpu_5Min:    maxCPU,
			MaxMemory_5Min: maxMem,
		})
	}
	return &pb.CallToolResponse{
		Success: true,
		Result:  &pb.CallToolResponse_ContainerStats{ContainerStats: &pb.ContainerStatsResult{Stats: result}},
	}, nil
}

func executeInspectContainer(argsJSON string, hostService ToolHostService, labels container.ContainerLabels) (*pb.CallToolResponse, error) {
	var args inspectContainerArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}
	if args.ContainerID == "" || args.Host == "" {
		return nil, fmt.Errorf("container_id and host are required")
	}

	cs, err := hostService.FindContainer(args.Host, args.ContainerID, labels)
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
			HostName:      resolveHostName(c.Host, buildHostNameMap(hostService)),
			HostId:        c.Host,
			Labels:        c.Labels,
			MemoryLimit:   c.MemoryLimit,
			CpuLimit:      c.CPULimit,
			Ports:         c.Ports,
			Mounts:        c.Mounts,
			RestartPolicy: c.RestartPolicy,
			NetworkMode:   c.NetworkMode,
		}},
	}, nil
}
