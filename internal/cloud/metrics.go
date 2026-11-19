package cloud

import (
	"context"
	"fmt"

	pb "github.com/amir20/dozzle/proto/cloud"
	"google.golang.org/grpc/metadata"
)

// MetricPoint is one bucket of a container's history.
type MetricPoint struct {
	// Milliseconds, because the browser reads this straight into a Date and
	// nanoseconds do not survive JSON as a number.
	TS          int64   `json:"ts"`
	CPU         float64 `json:"cpu"`
	Memory      float64 `json:"memory"`
	MemoryUsage int64   `json:"memoryUsage"`

	// Pointers because each one is absent from an older cloud, and the panel
	// tells "not sent" apart from a real zero: a missing max falls back to the
	// average, a missing rate hides its chart.
	CPUMax         *float64 `json:"cpuMax,omitempty"`
	MemoryUsageMax *int64   `json:"memoryUsageMax,omitempty"`
	NetworkRx      *float64 `json:"networkRx,omitempty"`
	NetworkTx      *float64 `json:"networkTx,omitempty"`
	DiskRead       *float64 `json:"diskRead,omitempty"`
	DiskWrite      *float64 `json:"diskWrite,omitempty"`
}

// MetricResult is what Cloud remembers about one container's resource use.
type MetricResult struct {
	Points []MetricPoint `json:"points"`
	// The bucket width cloud used, milliseconds. The panel says it out loud so
	// a flat line is read as an average and not as a missing sample.
	Bucket int64 `json:"bucket"`
}

// GetContainerMetrics reads back the stats this instance has been pushing.
//
// Dozzle keeps 300 samples in the browser and nothing behind them, so the
// question this answers ("what did it look like an hour ago") has no local
// answer to gate. The caller is responsible for confining the request to a
// container the user may see: Cloud scopes to the instance, not to a user.
func (c *Client) GetContainerMetrics(ctx context.Context, containerName, hostID string, sinceNs, untilNs int64, buckets int32) (*MetricResult, error) {
	apiKey := c.apiKeyFunc()
	if apiKey == "" {
		return nil, ErrNotConfigured
	}

	client, err := c.unaryServiceClient()
	if err != nil {
		return nil, err
	}

	mdPairs := []string{"x-api-key", apiKey}
	if c.instanceID != "" {
		mdPairs = append(mdPairs, "x-instance-id", c.instanceID)
	}
	callCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs(mdPairs...))

	resp, err := client.GetContainerMetrics(callCtx, &pb.GetContainerMetricsRequest{
		ContainerName: containerName,
		HostId:        hostID,
		SinceTsNs:     sinceNs,
		UntilTsNs:     untilNs,
		Buckets:       buckets,
	})
	if err != nil {
		return nil, fmt.Errorf("cloud: container metrics: %w", err)
	}

	return metricResultFromProto(resp), nil
}

func metricResultFromProto(resp *pb.GetContainerMetricsResponse) *MetricResult {
	result := &MetricResult{Points: make([]MetricPoint, 0, len(resp.GetPoints())), Bucket: resp.GetBucketNs() / 1e6}
	for _, p := range resp.GetPoints() {
		result.Points = append(result.Points, MetricPoint{
			TS:          p.GetTsNs() / 1e6,
			CPU:         p.GetCpu(),
			Memory:      p.GetMemory(),
			MemoryUsage: p.GetMemoryUsage(),

			CPUMax:         p.CpuMax,
			MemoryUsageMax: p.MemoryUsageMax,
			NetworkRx:      p.NetworkRxRate,
			NetworkTx:      p.NetworkTxRate,
			DiskRead:       p.DiskReadRate,
			DiskWrite:      p.DiskWriteRate,
		})
	}
	return result
}

// HostMetricPoint is one bucket of a host's history: its containers summed.
type HostMetricPoint struct {
	// Milliseconds, as for MetricPoint.
	TS          int64   `json:"ts"`
	CPU         float64 `json:"cpu"`
	MemoryUsage int64   `json:"memoryUsage"`

	// Absent from an older cloud, as on MetricPoint.
	CPUMax         *float64 `json:"cpuMax,omitempty"`
	MemoryUsageMax *int64   `json:"memoryUsageMax,omitempty"`
	NetworkRx      *float64 `json:"networkRx,omitempty"`
	NetworkTx      *float64 `json:"networkTx,omitempty"`
	DiskRead       *float64 `json:"diskRead,omitempty"`
	DiskWrite      *float64 `json:"diskWrite,omitempty"`
}

