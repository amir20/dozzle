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
func (c *Client) GetContainerMetrics(ctx context.Context, containerID string, sinceNs, untilNs int64, buckets int32) (*MetricResult, error) {
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
		ContainerId: containerID,
		SinceTsNs:   sinceNs,
		UntilTsNs:   untilNs,
		Buckets:     buckets,
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
		})
	}
	return result
}
