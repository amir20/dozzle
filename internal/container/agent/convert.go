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

	var pbVolumes []*pb.VolumeUsage
	for _, v := range c.Volumes {
		pbVolumes = append(pbVolumes, &pb.VolumeUsage{Name: v.Name, Destination: v.Destination, Size: v.Size, Links: v.Links})
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
		RestartCount:  int32(c.RestartCount),
		OomKilled:     c.OOMKilled,
		ExitCode:      int32(c.ExitCode),
		SizeRw:        c.SizeRw,
		Volumes:       pbVolumes,
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

	var volumes []container.VolumeUsage
	for _, v := range c.Volumes {
		volumes = append(volumes, container.VolumeUsage{Name: v.Name, Destination: v.Destination, Size: v.Size, Links: v.Links})
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
		RestartCount:  int(c.RestartCount),
		OOMKilled:     c.OomKilled,
		ExitCode:      int(c.ExitCode),
		SizeRw:        c.SizeRw,
		Volumes:       volumes,
	}
}

// setHostMetricsProto copies the host-level metrics an agent reads off its own
// machine onto the HostInfo reply.
func setHostMetricsProto(dst *pb.Host, h container.Host) {
	dst.MetricsAvailable = h.MetricsAvailable
	dst.Load1, dst.Load5, dst.Load15 = h.Load1, h.Load5, h.Load15
	dst.Uptime = h.Uptime
	dst.DiskTotal, dst.DiskFree = h.DiskTotal, h.DiskFree
	for _, d := range h.Disks {
		dst.Disks = append(dst.Disks, &pb.Disk{Name: d.Name, Total: d.Total, Free: d.Free})
	}
	if r := h.Reclaimable; r != nil {
		dst.Reclaimable = &pb.Reclaimable{
			Images: r.Images, ImagesSize: r.ImagesSize,
			Volumes: r.Volumes, VolumesSize: r.VolumesSize,
			Containers: r.Containers, ContainersSize: r.ContainersSize,
			BuildCacheSize: r.BuildCacheSize,
		}
	}
}

// hostMetricsFromProto is the reverse. An agent older than these fields leaves
// them all unset, which reads as a host with no metrics to show.
func hostMetricsFromProto(src *pb.Host) (container.HostMetrics, bool) {
	m := container.HostMetrics{
		Load1:     src.GetLoad1(),
		Load5:     src.GetLoad5(),
		Load15:    src.GetLoad15(),
		Uptime:    src.GetUptime(),
		DiskTotal: src.GetDiskTotal(),
		DiskFree:  src.GetDiskFree(),
	}
	for _, d := range src.GetDisks() {
		m.Disks = append(m.Disks, container.Disk{Name: d.GetName(), Total: d.GetTotal(), Free: d.GetFree()})
	}
	if r := src.GetReclaimable(); r != nil {
		m.Reclaimable = &container.Reclaimable{
			Images: r.GetImages(), ImagesSize: r.GetImagesSize(),
			Volumes: r.GetVolumes(), VolumesSize: r.GetVolumesSize(),
			Containers: r.GetContainers(), ContainersSize: r.GetContainersSize(),
			BuildCacheSize: r.GetBuildCacheSize(),
		}
	}
	return m, src.GetMetricsAvailable()
}

func updateResultToProto(r *container.UpdateResult) *pb.UpdateResult {
	if r == nil {
		return nil
	}
	out := &pb.UpdateResult{
		OldId:       r.OldID,
		NewId:       r.NewID,
		FromImageId: r.FromImageID,
		ToImageId:   r.ToImageID,
		FromDigest:  r.FromDigest,
		ToDigest:    r.ToDigest,
		RolledBack:  r.RolledBack,
	}
	// A zero time is left unset, so it reads back as zero rather than as the
	// Unix epoch.
	if !r.OldStartedAt.IsZero() {
		out.OldStartedAt = timestamppb.New(r.OldStartedAt)
	}
	return out
}

func updateResultFromProto(r *pb.UpdateResult) *container.UpdateResult {
	if r == nil {
		return nil
	}
	out := &container.UpdateResult{
		OldID:       r.GetOldId(),
		NewID:       r.GetNewId(),
		FromImageID: r.GetFromImageId(),
		ToImageID:   r.GetToImageId(),
		FromDigest:  r.GetFromDigest(),
		ToDigest:    r.GetToDigest(),
		RolledBack:  r.GetRolledBack(),
	}
	if t := r.GetOldStartedAt(); t != nil {
		out.OldStartedAt = t.AsTime()
	}
	return out
}
