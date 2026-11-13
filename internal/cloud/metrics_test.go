package cloud

import (
	"encoding/json"
	"testing"

	pb "github.com/amir20/dozzle/proto/cloud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestMetricResultFromProtoCarriesMaxAndRates(t *testing.T) {
	resp := &pb.GetContainerMetricsResponse{
		BucketNs: 3_600_000_000_000,
		Points: []*pb.ContainerMetricPoint{{
			TsNs:           1_700_000_000_000_000_000,
			Cpu:            9.8,
			Memory:         40,
			MemoryUsage:    470_000_000,
			CpuMax:         new(15.9),
			MemoryUsageMax: proto.Int64(500_000_000),
			NetworkRxRate:  proto.Float64(2048),
			NetworkTxRate:  proto.Float64(0),
			DiskReadRate:   proto.Float64(10),
			DiskWriteRate:  proto.Float64(20),
		}},
	}

	result := metricResultFromProto(resp)
	require.Len(t, result.Points, 1)
	p := result.Points[0]
	assert.Equal(t, int64(1_700_000_000_000), p.TS)
	assert.Equal(t, int64(3_600_000), result.Bucket)
	assert.Equal(t, 15.9, *p.CPUMax)
	assert.Equal(t, int64(500_000_000), *p.MemoryUsageMax)
	assert.Equal(t, 2048.0, *p.NetworkRx)
	// A real zero is sent and kept, not mistaken for absence.
	require.NotNil(t, p.NetworkTx)
	assert.Equal(t, 0.0, *p.NetworkTx)
	assert.Equal(t, 10.0, *p.DiskRead)
	assert.Equal(t, 20.0, *p.DiskWrite)

	body, err := json.Marshal(p)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"networkTx":0`)
}

// An older cloud sends averages only. The new fields must be absent from the
// JSON, so the panel falls back to the averages and hides the I/O charts instead
// of drawing zeros as though they were measured.
func TestMetricResultFromProtoOlderCloudOmitsNewFields(t *testing.T) {
	resp := &pb.GetContainerMetricsResponse{Points: []*pb.ContainerMetricPoint{{TsNs: 1e9, Cpu: 5, Memory: 10, MemoryUsage: 100}}}

	body, err := json.Marshal(metricResultFromProto(resp).Points[0])
	require.NoError(t, err)
	assert.JSONEq(t, `{"ts":1000,"cpu":5,"memory":10,"memoryUsage":100}`, string(body))
}
