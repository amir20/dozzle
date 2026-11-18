package cloud

import (
	"testing"

	pb "github.com/amir20/dozzle/proto/cloud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestVisualsFromProto(t *testing.T) {
	from, to := int64(1_700_000_000_000_000_000), int64(1_700_086_400_000_000_000)
	got := visualsFromProto([]*pb.ChatVisual{
		{
			Type: "memory_vs_limit", Container: "rook", HostId: "h1", HostName: "nas", FromNs: from, ToNs: to,
			Facts: &pb.ChatVisualFacts{
				Exits: 24, ExitCodes: []*pb.ChatVisualExitCode{{Code: "137", Count: 17}, {Code: "1", Count: 7}},
				EventsRecorded: true, MemoryPeak: 17179869184,
			},
			Points:      []*pb.ContainerMetricPoint{{TsNs: from, Memory: 99.5, MemoryUsage: 17000000000, MemoryUsageMax: proto.Int64(17179869184)}},
			MemoryLimit: 17179869184,
		},
		{
			Type: "pattern_count", Container: "rook", Query: "Invalid HTTP request", Total: proto.Int64(147),
			Volume: []*pb.SearchVolumeBucket{{TimestampNs: from, Level: "warn", Count: 12}}, StepNs: 3_600_000_000_000,
		},
		{Type: "container_ranking", Metric: "restarts", Rows: []*pb.ChatVisualRankRow{{Container: "rook", HostId: "h1", Value: 24}}},
		{Type: "logs", Container: "rook", Lines: []*pb.ViewLogLine{{TimestampNs: from, Level: "warn", Message: "WARNING: Invalid HTTP request received."}}},
		// CPU carries only its series: the pane reads the peak off it.
		{
			Type: "cpu_over_time", Container: "rook", HostId: "h1", FromNs: from, ToNs: to,
			Points: []*pb.ContainerMetricPoint{{TsNs: from, Cpu: 12.5, CpuMax: new(87.5)}},
		},
		// A type this build cannot draw is dropped, never handed to the pane.
		{Type: "disk_vs_quota", Container: "rook"},
	})

	assert.Len(t, got, 5)

	memory := got[0]
	assert.Equal(t, int64(1_700_000_000_000), memory.From, "windows reach the browser in milliseconds")
	assert.Equal(t, int64(1_700_086_400_000), memory.To)
	assert.Equal(t, []ChatVisualExitCode{{Code: "137", Count: 17}, {Code: "1", Count: 7}}, memory.Facts.ExitCodes)
	assert.True(t, memory.Facts.EventsRecorded)
	assert.Equal(t, int64(17179869184), memory.MemoryLimit)
	assert.Len(t, memory.Points, 1)
	assert.Equal(t, int64(1_700_000_000_000), memory.Points[0].TS, "points go through the metrics panel's conversion")
	assert.Equal(t, int64(17179869184), *memory.Points[0].MemoryUsageMax)

	count := got[1]
	assert.Equal(t, int64(147), *count.Total)
	assert.Equal(t, []SearchVolumeBucket{{TimestampNs: from, Level: "warn", Count: 12}}, count.Volume)

	assert.Equal(t, []ChatVisualRankRow{{Container: "rook", HostID: "h1", Value: 24}}, got[2].Rows)
	assert.Equal(t, "2023-11-14T22:13:20Z", got[3].Lines[0].Timestamp)

	cpu := got[4]
	assert.Equal(t, "cpu_over_time", cpu.Type)
	assert.Equal(t, 12.5, cpu.Points[0].CPU)
	assert.Equal(t, 87.5, *cpu.Points[0].CPUMax)

	assert.Nil(t, visualsFromProto(nil))
}

func TestVisualsFromProtoDropsCPUWithoutPoints(t *testing.T) {
	got := visualsFromProto([]*pb.ChatVisual{
		{Type: "cpu_over_time", Container: "api"},
		{Type: "memory_vs_limit", Container: "api"},
	})
	require.Len(t, got, 1)
	assert.Equal(t, "memory_vs_limit", got[0].Type)
}
