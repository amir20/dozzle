package agent

import (
	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/container/agent/pb"
	"github.com/amir20/dozzle/internal/utils"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func containerToProto(c container.Container) pb.Container {
	var pbStats []*pb.ContainerStat
	for _, stat := range c.Stats.Data() {
		pbStats = append(pbStats, &pb.ContainerStat{
			Id:             stat.ID,
			CpuPercent:     stat.CPUPercent,
			MemoryPercent:  stat.MemoryPercent,
			MemoryUsage:    stat.MemoryUsage,
			NetworkRxTotal: stat.NetworkRxTotal,
			NetworkTxTotal: stat.NetworkTxTotal,
			DiskReadTotal:  stat.DiskReadTotal,
			DiskWriteTotal: stat.DiskWriteTotal,
		})
	}

	pbMounts := make([]*pb.Mount, 0, len(c.Mounts))
	for _, m := range c.Mounts {
		pbMounts = append(pbMounts, &pb.Mount{
			Type:        m.Type,
			Source:      m.Source,
			Destination: m.Destination,
			Rw:          m.RW,
		})
	}

	pbMountStats := make([]*pb.MountStat, 0, len(c.MountStats))
	for _, ms := range c.MountStats {
		pbMountStats = append(pbMountStats, &pb.MountStat{
			Destination: ms.Destination,
			Total:       ms.Total,
			Free:        ms.Free,
			Used:        ms.Used,
			Available:   ms.Available,
			LastChecked: timestamppb.New(ms.LastChecked),
		})
	}

	return pb.Container{
		Id:            c.ID,
		Name:          c.Name,
		Image:         c.Image,
		Created:       timestamppb.New(c.Created),
		State:         c.State,
		Health:        c.Health,
		Host:          c.Host,
		Tty:           c.Tty,
		Labels:        c.Labels,
		Group:         c.Group,
		Started:       timestamppb.New(c.StartedAt),
		Finished:      timestamppb.New(c.FinishedAt),
		Stats:         pbStats,
		Command:       c.Command,
		MemoryLimit:   c.MemoryLimit,
		CpuLimit:      c.CPULimit,
		FullyLoaded:   c.FullyLoaded,
		Env:           c.Env,
		Ports:         c.Ports,
		Mounts:        pbMounts,
		MountStats:    pbMountStats,
		RestartPolicy: c.RestartPolicy,
		NetworkMode:   c.NetworkMode,
	}
}

func containerFromProto(c *pb.Container) container.Container {
	var stats []container.ContainerStat
	for _, stat := range c.Stats {
		stats = append(stats, container.ContainerStat{
			ID:             stat.Id,
			CPUPercent:     stat.CpuPercent,
			MemoryPercent:  stat.MemoryPercent,
			MemoryUsage:    stat.MemoryUsage,
			NetworkRxTotal: stat.NetworkRxTotal,
			NetworkTxTotal: stat.NetworkTxTotal,
			DiskReadTotal:  stat.DiskReadTotal,
			DiskWriteTotal: stat.DiskWriteTotal,
		})
	}

	labels := c.Labels
	if labels == nil {
		labels = make(map[string]string)
	}

	env := c.Env
	if env == nil {
		env = []string{}
	}

	mounts := make([]container.Mount, 0, len(c.Mounts))
	for _, m := range c.Mounts {
		mounts = append(mounts, container.Mount{
			Type:        m.Type,
			Source:      m.Source,
			Destination: m.Destination,
			RW:          m.Rw,
		})
	}

	var mountStats map[string]container.MountStat
	if len(c.MountStats) > 0 {
		mountStats = make(map[string]container.MountStat, len(c.MountStats))
		for _, ms := range c.MountStats {
			mountStats[ms.Destination] = container.MountStat{
				Destination: ms.Destination,
				Total:       ms.Total,
				Free:        ms.Free,
				Used:        ms.Used,
				Available:   ms.Available,
				LastChecked: ms.LastChecked.AsTime(),
			}
		}
	}

	return container.Container{
		ID:            c.Id,
		Name:          c.Name,
		Image:         c.Image,
		Labels:        labels,
		Group:         c.Group,
		Created:       c.Created.AsTime(),
		State:         c.State,
		Health:        c.Health,
		Host:          c.Host,
		Tty:           c.Tty,
		Command:       c.Command,
		StartedAt:     c.Started.AsTime(),
		FinishedAt:    c.Finished.AsTime(),
		Stats:         utils.RingBufferFrom(300, stats),
		MemoryLimit:   c.MemoryLimit,
		CPULimit:      c.CpuLimit,
		FullyLoaded:   c.FullyLoaded,
		Env:           env,
		Ports:         c.Ports,
		Mounts:        mounts,
		MountStats:    mountStats,
		RestartPolicy: c.RestartPolicy,
		NetworkMode:   c.NetworkMode,
	}
}
