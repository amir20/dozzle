package web

import (
	"encoding/json"
	"testing"

	"github.com/amir20/dozzle/internal/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The metrics tick must not carry available or type: the raw client's Host() has
// neither set correctly, and the UI would take them as the host going offline.
func TestHostMetricsEventCarriesOnlyMetrics(t *testing.T) {
	host := container.Host{ID: "abc", Type: "local", Available: false, MetricsAvailable: true, Load1: 1.5, DiskTotal: 100}
	require.True(t, hostHasMetrics(host))

	b, err := json.Marshal(newHostMetricsEvent(host))
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(b, &got))
	assert.NotContains(t, got, "available")
	assert.NotContains(t, got, "type")
	assert.Equal(t, "abc", got["id"])
	assert.Equal(t, 1.5, got["load1"])
	// zero values are sent so a value that drops to zero replaces the stale one
	assert.Contains(t, got, "uptime")
	// and an empty array, not null, so an unmounted drive clears
	assert.Equal(t, []any{}, got["disks"])
}

func TestHostHasMetrics(t *testing.T) {
	assert.False(t, hostHasMetrics(container.Host{Type: "local"}))
	assert.True(t, hostHasMetrics(container.Host{Type: "local", DiskTotal: 1}))
	assert.True(t, hostHasMetrics(container.Host{Type: "local", Disks: []container.Disk{{Name: "media", Total: 1}}}))
	// agents report their own machine now; an older one sends nothing and stays out
	assert.True(t, hostHasMetrics(container.Host{Type: "agent", MetricsAvailable: true}))
	assert.False(t, hostHasMetrics(container.Host{Type: "agent"}))
}

// A swarm node reached both locally and as an agent reports twice per tick;
// hosts with nothing to show come back nil.
func TestCollapseHostMetrics(t *testing.T) {
	a := hostMetricsEvent{ID: "a", Load1: 1}
	b := hostMetricsEvent{ID: "b", Load1: 2}
	dup := hostMetricsEvent{ID: "a", Load1: 1}

	got := collapseHostMetrics([]*hostMetricsEvent{&a, nil, &b, &dup})
	assert.Equal(t, []hostMetricsEvent{a, b}, got)
}
