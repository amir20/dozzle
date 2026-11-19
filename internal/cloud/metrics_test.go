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

func TestHostMetricResultFromProto(t *testing.T) {
	resp := &pb.GetHostMetricsResponse{
		BucketNs: 60_000_000_000,
		Hosts: []*pb.HostMetricSeries{{
			HostId: "h1",
			Points: []*pb.HostMetricPoint{{TsNs: 1_700_000_000_000_000_000, Cpu: 42.5, MemoryUsage: 1 << 30}},
		}},
	}

	result := hostMetricResultFromProto(resp)
	require.Len(t, result.Hosts, 1)
	assert.Equal(t, "h1", result.Hosts[0].HostID)
	assert.Equal(t, int64(60_000), result.Bucket)
	require.Len(t, result.Hosts[0].Points, 1)
	assert.Equal(t, HostMetricPoint{TS: 1_700_000_000_000, CPU: 42.5, MemoryUsage: 1 << 30}, result.Hosts[0].Points[0])
}

func TestHostMetricResultFromProtoCarriesPeaksRatesAndTop(t *testing.T) {
	resp := &pb.GetHostMetricsResponse{
		Hosts: []*pb.HostMetricSeries{{
			HostId:     "h1",
			Containers: 7,
			Points: []*pb.HostMetricPoint{{
				TsNs: 1_000_000, Cpu: 10, MemoryUsage: 100,
				CpuMax: proto.Float64(30), MemoryUsageMax: proto.Int64(150),
				NetworkRxRate: proto.Float64(1), NetworkTxRate: proto.Float64(2),
				DiskReadRate: proto.Float64(3), DiskWriteRate: proto.Float64(4),
			}},
			TopCpu:    []*pb.HostTopContainer{{Name: "api", Avg: 5, Peak: 9}},
			TopMemory: []*pb.HostTopContainer{{Name: "db", Avg: 1 << 20, Peak: 1 << 21}},
		}},
	}

	host := hostMetricResultFromProto(resp).Hosts[0]
	assert.Equal(t, int32(7), host.Containers)
	p := host.Points[0]
	assert.Equal(t, 30.0, *p.CPUMax)
	assert.Equal(t, int64(150), *p.MemoryUsageMax)
	assert.Equal(t, []float64{1, 2, 3, 4}, []float64{*p.NetworkRx, *p.NetworkTx, *p.DiskRead, *p.DiskWrite})
	require.NotNil(t, host.Top)
	assert.Equal(t, []HostTopContainer{{Name: "api", Avg: 5, Peak: 9}}, host.Top.CPU)
	assert.Equal(t, "db", host.Top.Memory[0].Name)

	// An older cloud sends no lists: empty, never null, so the browser reads [].
	body, err := json.Marshal(host.Top)
	require.NoError(t, err)
	assert.JSONEq(t, `{"cpu":[{"name":"api","avg":5,"peak":9}],"memory":[{"name":"db","avg":1048576,"peak":2097152}],"network":[],"disk":[]}`, string(body))
}
