package web

import (
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateMarkers(t *testing.T) {
	records := container.NewUpdateRecords()
	at := time.Date(2026, 10, 1, 3, 0, 0, 0, time.UTC)
	records.Add(container.UpdateRecord{Host: "nas", Name: "web", OldID: "old000000000", NewID: "web000000000", At: at, Source: container.UpdateSourceSchedule})
	records.Add(container.UpdateRecord{Host: "nas", Name: "db", NewID: "db0000000000", At: at})

	m := newUpdateMarkers(records, []container.Container{{Host: "nas", ID: "web000000000"}, {Host: "other", ID: "db0000000000"}})
	opening := m.opening()
	require.Len(t, opening, 1, "only the update that created a container on screen, on its own host")
	assert.Equal(t, "web", opening[0].Name)

	// A container the update creates starts before the update is recorded.
	_, ok := m.started(container.Container{Host: "nas", ID: "new000000000"})
	assert.False(t, ok)
	assert.True(t, m.shows(container.UpdateRecord{Host: "nas", NewID: "new000000000"}), "the record that arrives later is sent")
	assert.False(t, m.shows(container.UpdateRecord{Host: "nas", NewID: "elsewhere000"}))
}
