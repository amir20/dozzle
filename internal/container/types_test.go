package container

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/amir20/dozzle/internal/utils"
	"github.com/go-faker/faker/v4"
	"github.com/go-faker/faker/v4/pkg/options"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProto(t *testing.T) {
	expected := Container{}
	faker.FakeData(&expected, options.WithFieldsToIgnore("Stats", "MountStats"))
	expected.FinishedAt = expected.FinishedAt.UTC()
	expected.Created = expected.Created.UTC()
	expected.StartedAt = expected.StartedAt.UTC()
	expected.Stats = utils.NewRingBuffer[ContainerStat](300)

	pb := expected.ToProto()
	actual := FromProto(&pb)

	assert.Equal(t, expected, actual)

}

func TestContainerStat_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		stat     ContainerStat
		expected string
	}{
		{
			name: "keeps the id when a stat event carries one",
			stat: ContainerStat{ID: "abc123", CPUPercent: 12.5, MemoryPercent: 3.25, MemoryUsage: 1024},
			expected: `{"id":"abc123","cpu":12.5,"memory":3.25,"memoryUsage":1024,` +
				`"networkRxTotal":0,"networkTxTotal":0,"diskReadTotal":0,"diskWriteTotal":0}`,
		},
		{
			name: "drops the id on a buffered history point",
			stat: ContainerStat{CPUPercent: 0.4212580283090664, MemoryPercent: 1.4223477542648744, MemoryUsage: 268435456.4},
			expected: `{"cpu":0.42,"memory":1.42,"memoryUsage":268435456,` +
				`"networkRxTotal":0,"networkTxTotal":0,"diskReadTotal":0,"diskWriteTotal":0}`,
		},
		{
			name: "writes zeros short rather than padded",
			stat: ContainerStat{NetworkRxTotal: 42, DiskWriteTotal: 7},
			expected: `{"cpu":0,"memory":0,"memoryUsage":0,` +
				`"networkRxTotal":42,"networkTxTotal":0,"diskReadTotal":0,"diskWriteTotal":7}`,
		},
		{
			name: "falls back to 0 for values json cannot encode",
			stat: ContainerStat{CPUPercent: math.NaN(), MemoryPercent: math.Inf(1), MemoryUsage: math.Inf(-1)},
			expected: `{"cpu":0,"memory":0,"memoryUsage":0,` +
				`"networkRxTotal":0,"networkTxTotal":0,"diskReadTotal":0,"diskWriteTotal":0}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := json.Marshal(tt.stat)
			require.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(actual))
			assert.Equal(t, tt.expected, string(actual))
		})
	}
}

func TestContainerStat_MarshalJSON_roundTrips(t *testing.T) {
	stat := ContainerStat{
		ID:             "abc123",
		CPUPercent:     12.34,
		MemoryPercent:  56.78,
		MemoryUsage:    268435456,
		NetworkRxTotal: 1234567890,
		NetworkTxTotal: 987654321,
		DiskReadTotal:  555,
		DiskWriteTotal: 666,
	}

	b, err := json.Marshal(stat)
	require.NoError(t, err)

	var actual ContainerStat
	require.NoError(t, json.Unmarshal(b, &actual))
	assert.Equal(t, stat, actual)
}
