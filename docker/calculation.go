package docker

import "github.com/docker/docker/api/types"

func calculateMemUsageUnixNoCache(mem types.MemoryStats) float64 {
	// re implementation of the docker calculation
	// https://github.com/docker/cli/blob/53f8ed4bec07084db4208f55987a2ea94b7f01d6/cli/command/container/stats_helpers.go#L227-L249
	// cgroup v1
	if v, isCGroup := mem.Stats["total_inactive_file"]; isCGroup && v < mem.Usage {
		return float64(mem.Usage - v)
	}
	// cgroup v2
	if v := mem.Stats["inactive_file"]; v < mem.Usage {
		return float64(mem.Usage - v)
	}
	return float64(mem.Usage)
}
