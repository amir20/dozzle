package container

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/amir20/dozzle/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestContainer_MarshalJSON_statColumns(t *testing.T) {
	stats := utils.NewRingBuffer[ContainerStat](300)
	stats.Push(ContainerStat{CPUPercent: 1.234, MemoryPercent: 2, MemoryUsage: 1000.4, NetworkRxTotal: 5000, NetworkTxTotal: 10, DiskReadTotal: 7, DiskWriteTotal: 100})
	stats.Push(ContainerStat{CPUPercent: 0.5, MemoryPercent: 2.5, MemoryUsage: 1200, NetworkRxTotal: 5100, NetworkTxTotal: 10, DiskReadTotal: 7, DiskWriteTotal: 150})
	// a restart resets the counters
	stats.Push(ContainerStat{CPUPercent: 0, MemoryPercent: 1, MemoryUsage: 900, NetworkRxTotal: 40, NetworkTxTotal: 0, DiskReadTotal: 0, DiskWriteTotal: 0})

	b, err := json.Marshal(Container{ID: "abc", Name: "web", Stats: stats})
	require.NoError(t, err)

	var actual struct {
		ID    string               `json:"id"`
		Name  string               `json:"name"`
		Stats map[string][]float64 `json:"stats"`
	}
	require.NoError(t, json.Unmarshal(b, &actual))

	assert.Equal(t, "abc", actual.ID)
	assert.Equal(t, "web", actual.Name)
	assert.Equal(t, []float64{1.23, 0.5, 0}, actual.Stats["cpu"])
	assert.Equal(t, []float64{2, 2.5, 1}, actual.Stats["memory"])
	assert.Equal(t, []float64{1000, 200, -300}, actual.Stats["memoryUsage"])
	assert.Equal(t, []float64{5000, 100, -5060}, actual.Stats["networkRxTotal"])
	assert.Equal(t, []float64{10, 0, -10}, actual.Stats["networkTxTotal"])
	assert.Equal(t, []float64{7, 0, -7}, actual.Stats["diskReadTotal"])
	assert.Equal(t, []float64{100, 50, -150}, actual.Stats["diskWriteTotal"])
}

func TestContainer_MarshalJSON_withoutStats(t *testing.T) {
	b, err := json.Marshal(Container{ID: "abc", Labels: map[string]string{"a": "b"}})
	require.NoError(t, err)
	assert.NotContains(t, string(b), `"stats"`)
	assert.Contains(t, string(b), `"labels":{"a":"b"}`)
	assert.NotContains(t, string(b), `"Tty"`)
}

func TestContainer_MarshalJSON_isSmallerThanObjects(t *testing.T) {
	stats := utils.NewRingBuffer[ContainerStat](300)
	var rx, tx, rd, wr uint64 = 351051673, 19232507, 1197924352, 2277904384
	for i := range 300 {
		rx += uint64(1500 + i%7*100)
		tx += uint64(200 + i%3*40)
		wr += uint64(i % 5 * 4096)
		stats.Push(ContainerStat{CPUPercent: 0.03 + float64(i%10)/100, MemoryPercent: 0.12, MemoryUsage: float64(51531776 + i%4*4096), NetworkRxTotal: rx, NetworkTxTotal: tx, DiskReadTotal: rd, DiskWriteTotal: wr})
	}

	columns, err := json.Marshal(Container{Stats: stats})
	require.NoError(t, err)
	objects, err := json.Marshal(stats.Data())
	require.NoError(t, err)

	t.Logf("objects %d bytes, columns %d bytes", len(objects), len(columns))
	assert.Less(t, len(columns), len(objects)/3)
}

func TestTruncateLogLine(t *testing.T) {
	short := "hello\n"
	assert.Equal(t, short, TruncateLogLine(short))

	// A 3-byte rune straddling the cap must not be split.
	long := strings.Repeat("a", MaxLogLineBytes-1) + "€" + "tail\n"
	got := TruncateLogLine(long)
	assert.True(t, utf8.ValidString(got))
	assert.Equal(t, strings.Repeat("a", MaxLogLineBytes-1)+LogTruncationSuffix+"\n", got)
}