// HostTopContainer is one container's share of a host over the window.
type HostTopContainer struct {
	Name string  `json:"name"`
	Avg  float64 `json:"avg"`
	Peak float64 `json:"peak"`
}

// HostTop is a host's heaviest containers, one list per metric.
type HostTop struct {
	CPU     []HostTopContainer `json:"cpu"`
	Memory  []HostTopContainer `json:"memory"`
	Network []HostTopContainer `json:"network"`
	Disk    []HostTopContainer `json:"disk"`
}

// HostMetrics is one host's series.
type HostMetrics struct {
	HostID     string            `json:"hostId"`
	Points     []HostMetricPoint `json:"points"`
	Containers int32             `json:"containers"`
	// Nil for a reader who may not see every container: the names are theirs to
	// see only per container, and the totals above already say how busy it is.
	Top *HostTop `json:"top,omitempty"`
}

// HostMetricResult is what Cloud remembers about every host on the instance.
type HostMetricResult struct {
	Hosts []HostMetrics `json:"hosts"`
	// The bucket width cloud used, milliseconds.
	Bucket int64 `json:"bucket"`
}

// HostMetricsQuery is what the host panel asks Cloud for.
type HostMetricsQuery struct {
	SinceNs, UntilNs int64
	Buckets          int32
	// The one top-container list to compute ("cpu", "memory", "network",
	// "disk"), or empty for none. TopOnly skips the points and the count.
	Top     string
	TopOnly bool
}

// GetHostMetrics reads back the stats this instance has been pushing, summed
// per host. Cloud answers for every host on the instance.
func (c *Client) GetHostMetrics(ctx context.Context, q HostMetricsQuery) (*HostMetricResult, error) {
	apiKey := c.apiKeyFunc()
	if apiKey == "" {
		return nil, ErrNotConfigured
	}

	client, err := c.unaryServiceClient()
	if err != nil {
		return nil, err
	}

	mdPairs := []string{"x-api-key", apiKey}
	if c.instanceID != "" {
		mdPairs = append(mdPairs, "x-instance-id", c.instanceID)
	}
	callCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs(mdPairs...))

	resp, err := client.GetHostMetrics(callCtx, &pb.GetHostMetricsRequest{
		SinceTsNs: q.SinceNs,
		UntilTsNs: q.UntilNs,
		Buckets:   q.Buckets,
		Top:       q.Top,
		TopOnly:   q.TopOnly,
	})
	if err != nil {
		return nil, fmt.Errorf("cloud: host metrics: %w", err)
	}

	return hostMetricResultFromProto(resp), nil
}

func hostMetricResultFromProto(resp *pb.GetHostMetricsResponse) *HostMetricResult {
	result := &HostMetricResult{Hosts: make([]HostMetrics, 0, len(resp.GetHosts())), Bucket: resp.GetBucketNs() / 1e6}
	for _, h := range resp.GetHosts() {
		host := HostMetrics{
			HostID:     h.GetHostId(),
			Points:     make([]HostMetricPoint, 0, len(h.GetPoints())),
			Containers: h.GetContainers(),
			Top: &HostTop{
				CPU:     topFromProto(h.GetTopCpu()),
				Memory:  topFromProto(h.GetTopMemory()),
				Network: topFromProto(h.GetTopNetwork()),
				Disk:    topFromProto(h.GetTopDisk()),
			},
		}
		for _, p := range h.GetPoints() {
			host.Points = append(host.Points, HostMetricPoint{
				TS:             p.GetTsNs() / 1e6,
				CPU:            p.GetCpu(),
				MemoryUsage:    p.GetMemoryUsage(),
				CPUMax:         p.CpuMax,
				MemoryUsageMax: p.MemoryUsageMax,
				NetworkRx:      p.NetworkRxRate,
				NetworkTx:      p.NetworkTxRate,
				DiskRead:       p.DiskReadRate,
				DiskWrite:      p.DiskWriteRate,
			})
		}
		result.Hosts = append(result.Hosts, host)
	}
	return result
}

// topFromProto never returns nil, so the browser gets [] rather than null.
func topFromProto(list []*pb.HostTopContainer) []HostTopContainer {
	out := make([]HostTopContainer, 0, len(list))
	for _, c := range list {
		out = append(out, HostTopContainer{Name: c.GetName(), Avg: c.GetAvg(), Peak: c.GetPeak()})
	}
	return out
}
