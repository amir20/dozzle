package container

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealthStatusOf(t *testing.T) {
	tests := []struct {
		name   string
		event  ContainerEvent
		status string
		ok     bool
	}{
		{
			name:   "docker healthy action",
			event:  ContainerEvent{Name: "health_status: healthy"},
			status: "healthy",
			ok:     true,
		},
		{
			name:   "docker unhealthy action",
			event:  ContainerEvent{Name: "health_status: unhealthy"},
			status: "unhealthy",
			ok:     true,
		},
		{
			name: "podman attribute health_status",
			event: ContainerEvent{
				Name:            "health_status",
				ActorAttributes: map[string]string{"image": "nginx", "health_status": "healthy"},
			},
			status: "healthy",
			ok:     true,
		},
		{
			name: "podman attribute healthStatus",
			event: ContainerEvent{
				Name:            "health_status",
				ActorAttributes: map[string]string{"healthStatus": "unhealthy"},
			},
			status: "unhealthy",
			ok:     true,
		},
		{
			name: "podman attribute HealthStatus",
			event: ContainerEvent{
				Name:            "health_status",
				ActorAttributes: map[string]string{"HealthStatus": "starting"},
			},
			status: "starting",
			ok:     true,
		},
		{
			name:  "bare health_status without attributes",
			event: ContainerEvent{Name: "health_status"},
			ok:    false,
		},
		{
			name:  "non-health event",
			event: ContainerEvent{Name: "die"},
			ok:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, ok := HealthStatusOf(tt.event)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.status, status)
		})
	}
}
