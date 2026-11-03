package docker

import (
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	docker_types "github.com/moby/moby/api/types/container"
	"github.com/stretchr/testify/assert"
)

func TestHealthProbesReadsStateHealthLog(t *testing.T) {
	start := time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	state := &docker_types.State{Health: &docker_types.Health{
		Status: docker_types.Unhealthy,
		Log: []*docker_types.HealthcheckResult{
			{Start: start, End: start.Add(time.Second), ExitCode: 0, Output: "ok"},
			nil,
			{Start: start.Add(30 * time.Second), End: start.Add(31 * time.Second), ExitCode: 1, Output: "connection refused"},
		},
	}}

	assert.Equal(t, []container.HealthProbe{
		{Start: start, End: start.Add(time.Second), ExitCode: 0, Output: "ok"},
		{Start: start.Add(30 * time.Second), End: start.Add(31 * time.Second), ExitCode: 1, Output: "connection refused"},
	}, healthProbes(state))
}

func TestHealthProbesEmptyWithoutHealthcheck(t *testing.T) {
	assert.Empty(t, healthProbes(nil))
	assert.Empty(t, healthProbes(&docker_types.State{}))
}
